// Пакет ws — клиент ws-server для TUI.
//
// Симметричен серверной стороне (ws-server/internal/hub/hub.go):
// то же самое поле-в-поле сообщение {"channel","symbol","data"},
// та же библиотека (gorilla/websocket), тот же заголовок авторизации
// X-API-Key (см. ws-server/internal/handler/ws.go).
package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// Message — входящее сообщение от ws-server. Data оставлен как
// json.RawMessage, а не разобран сразу здесь: пакет ws не должен
// знать структуру каждого канала (system/indicators/trades/...) —
// это дело вызывающего кода (модели bubbletea), тот же принцип,
// что ws-server применяет к Pressure/Positions у себя.
type Message struct {
	Channel string          `json:"channel"`
	Symbol  string          `json:"symbol"`
	Data    json.RawMessage `json:"data"`
}

// Client — клиент ws-server с автопереподключением.
//
// reconnectInterval — пауза перед повторной попыткой подключения при
// разрыве соединения. Захардкожена константой ниже, а не вынесена в
// Config: в отличие от bot (где reconnect_interval — настройка
// эксплуатации, задаваемая на VDS через config.yaml), TUI — клиентское
// приложение конечного пользователя, лишний параметр в .env добавлял
// бы конфигурационный шум без реальной пользы на этом этапе (MVP).
// Если понадобится — поднять до Config так же просто, как в bot.
type Client struct {
	url    string
	apiKey string

	// readTimeout — см. подробное объяснение у места использования
	// (SetReadDeadline в connectAndRead). Поле, а не глобальная
	// константа — тестам нужно короткое значение, чтобы проверять
	// реальное срабатывание таймаута без ожидания настоящих 30s
	// (см. internal/ws/client_test.go).
	readTimeout time.Duration

	// Messages — канал, куда клиент публикует каждое успешно
	// разобранное сообщение от сервера. Небуферизован специально:
	// вызывающий код (bubbletea Update-цикл через tea.Cmd) должен
	// явно вычитывать каждое сообщение, а не полагаться на то, что
	// клиент будет копить их где-то за его спиной.
	Messages chan Message

	// Status — канал состояний соединения, для отображения в TUI
	// (например строка "● connected" / "○ reconnecting..." в шапке).
	Status chan Status
}

// Status — состояние соединения, публикуется в Client.Status при
// каждом переходе (не при каждой попытке — только при смене состояния),
// чтобы TUI не перерисовывал индикатор чаще, чем он реально меняется.
type Status int

const (
	StatusConnecting Status = iota
	StatusConnected
	StatusReconnecting
)

func (s Status) String() string {
	switch s {
	case StatusConnected:
		return "connected"
	case StatusReconnecting:
		return "reconnecting"
	default:
		return "connecting"
	}
}

const reconnectInterval = 3 * time.Second

// defaultReadTimeout — значение readTimeout по умолчанию для New().
// 30s: заметно больше самого редкого регулярного потока (system-
// heartbeat от ws-server раз в ~10s, см. RunSystem в ws-server), чтобы
// не считать живое, но временно тихое соединение мёртвым, и заметно
// меньше, чем "минуты", которые взяла бы себе ОС по умолчанию без
// явного дедлайна.
const defaultReadTimeout = 30 * time.Second

// New создаёт клиент. url — полный адрес, например ws://1.2.3.4:9000/ws
// (см. config.Config.WSServerURL).
func New(url, apiKey string) *Client {
	return &Client{
		url:         url,
		apiKey:      apiKey,
		readTimeout: defaultReadTimeout,
		Messages:    make(chan Message),
		Status:      make(chan Status, 1),
	}
}

// Run держит соединение с ws-server, переподключаясь при разрыве, пока
// ctx не отменён. Блокирующий вызов — предназначен для запуска в
// отдельной горутине (см. cmd/main.go).
//
// Как и в bot/cmd/main.go (цикл реконнекта wsClient.Connect), явной
// верхней границы числа попыток нет — TUI должен пытаться переподключиться,
// пока пользователь сам не выйдет (ctx.Done()) или процесс не убьют:
// это интерактивный клиент, а не одноразовая задача, задача "сдаться после
// N попыток" была бы хуже, чем просто продолжать пытаться на разумном
// интервале.
func (c *Client) Run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		c.sendStatus(StatusConnecting)
		if err := c.connectAndRead(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("⚠️ ws: соединение прервано: %v — повтор через %s", err, reconnectInterval)
		}

		c.sendStatus(StatusReconnecting)
		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectInterval):
		}
	}
}

// connectAndRead устанавливает одно соединение и читает сообщения из
// него, пока соединение живо или ctx не отменён. Возвращает ошибку,
// если соединение оборвалось не из-за отмены ctx — вызывающий код
// (Run) решает, что делать дальше (реконнект).
func (c *Client) connectAndRead(ctx context.Context) error {
	header := http.Header{}
	header.Set("X-API-Key", c.apiKey)

	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, resp, err := websocket.DefaultDialer.DialContext(dialCtx, c.url, header)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			// Неверный WS_API_KEY — реконнект бессмысленным не станет
			// (тот же неверный ключ будет отклонён снова и снова), но
			// TUI всё равно не должен падать намертво: пользователь
			// может поправить .env и перезапустить процесс, а до тех
			// пор пусть в статусной строке будет видно "reconnecting",
			// а не краш без объяснений.
			return fmt.Errorf("сервер отклонил ключ (401 Unauthorized)")
		}
		return fmt.Errorf("не удалось подключиться: %w", err)
	}
	defer conn.Close()

	log.Printf("✅ ws: подключено к %s", c.url)
	c.sendStatus(StatusConnected)

	// Горутина закрытия по ctx: gorilla/websocket не понимает context
	// напрямую при чтении (ReadMessage блокирующий), поэтому закрываем
	// соединение вручную при отмене ctx — это разблокирует ReadMessage
	// в основном цикле ниже с ошибкой закрытого соединения.
	closed := make(chan struct{})
	defer close(closed)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-closed:
		}
	}()

	for {
		// readTimeout: без явного дедлайна conn.ReadMessage() блокируется
		// неопределённо долго, если TCP-соединение "подвисло" молча
		// (сеть перестала отвечать, но не прислала явный close-фрейм —
		// частый случай на нестабильной связи, например мобильном
		// интернете). ОС в итоге сама вернёт ошибку по своему таймауту,
		// но это может занять заметно больше времени, чем нужно
		// интерактивному TUI, где пользователь видит "подвисший" экран
		// без объяснений. Дедлайн переустанавливается на каждой
		// итерации — отсчёт идёт от последнего реально полученного
		// сообщения, а не от момента установления соединения.
		//
		// readTimeout=30s: заметно больше самого редкого регулярного
		// потока (system-heartbeat от ws-server раз в ~10s, см.
		// RunSystem/pollSymbols в ws-server), чтобы не считать живое,
		// но временно тихое соединение мёртвым, и заметно меньше, чем
		// "минуты", которые взяла бы себе ОС по умолчанию.
		if err := conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
			return fmt.Errorf("не удалось установить read deadline: %w", err)
		}

		_, raw, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return nil // штатное закрытие по отмене ctx, не ошибка
			}
			return fmt.Errorf("ошибка чтения: %w", err)
		}

		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("⚠️ ws: не удалось разобрать сообщение: %v", err)
			continue
		}

		select {
		case c.Messages <- msg:
		case <-ctx.Done():
			return nil
		}
	}
}

// sendStatus публикует новый статус, не блокируясь, если получатель
// ещё не читает канал (буфер 1) — заменяет предыдущее непрочитанное
// значение, чтобы не копить устаревшие статусы, актуален только
// последний.
func (c *Client) sendStatus(s Status) {
	select {
	case c.Status <- s:
	default:
		select {
		case <-c.Status:
		default:
		}
		c.Status <- s
	}
}
