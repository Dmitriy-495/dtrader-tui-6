// Файл dashboard.go — вкладка №0 (Dashboard), общий обзор всех
// торгуемых символов. Решение из чата: разбивка на блоки по символу,
// каждый блок — заголовок (символ + midprice + двусторонняя шкала
// pressure) и три строки ниже, по одной на таймфрейм (1m/8m/24m) с
// TREND и VOL Δ. PRESSURE не привязан к ТФ (см. indicators.Pressure —
// одно значение на весь символ, не map по ТФ, как Trend/Volume),
// поэтому показан один раз в заголовке блока, не дублируется в
// строках ТФ.
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Dmitriy-495/dtrader-tui-6/internal/indicators"
)

var (
	dashboardSymbolStyle  = lipgloss.NewStyle().Foreground(colorText).Bold(true)
	dashboardTFLabelStyle = lipgloss.NewStyle().Foreground(colorMuted).Width(5)
	dashboardDividerStyle = lipgloss.NewStyle().Foreground(colorMuted)
)

// renderDashboard рисует блоки по всем символам, используя уже
// накопленное состояние их вкладок (symbolModels) — Dashboard не
// хранит собственную копию snapshot/orderbook, а читает их из тех же
// symbol.Model, что показываются на отдельных вкладках, одного
// источника истины для одних и тех же данных.
//
// Возвращает только КОНТЕНТ (без внешней рамки) — рамку накладывает
// вызывающий код (App.renderContent) поверх viewport.View(), тот же
// принцип, что уже применяется для вкладок символов (symbol.Model.View
// оборачивает m.vp.View() в borderStyle снаружи, не сама generatorBody).
// width — ширина именно контента (уже без учёта рамки/паддинга,
// которые добавит внешний borderStyle).
func renderDashboard(sys *indicators.SystemMsg, models map[string]Model, width int) string {
	if width < 1 {
		width = 1
	}
	if sys == nil || len(sys.Symbols) == 0 {
		return mutedStyle.Render("ожидание списка торгуемых пар (system)...")
	}

	// Сортируем для стабильного порядка блоков между кадрами —
	// sys.Symbols приходит в том порядке, что задал сервер (см.
	// tabLabels — там порядок сервера сохраняется намеренно для
	// вкладок), но для обзора стабильный алфавитный порядок читать
	// легче, чем порядок, который может отличаться от прихода к
	// приходу.
	symbols := append([]string(nil), sys.Symbols...)
	sort.Strings(symbols)

	var b strings.Builder
	for i, symbol := range symbols {
		b.WriteString(renderDashboardBlock(symbol, models[symbol], width))
		if i < len(symbols)-1 {
			b.WriteString("\n")
			b.WriteString(dashboardDividerStyle.Render(strings.Repeat("─", width)))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderDashboardBlock строит один блок символа: заголовок (символ +
// midprice + pressure) и три строки ТФ ниже. m — вкладка символа
// (может быть с nil snapshot/orderbook, если данные ещё не пришли —
// валидное состояние сразу после появления новой вкладки, не ошибка).
func renderDashboardBlock(symbol string, m Model, width int) string {
	var b strings.Builder

	b.WriteString(renderDashboardHeader(symbol, m))
	b.WriteString("\n")

	if m.snapshot == nil {
		b.WriteString(mutedStyle.Render("  ожидание данных indicators..."))
		return b.String()
	}

	// Пустая строка между заголовком и первым ТФ, и между каждым ТФ —
	// решение из чата: "отдели строки заголовков и таймфреймов
	// пустыми строками", тот же принцип "воздуха" между показателями,
	// что уже применён на вкладке одного символа.
	for _, tf := range indicators.Timeframes {
		b.WriteString("\n")
		b.WriteString(renderDashboardTFRow(tf, m.snapshot))
	}

	return b.String()
}

// renderDashboardHeader строит заголовочную строку блока: имя символа,
// midprice (из orderbook той же вкладки) и двусторонняя шкала pressure
// (переиспользует pressureBar — общую с renderPressureBlock в
// symbol.go функцию расчёта шкалы, решение из чата: "показывать в
// виде индикатора в стиле других вкладок").
func renderDashboardHeader(symbol string, m Model) string {
	midText := mutedStyle.Render("mid: n/a")
	if m.orderbook != nil {
		if mid, ok := m.orderbook.MidPrice(); ok {
			midText = mutedStyle.Render("mid: ") + dataStyle.Render(formatNumber(mid))
		}
	}

	pressureText := mutedStyle.Render("pressure: n/a")
	if m.snapshot != nil {
		bar, pct := pressureBar(m.snapshot.Pressure)
		pressureText = fmt.Sprintf("%s %s %s", mutedStyle.Render("pressure:"), bar, fmt.Sprintf("%+.1f%%", pct))
	}

	return fmt.Sprintf(
		"%s   %s   %s",
		dashboardSymbolStyle.Render(symbol),
		midText,
		pressureText,
	)
}

// renderDashboardTFRow строит одну строку таймфрейма: лейбл ТФ,
// направление тренда, шкала дельты объёма — переиспользует
// directionText/volumeDeltaBar из symbol.go, те же функции, что
// рисуют ТФ-блоки на вкладке одного символа, чтобы вид совпадал
// между Dashboard и отдельными вкладками.
func renderDashboardTFRow(tf string, snap *indicators.Snapshot) string {
	t, hasTrend := snap.Trend[tf]
	v, hasVolume := snap.Volume[tf]

	trendCell := mutedStyle.Render("n/a")
	if hasTrend {
		trendCell = directionText(t.Direction)
	}

	volCell := mutedStyle.Render("n/a")
	if hasVolume {
		volCell = volumeDeltaBar(v.BuyVol, v.SellVol, v.Spike)
	}

	return fmt.Sprintf(
		"  %s%s  %s",
		dashboardTFLabelStyle.Render(tf),
		trendCell,
		volCell,
	)
}
