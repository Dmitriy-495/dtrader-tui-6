package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Стили Dashboard
// =============================================================================
var (
	orange  = lipgloss.Color("214")
	green   = lipgloss.Color("82")
	red     = lipgloss.Color("196")
	gray    = lipgloss.Color("239")

	headerStyle = lipgloss.NewStyle().Foreground(orange).Bold(true)
	sepStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	orangeStyle = lipgloss.NewStyle().Foreground(orange).Bold(true)
	greenStyle  = lipgloss.NewStyle().Foreground(green).Bold(true)
	redStyle    = lipgloss.NewStyle().Foreground(red).Bold(true)
	grayStyle   = lipgloss.NewStyle().Foreground(gray)
)

// =============================================================================
// PairData — данные по одной торговой паре
// =============================================================================
type PairData struct {
	Symbol  string
	Price   string
	BuyVol  float64
	SellVol float64
	LSR     float64 // Long/Short Ratio
	OI      float64 // Open Interest USD
}

// =============================================================================
// DashboardModel — модель экрана Dashboard
// =============================================================================
type DashboardModel struct {
	Pairs   map[string]*PairData
	Symbols []string
}

func NewDashboard() DashboardModel {
	return DashboardModel{
		Pairs:   make(map[string]*PairData),
		Symbols: []string{},
	}
}

// EnsurePair — добавляет пару если её нет
func (m *DashboardModel) EnsurePair(symbol string) {
	if _, ok := m.Pairs[symbol]; !ok {
		m.Pairs[symbol] = &PairData{Symbol: symbol}
		m.Symbols = append(m.Symbols, symbol)
	}
}

// Render — рендерит таблицу пар
func (m DashboardModel) Render(w, h int) string {
	var sb strings.Builder

	// Шапка таблицы
	header := fmt.Sprintf("  %-12s %-14s %-12s %-12s %-8s %-12s",
		"Пара", "Цена", "Buy Vol", "Sell Vol", "LSR", "OI USD",
	)
	sb.WriteString(headerStyle.Render(header) + "\n")
	sb.WriteString(sepStyle.Render(strings.Repeat("─", w-4)) + "\n")

	if len(m.Symbols) == 0 {
		sb.WriteString("\n  " + grayStyle.Render("Ожидание данных..."))
		return sb.String()
	}

	for _, sym := range m.Symbols {
		p, ok := m.Pairs[sym]
		if !ok {
			continue
		}
		sb.WriteString(renderRow(p) + "\n")
	}

	return sb.String()
}

// renderRow — одна строка таблицы
func renderRow(p *PairData) string {
	symbol  := orangeStyle.Render(fmt.Sprintf("  %-12s", p.Symbol))
	price   := orangeStyle.Render(fmt.Sprintf("%-14s", p.Price))
	buyVol  := greenStyle.Render(fmt.Sprintf("%-12s", fmt.Sprintf("+%.0f", p.BuyVol)))
	sellVol := redStyle.Render(fmt.Sprintf("%-12s", fmt.Sprintf("-%.0f", p.SellVol)))

	var lsr string
	if p.LSR >= 1.0 {
		lsr = greenStyle.Render(fmt.Sprintf("%-8.2f", p.LSR))
	} else {
		lsr = redStyle.Render(fmt.Sprintf("%-8.2f", p.LSR))
	}

	oi := orangeStyle.Render(fmt.Sprintf("%-12.0f", p.OI))
	return symbol + price + buyVol + sellVol + lsr + oi
}
