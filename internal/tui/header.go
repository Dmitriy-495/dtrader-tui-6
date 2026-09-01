// Файл header.go — верхняя панель главного лайаута приложения:
// время, баланс, суммарный PnL по открытым позициям, статусы
// соединения (SERV — латентность до ws-server, EXCH — латентность
// ws-server до биржи). Формат и пороги — раздел 11 CHECKPOINT.md
// dtrader-6 (дизайн-система TUI), не придуманы заново.
package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/Dmitriy-495/dtrader-tui-6/internal/indicators"
)

// Пороги статусов — раздел 11 CHECKPOINT.md: SERV <100ms OK, ≥100ms
// WARNING; EXCH <300ms OK, 300-1000ms WARNING, ≥1000ms SOS. OFF —
// отдельное состояние (нет данных вообще), не то же самое, что
// "плохое значение" — используется, пока system-канал ещё не пришёл
// ни разу.
const (
	servWarnThresholdMs = 100

	// servStaleThresholdMs — если разница между "сейчас" и server_ts
	// превышает эту величину, значение считается протухшим/повреждённым
	// (например, некорректная метка времени), а не просто "очень
	// медленным" — показываем "n/a" вместо абсурдно большого числа
	// миллисекунд. system-канал в норме шлётся каждые ~10s (см.
	// ws-server RunSystem), так что разумный запас — на пару порядков
	// больше, не впритык к нормальному интервалу.
	servStaleThresholdMs = 60_000

	exchWarnThresholdMs = 300
	exchSOSThresholdMs  = 1000
)

var (
	headerBrandStyle = lipgloss.NewStyle().Foreground(colorBorder).Bold(true)
)

// renderHeader строит содержимое шапки. sys может быть nil (ещё не
// пришло ни одного system-сообщения) — тогда показываем плейсхолдеры
// вместо паники на nil-разыменовании, тот же принцип, что уже
// применяется к m.snapshot/m.orderbook в symbol.go.
//
// width — полная ширина терминала. ВАЖНО про lipgloss.Style.Width():
// она задаёт ОБЩУЮ внешнюю ширину блока включая padding (см.
// style.go: wrapAt := width - leftPadding - rightPadding — то есть
// сам lipgloss уже вычитает padding из переданного Width при переносе
// строк). Раньше здесь ошибочно вычитался padding ЕЩЁ РАЗ вручную
// (innerWidth := width - GetHorizontalFrameSize()), из-за чего
// реальная область текста получалась на 2×padding уже, чем нужно, и
// короткие строки переносились на вторую строку без явной причины
// (см. историю правки — тест TestRenderHeader_NilShowsWaiting поймал
// это на строке "ожидание данных system...", разорванной пополам).
func renderHeader(sys *indicators.SystemMsg, width int) string {
	left := headerBrandStyle.Render("⚡ DTrader 6") + "  " + mutedStyle.Render(time.Now().Format("15:04:05 MST"))

	var center, right string
	if sys == nil {
		center = mutedStyle.Render("ожидание данных system...")
	} else {
		center = fmt.Sprintf("%s  %s", balanceText(sys.Balance), pnlText(sys.TotalPnL()))
		right = fmt.Sprintf("%s  %s", servStatusText(sys.ServerTs), exchStatusText(sys.ExchangePing))
	}

	// Решение из чата: "блок с балансом и PnL разместить по центру
	// хедера" — раньше баланс/PnL были частью правого блока вместе со
	// статусами SERV/EXCH. Теперь три зоны: left (бренд+время), center
	// (баланс+PnL, реально по центру всей ширины), right (статусы).
	//
	// textWidth — значение, которое нужно передать в .Width() ниже,
	// чтобы ИТОГОВАЯ ширина блока с рамкой была равна width. Проверено
	// экспериментально (см. историю правки в чате — несколько неверных
	// попыток формулы): .Padding(0,2).Width(N) уже включает паддинг
	// внутрь N (даёт видимую ширину N, не N+4), а внешний
	// .Border(...) добавляет ровно 2 символа (по одному слева/справа).
	// Итог: N = width - 2 даёт финальную ширину == width.
	textWidth := width - 2

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	centerWidth := lipgloss.Width(center)

	// headerSafetyMargin — запас, используемый ТОЛЬКО при расчёте
	// зазоров (totalPad) вокруг center, не при определении итоговой
	// ширины рамки. Обнаруженная в чате причина нужды в запасе:
	// lipgloss.Width() и реальная визуальная ширина строки расходятся,
	// когда в тексте есть эмодзи (⚡💰↑●) — некоторые терминалы рисуют
	// их шире одной колонки, а библиотека считает по-другому. Без
	// этого запаса расчёт "площадь текста == textWidth впритык" мог
	// приводить к переносу строки внутри рамки (header становился 4
	// строки вместо 3).
	const headerSafetyMargin = 4

	// Центр центрируем не относительно left/right по отдельности, а
	// относительно ВСЕЙ ширины — leftPad/rightPad вокруг center
	// считаются так, чтобы center оказался в геометрической середине
	// строки, а left/right просто прижаты к своим краям поверх этого.
	totalPad := textWidth - headerSafetyMargin - leftWidth - rightWidth - centerWidth
	if totalPad < 0 {
		totalPad = 0
	}
	leftGap := totalPad / 2
	rightGap := totalPad - leftGap

	line := left + strings.Repeat(" ", leftGap) + center + strings.Repeat(" ", rightGap) + right

	// Паттерн, единый с symbol.go/rightbar.go: .Width()/.Padding()
	// применяются на ОТДЕЛЬНОМ внутреннем стиле контента, а не на
	// самом стиле с рамкой — вызов .Width() прямо на стиле с рамкой
	// оказался ненадёжным (в этой версии lipgloss Padding уже
	// учитывается внутри .Width(), а не добавляется сверх ннего, что
	// и вызвало баг "правый бордер сместился влево" — сначала пытались
	// решить точным подбором формулы вычитания, но раз в других местах
	// проекта уже есть работающий паттерн, надёжнее привести header к
	// нему, чем полагаться на арифметику конкретного нюанса lipgloss).
	content := lipgloss.NewStyle().Padding(0, 2).Width(textWidth).Render(line)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Render(content)
}

func balanceText(b indicators.Balance) string {
	total, err := strconv.ParseFloat(b.Total, 64)
	if err != nil {
		return mutedStyle.Render("баланс: n/a")
	}
	return fmt.Sprintf("%s %s USDT", mutedStyle.Render("💰"), dataStyle.Render(formatNumber(total)))
}

func pnlText(pnl float64) string {
	arrow, color := "↑", colorOK
	if pnl < 0 {
		arrow, color = "↓", colorSOS
	}
	// formatNumber сама добавляет знак минус для отрицательных, плюс
	// добавляем явно для положительных — иначе "1 234,00" не отличить
	// от отрицательного PnL на глаз без явного +/-.
	sign := "+"
	if pnl < 0 {
		sign = ""
	}
	return lipgloss.NewStyle().Foreground(color).Bold(true).Render(fmt.Sprintf("%s%s%s", arrow, sign, formatNumber(pnl)))
}

// servStatusText — латентность SERV (наш ws-server), считается на
// стороне TUI как разница между временем получения сообщения и
// serverTs внутри него — ws-server не присылает готовое значение
// SERV latency (в отличие от EXCH, которое он уже сам измеряет и
// присылает как exchange_ping), поэтому считаем сами.
func servStatusText(serverTs int64) string {
	latency := time.Now().UnixMilli() - serverTs
	if latency < 0 {
		latency = 0 // защита от рассинхронизации часов клиент/сервер
	}
	if latency >= servStaleThresholdMs {
		// Протухшее/повреждённое значение — не показываем абсурдно
		// большое число миллисекунд, это выглядит как баг, а не как
		// содержательная информация о реальной задержке.
		return lipgloss.NewStyle().Foreground(colorSOS).Render("●SERV n/a")
	}
	color := colorOK
	if latency >= servWarnThresholdMs {
		color = colorWarn
	}
	return lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("●SERV %dms", latency))
}

func exchStatusText(ping indicators.ExchangePing) string {
	color := colorOK
	switch {
	case ping.Current >= exchSOSThresholdMs:
		color = colorSOS
	case ping.Current >= exchWarnThresholdMs:
		color = colorWarn
	}
	return lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("●EXCH %dms", ping.Current))
}
