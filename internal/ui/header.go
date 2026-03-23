package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	hWhite  = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	hOrange = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	hGreen  = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	hYellow = lipgloss.NewStyle().Foreground(colorYellow).Bold(true)
	hRed    = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	hGray   = lipgloss.NewStyle().Foreground(colorGray)
)

func (m Model) renderHeader() string {
	title    := hWhite.Render("  ⚡ DTrader 6")
	clock    := hWhite.Render(m.clockTime.UTC().Format("15:04:05 UTC"))
	balance  := hOrange.Render(fmt.Sprintf("💰 %s", m.balance))
	pnlDay   := hGreen.Render("↑ +$0.17 (+0.67%) 24h")
	pnlTotal := hGreen.Render("↑ +$2.43 (+10.6%) total")
	serv     := renderServ(m.connected, m.servMs)
	exch     := renderExch(m.exchCurMs, m.exchEmaMs)
	settings := hOrange.Render("⚙")

	indicators := serv + hGray.Render("  ") + exch + hGray.Render("  ") + settings

	blocks := []string{title, clock, balance, pnlDay, pnlTotal, indicators}
	totalW := 0
	for _, b := range blocks {
		totalW += lipgloss.Width(b)
	}
	totalGap := m.width - totalW - 2
	if totalGap < len(blocks)-1 {
		totalGap = len(blocks) - 1
	}
	gap := strings.Repeat(" ", totalGap/(len(blocks)-1))

	return HeaderStyle.Width(m.width - 2).Render(strings.Join(blocks, gap))
}

func renderServ(connected bool, ms int64) string {
	if !connected {
		return hRed.Render("● SERV OFF")
	} else if ms < 100 {
		return hGreen.Render(fmt.Sprintf("● SERV %dms", ms))
	}
	return hYellow.Render(fmt.Sprintf("● SERV %dms", ms))
}

func renderExch(cur, ema int64) string {
	if cur == 0 {
		return hRed.Render("● EXCH OFF")
	} else if cur < 300 {
		return hGreen.Render(fmt.Sprintf("● EXCH %dms (avg %dms)", cur, ema))
	} else if cur < 1000 {
		return hYellow.Render(fmt.Sprintf("● EXCH %dms (avg %dms)", cur, ema))
	}
	return hRed.Render(fmt.Sprintf("● EXCH %dms SOS (avg %dms)", cur, ema))
}
