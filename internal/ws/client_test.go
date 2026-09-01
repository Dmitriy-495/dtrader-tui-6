package ws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// newTestServer поднимает реальный HTTP+WS сервер на localhost через
// httptest — не мок, настоящее сетевое соединение через TCP, так что
// тест проверяет реальное поведение net.Conn (включая SetReadDeadline),
// а не имитацию через каналы в памяти, которая не воспроизвела бы
// собственно сетевые таймауты.
func newTestServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		handler(conn)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestClient_ReadTimeout_TriggersReconnectOnSilentConnection(t *testing.T) {
	// Сервер принимает соединение и намеренно ничего не шлёт —
	// имитация "тихого" обрыва (сеть перестала отвечать, но TCP
	// формально ещё открыт), см. обсуждение в чате: "терминал
	// подвисал" на нестабильной (мобильной) связи.
	var connectCount atomic.Int32
	srv := newTestServer(t, func(conn *websocket.Conn) {
		connectCount.Add(1)
		// Держим соединение открытым, ничего не шлём и не закрываем
		// сами — ждём, пока клиент сам решит, что пора реконнектиться
		// (через readTimeout), либо пока тест не завершится.
		<-make(chan struct{})
	})

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	client := New(wsURL, "any-key")
	client.readTimeout = 200 * time.Millisecond // короткий, для быстрого теста

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go client.Run(ctx)

	// Собираем статусы, ожидая увидеть StatusReconnecting — если бы
	// readTimeout не сработал, клиент завис бы в conn.ReadMessage()
	// без дедлайна и статус не изменился бы за отведённое время.
	seenReconnecting := false
	timeout := time.After(5 * time.Second)
loop:
	for {
		select {
		case s := <-client.Status:
			if s == StatusReconnecting {
				seenReconnecting = true
				break loop
			}
		case <-timeout:
			break loop
		}
	}

	if !seenReconnecting {
		t.Fatal("не увидели StatusReconnecting за 5s — read timeout не сработал на тихом соединении")
	}

	// Даём время на реальную попытку повторного подключения
	// (reconnectInterval=3s в клиенте) — connectCount должен вырасти
	// без дополнительных действий с нашей стороны.
	time.Sleep(3500 * time.Millisecond)
	if connectCount.Load() < 2 {
		t.Errorf("ожидался минимум повторный коннект после read timeout, реальных подключений: %d", connectCount.Load())
	}
}

func TestClient_NormalMessages_NotAffectedByReadTimeout(t *testing.T) {
	// Контрольный тест: read timeout не должен мешать нормальному
	// потоку сообщений — дедлайн переустанавливается на каждой
	// итерации, так что регулярно приходящие сообщения держат
	// соединение живым сколь угодно долго.
	srv := newTestServer(t, func(conn *websocket.Conn) {
		for i := 0; i < 5; i++ {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"channel":"system","symbol":"","data":{}}`)); err != nil {
				return
			}
			time.Sleep(50 * time.Millisecond) // короче readTimeout — соединение не должно рваться
		}
		<-make(chan struct{})
	})

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	client := New(wsURL, "any-key")
	client.readTimeout = 200 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go client.Run(ctx)

	received := 0
	timeout := time.After(1 * time.Second)
	for received < 5 {
		select {
		case <-client.Messages:
			received++
		case <-timeout:
			t.Fatalf("получено только %d/5 сообщений за 1s — read timeout мог ложно оборвать нормально работающее соединение", received)
		}
	}
}
