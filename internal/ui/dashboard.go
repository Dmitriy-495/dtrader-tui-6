package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PairData — данные по одной торговой паре
type PairData struct {
	Symbol  string
	Price   string
	BuyVol  float64
	SellVol float64
	LSR     float64 // Long/Short Ratio
	OI      float64 // Open Interest USD
}

var (
	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(colorOrange).
				Bold(true)

	tableSepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))
)

// renderDashboard — сводная таблица по всем парам
func (m Model) renderDashboard() string {
	var sb strings.Builder

	// Шапка таблицы
	header := fmt.Sprintf("  %-12s %-14s %-12s %-12s %-8s %-12s",
		"Пара", "Цена", "Buy Vol", "Sell Vol", "LSR", "OI USD",
	)
	sb.WriteString(tableHeaderStyle.Render(header) + "\n")
	sb.WriteString(tableSepStyle.Render(strings.Repeat("─", 74)) + "\n")

	// Строки по парам
	if len(m.symbols) == 0 {
		sb.WriteString("\n  " + GrayStyle.Render("Данные загружаются..."))
	} else {
		for _, sym := range m.symbols {
			if p, ok := m.pairs[sym]; ok {
				sb.WriteString(renderPairRow(p) + "\n")
			}
		}
	}

	return sb.String()
}

// renderPairRow — одна строка таблицы
func renderPairRow(p *PairData) string {
	symbol  := OrangeStyle.Render(fmt.Sprintf("  %-12s", p.Symbol))
	price   := OrangeStyle.Render(fmt.Sprintf("%-14s", p.Price))
	buyVol  := GreenStyle.Render(fmt.Sprintf("%-12s", fmt.Sprintf("+%.0f", p.BuyVol)))
	sellVol := RedStyle.Render(fmt.Sprintf("%-12s", fmt.Sprintf("-%.0f", p.SellVol)))

	var lsr string
	if p.LSR >= 1.0 {
		lsr = GreenStyle.Render(fmt.Sprintf("%-8.2f", p.LSR))
	} else {
		lsr = RedStyle.Render(fmt.Sprintf("%-8.2f", p.LSR))
	}

	oi := OrangeStyle.Render(fmt.Sprintf("%-12.0f", p.OI))

	return symbol + price + buyVol + sellVol + lsr + oi
}

// renderPairScreen — детальный экран по одной паре (заглушка)
func (m Model) renderPairScreen(symbol string) string {
	p, ok := m.pairs[symbol]
	if !ok {
		return GrayStyle.Render("нет данных по " + symbol)
	}
	var sb strings.Builder
	sb.WriteString(OrangeStyle.Render("📊 "+symbol) + "\n\n")
	sb.WriteString(OrangeStyle.Render("Цена:    ") + p.Price + "\n")
	sb.WriteString(GreenStyle.Render(fmt.Sprintf("Buy Vol: +%.0f\n", p.BuyVol)))
	sb.WriteString(RedStyle.Render(fmt.Sprintf("Sel Vol: -%.0f\n", p.SellVol)))
	if p.LSR >= 1.0 {
		sb.WriteString(GreenStyle.Render(fmt.Sprintf("LSR:     %.2f\n", p.LSR)))
	} else {
		sb.WriteString(RedStyle.Render(fmt.Sprintf("LSR:     %.2f\n", p.LSR)))
	}
	sb.WriteString(OrangeStyle.Render(fmt.Sprintf("OI USD:  %.0f\n", p.OI)))
	return sb.String()
}
