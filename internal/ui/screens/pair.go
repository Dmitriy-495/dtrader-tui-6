package screens

import (
	"fmt"
	"strings"
)

// =============================================================================
// PairModel — модель детального экрана пары
// =============================================================================
type PairModel struct {
	Symbol string
	Data   *PairData
}

func NewPair(symbol string) PairModel {
	return PairModel{
		Symbol: symbol,
		Data:   &PairData{Symbol: symbol},
	}
}

// Render — детальный экран пары
func (m PairModel) Render(w, h int) string {
	if m.Data == nil {
		return grayStyle.Render("нет данных по " + m.Symbol)
	}

	var sb strings.Builder

	p := m.Data
	title := fmt.Sprintf("  %s  %s", m.Symbol, p.Price)
	sb.WriteString(orangeStyle.Render(title) + "\n")
	sb.WriteString(sepStyle.Render(strings.Repeat("─", w-4)) + "\n\n")

	// Объёмы
	sb.WriteString(headerStyle.Render("  Объёмы") + "\n")
	sb.WriteString(fmt.Sprintf("  %s  %s\n",
		greenStyle.Render(fmt.Sprintf("Buy:  +%.0f", p.BuyVol)),
		redStyle.Render(fmt.Sprintf("Sell: -%.0f", p.SellVol)),
	))

	// LSR
	sb.WriteString("\n" + headerStyle.Render("  Long/Short Ratio") + "\n")
	lsrStr := fmt.Sprintf("%.2f", p.LSR)
	if p.LSR >= 1.0 {
		sb.WriteString("  " + greenStyle.Render("▲ "+lsrStr+" (лонги доминируют)") + "\n")
	} else {
		sb.WriteString("  " + redStyle.Render("▼ "+lsrStr+" (шорты доминируют)") + "\n")
	}

	// OI
	sb.WriteString("\n" + headerStyle.Render("  Open Interest USD") + "\n")
	sb.WriteString(fmt.Sprintf("  %s\n", orangeStyle.Render(fmt.Sprintf("$%.0f", p.OI))))

	return sb.String()
}
