// Файл rightbar.go — правая панель главного лайаута: Logs (последние
// сообщения о состоянии приложения — подключения, ошибки, реконнекты)
// сверху, Positions (открытые позиции аккаунта) снизу.
//
// Ширина панели и пропорция Logs/Positions по высоте — не константы
// здесь, а настройки лайаута (см. settings.go: LayoutSettings,
// решение из чата "вынести все пропорции панелей в настройки").
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Dmitriy-495/dtrader-tui-6/internal/indicators"
)

// maxLogLines — сколько последних строк лога хранить и показывать.
// Не безграничный буфer — иначе при долгой работе (часы, как
// обсуждали в чате про read timeout) память росла бы бесконечно, а
// показать в ограниченной высоте панели больше нескольких десятков
// строк всё равно невозможно без собственной прокрутки rightbar
// (которой пока нет — см. TODO ниже).
const maxLogLines = 100

var (
	// rightbarBorderStyle — единый стиль рамки со всеми остальными
	// блоками лайаута (RoundedBorder + colorBorder, решение из чата:
	// "рамку logs/position сделать в одном стиле с другими" — было
	// NormalBorder приглушённого цвета, выбивалось из общего вида).
	rightbarBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBorder).
				Padding(0, 1)

	rightbarSectionTitleStyle = lipgloss.NewStyle().
					Foreground(colorText).
					Bold(true)
)

// LogEntry — одна строка лога с уровнем важности (для цвета) и
// временем — тот же принцип, что statusBadge в symbol.go использует
// для WS-статуса, только здесь произвольные текстовые события, а не
// фиксированный набор состояний соединения.
type LogEntry struct {
	Time  string // предвычисленная строка времени (не time.Time) — упрощает тесты и сравнение содержимого
	Text  string
	Level LogLevel
}

type LogLevel int

const (
	LogInfo LogLevel = iota
	LogWarn
	LogError
)

func (l LogLevel) color() lipgloss.Color {
	switch l {
	case LogWarn:
		return colorWarn
	case LogError:
		return colorSOS
	default:
		return colorMuted
	}
}

// renderPositionsBlock — заголовок POSITIONS + список позиций, с
// тонкой верхней границей-разделителем над заголовком (решение из
// чата: "останется отдельным, всегда видимым блоком снизу, отделённый
// от логов тонким верхним бордером"). Возвращает готовый текст без
// внешней рамки rightbar — та накладывается в App.View() поверх
// [logsViewport, этот блок].
// renderPositionsBlock — заголовок POSITIONS + список позиций, с
// тонкой верхней границей-разделителем над заголовком (решение из
// чата: "останется отдельным, всегда видимым блоком снизу, отделённый
// от логов тонким верхним бордером"). height — ФИКСИРОВАННАЯ высота
// блока (решение из чата: "фиксированные 40% высоты rightbar всегда",
// не зависит от реального числа позиций) — контент дополняется
// пустыми строками, если короче, либо обрезается (с пометкой
// "ещё N..."), если не помещается. Возвращает готовый текст без
// внешней рамки rightbar — та накладывается в App.View().
func renderPositionsBlock(positions []indicators.Position, width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	var b strings.Builder
	b.WriteString(dashboardDividerStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n")
	b.WriteString(rightbarSectionTitleStyle.Render("POSITIONS"))
	b.WriteString("\n")
	b.WriteString(renderPositions(positions))

	full := b.String()
	lines := strings.Split(full, "\n")
	if len(lines) > height {
		// Контент не помещается в фиксированную высоту — обрезаем и
		// явно отмечаем, что часть скрыта (решение из чата: 40% —
		// это ВСЕГДА фиксированная высота, не резиновая под контент,
		// значит при большом числе позиций что-то неизбежно не
		// уместится; лучше явная пометка, чем молчаливая обрезка).
		visible := lines[:height-1]
		hidden := len(lines) - (height - 1)
		return strings.Join(visible, "\n") + "\n" + mutedStyle.Render(fmt.Sprintf("... ещё %d строк", hidden))
	}
	return lipgloss.NewStyle().Height(height).Render(full)
}

// renderLogsContent — заголовок LOGS + сами записи, единый текст для
// viewport (App.rightbarVP) — тот же принцип, что renderPositionsBlock,
// заголовок остаётся частью прокручиваемого контента.
func renderLogsContent(logs []LogEntry) string {
	var b strings.Builder
	b.WriteString(rightbarSectionTitleStyle.Render("LOGS"))
	b.WriteString("\n")
	b.WriteString(renderLogs(logs))
	return b.String()
}

// renderLogs показывает последние строки лога, самые свежие снизу
// (тот же порядок чтения, что в обычном лог-файле/консоли — "новое
// внизу"), каждая раскрашена по уровню важности.
func renderLogs(logs []LogEntry) string {
	if len(logs) == 0 {
		return mutedStyle.Render("(пока пусто)")
	}
	var b strings.Builder
	for i, e := range logs {
		color := e.Level.color()
		b.WriteString(lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("%s %s", e.Time, e.Text)))
		if i < len(logs)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderPositions показывает список открытых позиций. Пустой список —
// валидное состояние (позиций нет), не путать с nil/ещё-не-пришедшими
// данными — TUI не различает эти два случая на уровне rightbar,
// потому что system-канал либо пришёл целиком (с actual []Position,
// который может быть пустым), либо не пришёл вовсе (тогда рендерится
// плейсхолдер уровнем выше, в app.go, а не здесь).
func renderPositions(positions []indicators.Position) string {
	if len(positions) == 0 {
		return mutedStyle.Render("нет открытых позиций")
	}
	var b strings.Builder
	for i, p := range positions {
		direction := "LONG"
		color := colorOK
		if !p.IsLong() {
			direction = "SHORT"
			color = colorSOS
		}
		pnl := p.PnL()
		pnlColor := colorOK
		pnlSign := "+"
		if pnl < 0 {
			pnlColor = colorSOS
			pnlSign = ""
		}
		b.WriteString(fmt.Sprintf(
			"%s %s\n%s %s%s",
			dataStyle.Render(p.Contract),
			lipgloss.NewStyle().Foreground(color).Render(direction),
			mutedStyle.Render("PnL:"),
			lipgloss.NewStyle().Foreground(pnlColor).Render(pnlSign+formatNumber(pnl)),
			mutedStyle.Render(" USDT"),
		))
		if i < len(positions)-1 {
			b.WriteString("\n\n")
		}
	}
	return b.String()
}
