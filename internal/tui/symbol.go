// Пакет tui — bubbletea-модель TUI.
//
// Этот файл — первая, самая маленькая модель: одна вкладка одного
// символа. Задача — отработать саму механику (как Msg из ws.Client
// попадает в Update, как рисуется таблица T/V/P) на одном случае,
// прежде чем размножать это на несколько символов + главный экран
// (см. решение в чате: сначала одна вкладка, потом остальное).
package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dmitriy-495/dtrader-tui-6/internal/indicators"
	"github.com/Dmitriy-495/dtrader-tui-6/internal/ws"
)

// Цвета — раздел 11 CHECKPOINT.md dtrader-6 (дизайн-система TUI),
// значения взяты оттуда дословно, не подобраны на глаз.
var (
	colorBorder  = lipgloss.Color("214") // фирменный оранжевый
	colorOK      = lipgloss.Color("82")  // статус OK / direction up
	colorWarn    = lipgloss.Color("226") // статус WARNING
	colorSOS     = lipgloss.Color("196") // статус SOS/OFF / direction down
	colorText    = lipgloss.Color("255") // текст важный
	colorData    = lipgloss.Color("214") // текст данные
	colorMuted   = lipgloss.Color("239") // текст вспомогательный
	colorNeutral = colorMuted            // direction neutral — своего цвета в разделе 11 нет, берём приглушённый
)

var (
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Bold(true)

	mutedStyle = lipgloss.NewStyle().Foreground(colorMuted)
	dataStyle  = lipgloss.NewStyle().Foreground(colorData)
)

// wsMsg оборачивает ws.Message для передачи через tea.Cmd — bubbletea
// требует, чтобы всё, что попадает в Update, было тем же типом,
// который вернула соответствующая Cmd-функция.
type wsMsg ws.Message

// wsStatusMsg оборачивает ws.Status аналогично wsMsg.
type wsStatusMsg ws.Status

// Model — состояние экрана одного символа.
type Model struct {
	symbol string
	client *ws.Client

	status    ws.Status
	snapshot  *indicators.Snapshot  // nil, пока ничего не пришло по этому символу
	orderbook *indicators.OrderBook // nil, пока не пришло ни одного orderbook-сообщения для этого символа
	lastErr   string                // последняя ошибка разбора JSON, если была — не молчим о битых данных

	width, height int
}

// New создаёт модель для одного символа. client уже должен быть
// запущен (client.Run(ctx) в отдельной горутине) — Model только
// читает из client.Messages/client.Status, не управляет жизненным
// циклом соединения.
func New(symbol string, client *ws.Client) Model {
	return Model{
		symbol: symbol,
		client: client,
		status: ws.StatusConnecting,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(waitForMessage(m.client), waitForStatus(m.client))
}

// waitForMessage возвращает Cmd, блокирующийся на одном сообщении из
// client.Messages. bubbletea вызывает Cmd в своей горутине и присылает
// результат обратно в Update как обычный tea.Msg — так канал из
// внешнего мира (WS-клиент) превращается в поток bubbletea-сообщений,
// не блокируя основной Update/View цикл.
func waitForMessage(client *ws.Client) tea.Cmd {
	return func() tea.Msg {
		return wsMsg(<-client.Messages)
	}
}

func waitForStatus(client *ws.Client) tea.Cmd {
	return func() tea.Msg {
		return wsStatusMsg(<-client.Status)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
		return m, nil

	case wsStatusMsg:
		m.status = ws.Status(msg)
		// Продолжаем слушать следующий статус — иначе после первого
		// полученного статуса канал перестал бы читаться, и клиент
		// (see sendStatus в internal/ws/client.go) начал бы блокироваться
		// на непрочитанном канале с буфером 1.
		return m, waitForStatus(m.client)

	case wsMsg:
		cmd := waitForMessage(m.client) // сразу планируем ожидание следующего — та же причина, что и для статуса выше

		if msg.Symbol != m.symbol {
			// Не наш символ — игнорируем оба интересующих нас канала
			// (indicators и orderbook), но продолжаем слушать.
			return m, cmd
		}

		switch msg.Channel {
		case "indicators":
			var snap indicators.Snapshot
			if err := json.Unmarshal(msg.Data, &snap); err != nil {
				m.lastErr = err.Error()
				return m, cmd
			}
			m.snapshot = &snap
			m.lastErr = ""

		case "orderbook":
			var ob indicators.OrderBook
			if err := json.Unmarshal(msg.Data, &ob); err != nil {
				m.lastErr = err.Error()
				return m, cmd
			}
			m.orderbook = &ob
			m.lastErr = ""
		}

		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	var head strings.Builder

	head.WriteString(titleStyle.Render(fmt.Sprintf("  %s  ", m.symbol)))
	head.WriteString("  ")
	head.WriteString(bestPricesText(m.orderbook))
	head.WriteString("\n\n")

	if m.snapshot == nil {
		head.WriteString(mutedStyle.Render("ожидание данных indicators..."))
		if m.lastErr != "" {
			head.WriteString("\n")
			head.WriteString(lipgloss.NewStyle().Foreground(colorSOS).Render("ошибка разбора: " + m.lastErr))
		}
		return borderStyle.Render(head.String())
	}

	var right strings.Builder
	right.WriteString(renderPressureBlock(m.snapshot.Pressure))
	right.WriteString("\n")
	right.WriteString(blockDividerStyle.Render(strings.Repeat("─", timeframeBlockWidth)))
	right.WriteString("\n")
	right.WriteString(renderTimeframeBlocks(*m.snapshot))

	left := renderOrderbookColumn(m.orderbook)

	// Две колонки бок о бок: стакан (20 строк, приближенный к
	// биржевому виду) слева, ТФ-блоки справа — решение из чата
	// ("разделим карточку символа на две части по горизонтали").
	// lipgloss.JoinHorizontal выравнивает обе колонки по высоте сам,
	// не нужно вручную считать разницу строк между ними.
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, "   ", right.String())

	head.WriteString(body)

	if m.lastErr != "" {
		head.WriteString("\n")
		head.WriteString(lipgloss.NewStyle().Foreground(colorSOS).Render("ошибка разбора последнего сообщения: " + m.lastErr))
	}

	return borderStyle.Render(head.String())
}

// statusBadge — цветной индикатор состояния соединения, тот же
// набор цветов (OK/WARNING/SOS), что использует раздел 11 для
// SERV/EXCH статусов в header — здесь применяем его к статусу
// самого WS-соединения TUI.
// statusBadge — цветной индикатор состояния WS-соединения
// (connected/connecting/reconnecting). НЕ используется в этой вкладке
// (Model.View выше) — решение из чата: статус соединения общий на
// всё приложение (TUI↔ws-server), не свойство конкретной пары, и
// должен показываться один раз в хедере общего layout'а, когда он
// появится, а не дублироваться на каждой вкладке символа. Оставлена
// здесь готовой к переиспользованию тем layout'ом.
func statusBadge(s ws.Status) string {
	switch s {
	case ws.StatusConnected:
		return lipgloss.NewStyle().Foreground(colorOK).Render("● connected")
	case ws.StatusReconnecting:
		return lipgloss.NewStyle().Foreground(colorSOS).Render("○ reconnecting...")
	default:
		return lipgloss.NewStyle().Foreground(colorWarn).Render("○ connecting...")
	}
}

// bestPricesText показывает best bid/ask из стакана (канал orderbook),
// жирным, отформатированные через formatNumber. Заменяет собой
// statusBadge в шапке вкладки символа (решение из чата: статус связи
// переезжает в общий хедер, здесь вместо него — самая востребованная
// цифра для конкретной пары).
func bestPricesText(ob *indicators.OrderBook) string {
	if ob == nil {
		return mutedStyle.Render("bid/ask: ожидание данных orderbook...")
	}

	bidStr := "n/a"
	if bid, ok := ob.BestBid(); ok {
		bidStr = formatNumber(bid)
	}
	askStr := "n/a"
	if ask, ok := ob.BestAsk(); ok {
		askStr = formatNumber(ask)
	}

	bold := lipgloss.NewStyle().Bold(true)
	return fmt.Sprintf(
		"%s %s  %s %s  %s",
		mutedStyle.Render("bid"),
		bold.Foreground(colorOK).Render(bidStr),
		mutedStyle.Render("ask"),
		bold.Foreground(colorSOS).Render(askStr),
		mutedStyle.Render("USDT"),
	)
}

// timeframesToShow возвращает список ТФ для отображения: сначала
// известный порядок (indicators.Timeframes), затем — любые остальные
// ключи, которые реально пришли в snapshot, но не входят в этот
// список (см. комментарий у indicators.Timeframes: список — это
// порядок представления, не фильтр).
func timeframesToShow(snap indicators.Snapshot) []string {
	seen := make(map[string]bool, len(indicators.Timeframes))
	ordered := make([]string, 0, len(snap.Trend))

	for _, tf := range indicators.Timeframes {
		if _, ok := snap.Trend[tf]; ok {
			ordered = append(ordered, tf)
			seen[tf] = true
		}
	}
	for tf := range snap.Trend {
		if !seen[tf] {
			ordered = append(ordered, tf)
		}
	}
	return ordered
}

// timeframeBlockWidth — ширина одного ТФ-блока (разделительная линия
// и общая ширина рамки ориентируются на неё). С добавлением
// formatNumber (### ###,## — разделители тысяч) и двусторонних
// шкал (barWidth=15) самая длинная строка — PRESSURE в шапке
// ("PRESSURE <шкала> 0.518  bid/ask 41 053,00/79 193,00") — пересчитано
// под неё с запасом на случай ещё больших объёмов.
const timeframeBlockWidth = 65

var (
	// blockLabelStyle — лейбл строки внутри блока ("TREND", "ANGLE"...),
	// приглушённый и фиксированной ширины, чтобы значения начинались
	// с одного отступа во всех строках блока.
	blockLabelStyle = lipgloss.NewStyle().Foreground(colorMuted).Width(9)

	// blockTFHeaderStyle — заголовок блока (имя таймфрейма), крупнее
	// и заметнее обычного текста — это визуальный якорь блока.
	blockTFHeaderStyle = lipgloss.NewStyle().Foreground(colorText).Bold(true)

	// blockDividerStyle — тонкая граница снизу каждого блока
	// (border-bottom), отделяющая ТФ-блоки друг от друга — то, о
	// чём просили: "слегка заметный border-bottom", не полная рамка
	// вокруг каждого блока (это было бы избыточно тяжело для ленты
	// из трёх блоков подряд).
	blockDividerStyle = lipgloss.NewStyle().Foreground(colorMuted)
)

// formatNumber форматирует число в стиле "### ###,##" (пробел —
// разделитель тысяч, запятая — десятичный разделитель), по запросу
// из чата: "необходимо выводить форматированные значения в формате
// ### ###,##". Применяется к объёмам (buy/sell/bid/ask) — это не
// деньги в валюте счёта, а объём актива/контрактов (см. обсуждение
// в чате), но тот же читаемый формат чисел уместен и для них.
//
// Реализовано вручную (не через golang.org/x/text/message, который
// умеет это "из коробки"), чтобы не тащить лишнюю зависимость ради
// одной функции форматирования и не полагаться на её выбор
// разделителей по конкретной локали — нам нужен ровно фиксированный
// формат "пробел + запятая", а не локаль-зависимый.
func formatNumber(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}

	// %.2f сначала — так десятичное округление (в т.ч. до целых при
	// x,995 → x+1,00) делает стандартная библиотека, а не наша
	// самодельная арифметика, которую было бы легче сломать на
	// граничных случаях (например 999999.995).
	str := fmt.Sprintf("%.2f", v)
	intPart, fracPart, _ := strings.Cut(str, ".")

	var b strings.Builder
	n := len(intPart)
	for i, r := range intPart {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
	}

	result := b.String() + "," + fracPart
	if neg {
		result = "-" + result
	}
	return result
}

// renderTimeframeBlocks рисует ленту вертикальных карточек — по одной
// на таймфрейм, каждая со всеми индикаторами этого ТФ построчно
// (TREND/ANGLE/RSI/EMA/VOLUME), разделённых тонкой границей снизу.
//
// Решение из чата: "если рассматриваем как отдельную вкладку — можно
// делать макет более развёрнутым, вплоть до один индикатор в
// отдельных блоках". Группировка выбрана по таймфрейму (не по
// индикатору): один блок = полная картина по одному ТФ, а не один
// блок = один индикатор на все ТФ сразу — так остаётся понятно "что
// происходит на 1m" одним взглядом на один блок, а не приходится
// сопоставлять строки в трёх разных блоках-индикаторах.
func renderTimeframeBlocks(snap indicators.Snapshot) string {
	var b strings.Builder

	tfs := timeframesToShow(snap)
	for i, tf := range tfs {
		t := snap.Trend[tf]
		v := snap.Volume[tf]

		b.WriteString(blockTFHeaderStyle.Render(tf))
		b.WriteString("\n")

		b.WriteString(blockLabelStyle.Render("TREND"))
		b.WriteString(directionText(t.Direction))
		b.WriteString("\n")

		b.WriteString(blockLabelStyle.Render("ANGLE"))
		b.WriteString(angleBar(t.Angle))
		b.WriteString("\n")

		b.WriteString(blockLabelStyle.Render("RSI"))
		b.WriteString(dataStyle.Render(fmt.Sprintf("%.1f", t.RSI)))
		b.WriteString("\n")

		b.WriteString(blockLabelStyle.Render("EMA Δ%"))
		b.WriteString(emaSpreadBar(t.EMAFast, t.EMASlow))
		b.WriteString("\n")

		b.WriteString(blockLabelStyle.Render("VOL Δ%"))
		b.WriteString(volumeDeltaBar(v.BuyVol, v.SellVol, v.Spike))
		b.WriteString("\n")

		// border-bottom: не после последнего блока в ленте — там уже
		// есть внешняя рамка borderStyle, вторая граница подряд
		// выглядела бы избыточно.
		if i < len(tfs)-1 {
			b.WriteString(blockDividerStyle.Render(strings.Repeat("─", timeframeBlockWidth)))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// barWidth — количество ячеек двусторонней мини-шкалы (в 3 раза
// длиннее исходной версии по запросу — "увеличить в три раза длину
// индикаторов"). Нечётное число — есть единственная центральная
// ячейка для нуля, шкала визуально симметрична влево/вправо.
const barWidth = 15

// barCenter — индекс центральной ячейки (ноль), 0-based.
const barCenter = barWidth / 2 // 7 при barWidth=15

// bipolarBar рисует двустороннюю шкалу с центром в нуле: отрицательные
// значения растут влево от центра, положительные — вправо (решение
// из чата: "за ноль принять середину индикатора, отрицательные
// значения влево, положительные вправо"). Общая механика для
// angle/EMA-спреда/imbalance — каждый вызывает её со своим max
// (потолок нормализации) и своим смещением нуля (angle/EMA — ноль в
// нуле, imbalance — "ноль" смещён к 1.0, см. renderPressureInline).
//
// value — уже смещённое значение (то есть для imbalance это
// imbalance-1.0, не сам imbalance) — так сама bipolarBar не должна
// знать про разные точки отсчёта у разных индикаторов, это ответственность
// вызывающего кода.
func bipolarBar(value, max float64) (cells string, filledRight int, filledLeft int) {
	v := value
	if v > max {
		v = max
	}
	if v < -max {
		v = -max
	}
	filled := int(v / max * float64(barCenter))

	cellsRunes := make([]rune, barWidth)
	for i := range cellsRunes {
		cellsRunes[i] = '░'
	}
	cellsRunes[barCenter] = '│' // центральная отметка нуля — всегда видна,
	// даже когда filled==0, чтобы шкала не выглядела как "просто
	// пустая полоса", а явно показывала, где проходит ноль.

	switch {
	case filled > 0:
		for i := barCenter + 1; i <= barCenter+filled && i < barWidth; i++ {
			cellsRunes[i] = '█'
		}
		filledRight = filled
	case filled < 0:
		for i := barCenter - 1; i >= barCenter+filled && i >= 0; i-- {
			cellsRunes[i] = '█'
		}
		filledLeft = -filled
	}

	return string(cellsRunes), filledRight, filledLeft
}

// angleMaxDegrees — верхняя граница нормализации для шкалы угла.
// Не теоретический предел angle (у него формально такого нет) — это
// практический потолок: в реальных прод-данных (см.
// internal/tui/symbol_test.go, снятые с прода 2026-08-17) наблюдались
// значения примерно до ±80. 90 — круглое число с небольшим запасом
// сверх этого. Значения за пределами зажимаются (clamp), а не
// растягивают шкалу — иначе один резкий выброс "сплющил" бы шкалу
// для всех обычных, не экстремальных значений в этом же столбце и
// в последующих кадрах.
const angleMaxDegrees = 90.0

// angleBar рисует двустороннюю шкалу угла тренда (центр=0°, влево
// отрицательные, вправо положительные — direction уже показан
// отдельно в TREND, здесь только сила и знак угла), число — после
// шкалы (решение из чата).
func angleBar(angle float64) string {
	bar, right, left := bipolarBar(angle, angleMaxDegrees)

	color := colorNeutral
	switch {
	case right > 0:
		color = colorOK
	case left > 0:
		color = colorSOS
	}

	return lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("%s %+.0f°", bar, angle))
}

// emaSpreadMaxPercent — верхняя граница нормализации для шкалы
// EMA-спреда, тот же принцип clamp, что и angleMaxDegrees.
//
// Откалибровано по реальным данным с прода (см. realIndicatorsPayload
// в symbol_test.go, снято 2026-08-17): спред fast/slow EMA у BTC_USDT
// на 1m/8m/24m составлял порядка 0.00-0.03% — на порядки меньше, чем
// изначально предполагавшийся потолок 1% (при котором шкала всегда
// выглядела пустой, см. историю правки). 0.05% — практический потолок
// с небольшим запасом сверх типично наблюдаемых значений; при более
// широком наблюдении на разных символах/волатильности может
// понадобиться донастройка (см. TODO ниже).
//
// TODO(наблюдение): если на практике спред регулярно вылезает за
// 0.05% и упирается в потолок шкалы (все бары выглядят одинаково
// полными) — поднять эту константу; если наоборот почти никогда не
// отличим от нуля даже при заметном движении цены — понизить ещё.
const emaSpreadMaxPercent = 0.05

// emaSpreadBar рисует двустороннюю шкалу расхождения EMA fast/slow
// как процент от slow (спред в % = (fast-slow)/slow*100 — стандартный
// способ сделать разницу цен сопоставимой между разными по цене
// символами, а не сравнивать абсолютные дельты в валюте). Центр=0%
// (fast==slow), влево — fast<slow (нисходящее расхождение), вправо
// — fast>slow (восходящее).
// volumeDeltaMaxPercent — потолок нормализации для шкалы дельты
// объёма. В отличие от angleMaxDegrees/emaSpreadMaxPercent (сырые
// величины, требующие эмпирической калибровки под конкретный
// диапазон), это уже нормализованный процент по формуле ниже — у
// него естественный теоретический предел ровно ±100% (весь объём
// в одну сторону), так что калибровать нечего, берём сам предел.
const volumeDeltaMaxPercent = 100.0

// volumeDeltaBar рисует двустороннюю шкалу дисбаланса объёма покупок/
// продаж за таймфрейм: (buy-sell)/(buy+sell)*100 — симметричный
// процент, решение из чата ("числовые значения отображать в
// процентах"), заменяет собой прежние раздельные BUY/SELL-числа.
// Влево — sell доминирует, вправо — buy доминирует. При spike=true
// (см. indicators.Volume.Spike) добавляет пометку 🔥, тот же смысл,
// что раньше был у volCellWithSpike.
func volumeDeltaBar(buyVol, sellVol float64, spike bool) string {
	total := buyVol + sellVol
	if total == 0 {
		return mutedStyle.Render(strings.Repeat("░", barCenter) + "│" + strings.Repeat("░", barCenter) + " n/a")
	}
	deltaPct := (buyVol - sellVol) / total * 100

	bar, right, left := bipolarBar(deltaPct, volumeDeltaMaxPercent)

	color := colorNeutral
	switch {
	case right > 0:
		color = colorOK // buy доминирует
	case left > 0:
		color = colorSOS // sell доминирует
	}

	text := fmt.Sprintf("%s %+.1f%%", bar, deltaPct)
	if spike {
		text += " 🔥"
	}
	return lipgloss.NewStyle().Foreground(color).Render(text)
}

func emaSpreadBar(fast, slow float64) string {
	if slow == 0 {
		return mutedStyle.Render(strings.Repeat("░", barWidth) + " n/a")
	}
	spreadPct := (fast - slow) / slow * 100

	bar, right, left := bipolarBar(spreadPct, emaSpreadMaxPercent)

	color := colorNeutral
	switch {
	case right > 0:
		color = colorOK
	case left > 0:
		color = colorSOS
	}

	return lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("%s %+.2f%%", bar, spreadPct))
}

// directionText раскрашивает направление тренда — зелёный/красный/
// приглушённый, тот же принцип цветовых статусов, что в разделе 11.
func directionText(direction string) string {
	switch direction {
	case "up":
		return lipgloss.NewStyle().Foreground(colorOK).Render("▲ up")
	case "down":
		return lipgloss.NewStyle().Foreground(colorSOS).Render("▼ down")
	default:
		return lipgloss.NewStyle().Foreground(colorNeutral).Render("· neutral")
	}
}

// renderPressureBlock показывает bid/ask давление как двустороннюю
// шкалу симметричного процента (bid-ask)/(bid+ask)*100, центр — паритет
// (bid==ask), влево — давление продавцов, вправо — давление покупателей
// (замена прежнего коэффициента imbalance=bid/ask на процент — решение
// привязан ни к одному конкретному таймфрейму), поэтому рисуется
// отдельным блоком в шапке вкладки, под именем символа/статусом —
// шкала той же ширины (barWidth), что и остальные индикаторы, не
// поместилась бы в одну строку с заголовком (решение из чата).
func renderPressureBlock(p indicators.Pressure) string {
	// Симметричный процент вместо коэффициента imbalance=bid/ask,
	// решение из чата: "и в шапке тоже pressure" (в процентах, тем
	// же принципом, что дельта объёма — (bid-ask)/(bid+ask)*100).
	total := p.BidVol + p.AskVol
	var imbalancePct float64
	if total != 0 {
		imbalancePct = (p.BidVol - p.AskVol) / total * 100
	}

	bar, right, left := bipolarBar(imbalancePct, volumeDeltaMaxPercent)

	color := colorNeutral
	switch {
	case right > 0:
		color = colorOK // bid > ask — давление покупателей
	case left > 0:
		color = colorSOS // ask > bid — давление продавцов
	}

	return fmt.Sprintf(
		"%s%s %s  %s %s/%s",
		blockLabelStyle.Render("PRESSURE"),
		lipgloss.NewStyle().Foreground(color).Render(bar),
		lipgloss.NewStyle().Foreground(color).Bold(true).Render(fmt.Sprintf("%+.1f%%", imbalancePct)),
		mutedStyle.Render("bid/ask"),
		dataStyle.Render(formatNumber(p.BidVol)),
		dataStyle.Render(formatNumber(p.AskVol)),
	)
}
