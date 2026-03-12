package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Dmitriy-495/dtrader-tui-6/internal/config"
	"github.com/Dmitriy-495/dtrader-tui-6/internal/ui"
	"github.com/Dmitriy-495/dtrader-tui-6/internal/ws"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Ошибка конфига: %v", err)
	}

	fmt.Println("🚀 DTrader 6 TUI запускается...")

	// Создаём WS клиент
	wsClient := ws.New(cfg.WSServerURL, cfg.APIKey)

	// Запускаем WS в фоне
	go wsClient.Connect()

	// Запускаем TUI
	p := tea.NewProgram(
		ui.New(wsClient.MsgCh),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Ошибка TUI: %v\n", err)
		os.Exit(1)
	}
}
