// Рендер стакана (orderbook) в стиле, приближенном к биржевому —
// решение из чата: "20 строк по вертикали", "10 asks сверху + 10
// bids снизу (классический вид биржи)", цена+объём (без кумулятива),
// глубина как цветная полоса-фон позади чисел (образец — скриншот
// Binance-подобного стакана, приложенный в чате).
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Dmitriy-495/dtrader-tui-6/internal/indicators"
)

// orderbookLevelsPerSide — количество уровней с каждой стороны стакана
// (решение из чата: "10 asks сверху + 10 bids снизу").
const orderbookLevelsPerSide = 10

// orderbookRowWidth — ширина одной строки стакана в символах (цена +
// объём + отступы), под неё же нормализуется ширина полосы глубины и
// центрируются заголовок ORDERBOOK/шапка колонок. Должна точно
// совпадать с реальной длиной строки, которую строит renderOrderbookRow
// (orderbookPriceWidth + "  " + orderbookVolWidth = 12+2+14 = 28) —
// раньше было 26 (расхождение в 2 символа), из-за чего заголовок и
// шапка колонок центрировались не по факту видимой ширины данных.
const orderbookRowWidth = orderbookPriceWidth + 2 + orderbookVolWidth

// orderbookPriceWidth/orderbookVolWidth — ширины подколонок цены и
// объёма внутри строки стакана.
const (
	orderbookPriceWidth = 12
	orderbookVolWidth   = 14
)

var (
	// askBg/bidBg — приглушённые фоновые цвета полосы глубины,
	// сильно затемнённые относительно "чистых" colorSOS/colorOK
	// (196/82), чтобы текст поверх оставался читаемым — на
	// скриншоте-образце заливка тоже мягкая, не кричаще-яркая.
	askBgColor = lipgloss.Color("52") // приглушённый тёмно-красный
	bidBgColor = lipgloss.Color("22") // приглушённый тёмно-зелёный
	askFgColor = colorSOS
	bidFgColor = colorOK
)

// renderOrderbookColumn рисует полную колонку стакана: заголовок
// "ORDERBOOK" по центру, строку с названиями колонок (PRICE/SIZE),
// до orderbookLevelsPerSide строк asks (от дальней к best ask, сверху
// вниз — так лучшая цена продажи оказывается ближе к центру, как на
// бирже), строку спреда с текущей ценой, затем до
// orderbookLevelsPerSide строк bids (от best bid к дальней).
//
// ob может быть nil (ещё не пришло ни одного orderbook-сообщения) —
// тогда возвращает плейсхолдер того же типа, что остальные "ожидание
// данных" в этой вкладке (см. bestPricesText(nil)).
func renderOrderbookColumn(ob *indicators.OrderBook) string {
	if ob == nil {
		return mutedStyle.Render("ожидание данных orderbook...")
	}

	maxVol := maxOrderbookVolume(ob)

	var b strings.Builder

	b.WriteString(orderbookTitleStyle.Render("ORDERBOOK"))
	b.WriteString("\n")
	b.WriteString(renderOrderbookColumnHeader())
	b.WriteString("\n")

	// asks: биржевой порядок — дальняя цена сверху, best ask снизу
	// (ближе к спреду/центру) — берём последние N уровней среза и
	// проходим в обратном порядке, а не первые N просто по порядку.
	askLevels := ob.Asks
	if len(askLevels) > orderbookLevelsPerSide {
		askLevels = askLevels[:orderbookLevelsPerSide]
	}
	for i := len(askLevels) - 1; i >= 0; i-- {
		b.WriteString(renderOrderbookRow(askLevels[i], maxVol, askBgColor, askFgColor))
		b.WriteString("\n")
	}

	b.WriteString(renderSpreadRow(ob))
	b.WriteString("\n")

	bidLevels := ob.Bids
	if len(bidLevels) > orderbookLevelsPerSide {
		bidLevels = bidLevels[:orderbookLevelsPerSide]
	}
	for i, lvl := range bidLevels {
		b.WriteString(renderOrderbookRow(lvl, maxVol, bidBgColor, bidFgColor))
		if i < len(bidLevels)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// orderbookTitleStyle — заголовок "ORDERBOOK" по центру колонки,
// решение из чата: "шапка с наименованием колонок и заголовком
// ORDERBOOK (с выравниванием по центру)".
var orderbookTitleStyle = lipgloss.NewStyle().
	Foreground(colorText).
	Bold(true).
	Width(orderbookRowWidth).
	Align(lipgloss.Center)

// orderbookColumnHeaderStyle — стиль строки с названиями колонок
// (PRICE/SIZE), приглушённый — она вспомогательная, не должна
// перетягивать внимание с заголовка ORDERBOOK или самих чисел.
var orderbookColumnHeaderStyle = lipgloss.NewStyle().Foreground(colorMuted)

// renderOrderbookColumnHeader рисует строку "PRICE   SIZE" с теми же
// ширинами подколонок, что реальные строки стакана (orderbookPriceWidth/
// orderbookVolWidth) — так подписи оказываются точно над своими
// колонками чисел, не просто "где-то сверху".
func renderOrderbookColumnHeader() string {
	return orderbookColumnHeaderStyle.Render(fmt.Sprintf("%s  %s",
		padLeft("PRICE", orderbookPriceWidth),
		padLeft("SIZE", orderbookVolWidth),
	))
}

// maxOrderbookVolume находит наибольший объём среди отображаемых
// уровней (обеих сторон) — используется как база для нормализации
// ширины полосы глубины (решение из чата: полоса шире у большего
// объёма — тот же принцип, что на скриншоте-образце).
func maxOrderbookVolume(ob *indicators.OrderBook) float64 {
	var max float64
	check := func(levels []indicators.OrderBookLevel) {
		n := len(levels)
		if n > orderbookLevelsPerSide {
			n = orderbookLevelsPerSide
		}
		for _, lvl := range levels[:n] {
			if v := lvl.Size(); v > max {
				max = v
			}
		}
	}
	check(ob.Asks)
	check(ob.Bids)
	return max
}

// renderOrderbookRow рисует одну строку стакана: цена + объём, с
// полосой глубины позади (fg/bg цвета переданы вызывающим кодом —
// разные для ask/bid).
//
// Полоса глубины реализована как два стилизованных фрагмента разной
// ширины (закрашенный + обычный фон), а не через настоящий
// alpha-blend (терминал этого не умеет) — решение из чата: "цветная
// полоса-фон позади чисел".
func renderOrderbookRow(lvl indicators.OrderBookLevel, maxVol float64, bg, fg lipgloss.Color) string {
	price := lvl.Price()
	size := lvl.Size()

	text := fmt.Sprintf("%s  %s",
		padLeft(formatNumber(price), orderbookPriceWidth),
		padLeft(formatNumber(size), orderbookVolWidth),
	)
	// Строка короче orderbookRowWidth — дополняем пробелами справа,
	// чтобы полоса глубины могла занимать полную заданную долю
	// ширины независимо от длины конкретных чисел.
	if len(text) < orderbookRowWidth {
		text += strings.Repeat(" ", orderbookRowWidth-len(text))
	}

	filledWidth := 0
	if maxVol > 0 {
		filledWidth = int(size / maxVol * float64(orderbookRowWidth))
		if filledWidth > orderbookRowWidth {
			filledWidth = orderbookRowWidth
		}
	}

	runes := []rune(text)
	if filledWidth > len(runes) {
		filledWidth = len(runes)
	}

	filledPart := lipgloss.NewStyle().Background(bg).Foreground(fg).Render(string(runes[:filledWidth]))
	restPart := lipgloss.NewStyle().Foreground(fg).Render(string(runes[filledWidth:]))

	return filledPart + restPart
}

// renderSpreadRow показывает разницу best ask − best bid (спред)
// между блоками asks/bids — решение из чата: "спред между best bid
// и best ask посередине".
func renderSpreadRow(ob *indicators.OrderBook) string {
	bid, bidOK := ob.BestBid()
	ask, askOK := ob.BestAsk()
	if !bidOK || !askOK {
		return mutedStyle.Render(strings.Repeat("·", orderbookRowWidth))
	}
	spread := ask - bid
	var spreadPct float64
	if bid != 0 {
		spreadPct = spread / bid * 100
	}
	return titleStyle.Render(fmt.Sprintf("spread %s (%.3f%%)", formatNumber(spread), spreadPct))
}

// padLeft дополняет строку пробелами слева до нужной ширины (rune-safe
// для случаев вроде уже отформатированных чисел с пробелами-
// разделителями тысяч внутри — считаем длину в рунах, не байтах).
func padLeft(s string, width int) string {
	n := len([]rune(s))
	if n >= width {
		return s
	}
	return strings.Repeat(" ", width-n) + s
}
