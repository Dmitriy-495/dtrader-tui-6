package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Dmitriy-495/dtrader-tui-6/internal/config"
	"github.com/Dmitriy-495/dtrader-tui-6/internal/news"
	"github.com/Dmitriy-495/dtrader-tui-6/internal/ui"
	"github.com/Dmitriy-495/dtrader-tui-6/internal/ws"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Ошибка конфига: %v", err)
	}

	fmt.Println("🚀 DTrader 6 TUI запускается...")

	// News client — тянет новости каждые 5 минут
	newsClient := news.New(cfg.CryptoPanicKey)
	newsClient.Start()

	// WS клиент
	wsClient := ws.New(cfg.WSServerURL, cfg.APIKey)
	go wsClient.Connect()

	// TUI
	p := tea.NewProgram(
		ui.New(wsClient.MsgCh, newsClient.UpdateCh),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Ошибка TUI: %v\n", err)
		os.Exit(1)
	}
}
