// MVP-шаг 2: bubbletea UI для одной вкладки символа.
//
// Символ пока захардкожен (symbolMVP ниже) — это осознанное сужение
// для отработки самой модели (см. решение в чате: сначала одна
// вкладка, потом главный экран + автовкладки по symbols из system).
// Шаг 1 (internal/ws.Client, чтение raw-сообщений) уже подтверждён
// на проде — см. README.md.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dmitriy-495/dtrader-tui-6/internal/config"
	"github.com/Dmitriy-495/dtrader-tui-6/internal/tui"
	"github.com/Dmitriy-495/dtrader-tui-6/internal/ws"
)

// symbolMVP — временный захардкоженный символ для шага 2. Заменится
// автовкладками по symbols из system-канала на следующем шаге.
const symbolMVP = "BTC_USDT"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Ошибка загрузки конфига: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	client := ws.New(cfg.WSServerURL, cfg.WSAPIKey)
	go client.Run(ctx)

	model := tui.New(symbolMVP, client)
	// tea.WithAltScreen: рисуем в отдельном полноэкранном буфере
	// терминала (как vim/htop), а не построчно в обычном выводе.
	// Без этой опции первый кадр может отрисоваться с шириной по
	// умолчанию ещё до прихода tea.WindowSizeMsg — на терминалах это
	// иногда выглядит как короткая "вспышка" неправильно
	// отформатированной рамки перед тем, как отрисуется настоящий
	// кадр. Alt-screen переключает буфер целиком один раз при
	// старте, так что этот промежуточный кадр не виден вообще —
	// пользователь видит только финальный чистый экран.
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Пробрасываем отмену ctx (Ctrl+C/SIGTERM) в остановку bubbletea —
	// иначе программа продолжила бы рисовать экран после того, как
	// WS-клиент уже завершился по сигналу.
	go func() {
		<-ctx.Done()
		p.Quit()
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Ошибка TUI: %v\n", err)
		os.Exit(1)
	}
}
