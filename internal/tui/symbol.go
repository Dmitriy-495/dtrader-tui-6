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
			Padding(0, 1)

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

	status   ws.Status
	snapshot *indicators.Snapshot // nil, пока ничего не пришло по этому символу
	lastErr  string               // последняя ошибка разбора JSON, если была — не молчим о битых данных

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

		if msg.Channel != "indicators" || msg.Symbol != m.symbol {
			// Не наш канал/символ — просто игнорируем, но обязательно
			// продолжаем слушать (cmd уже запланирован строкой выше).
			return m, cmd
		}

		var snap indicators.Snapshot
		if err := json.Unmarshal(msg.Data, &snap); err != nil {
			m.lastErr = err.Error()
			return m, cmd
		}
		m.snapshot = &snap
		m.lastErr = ""
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("  %s  ", m.symbol)))
	b.WriteString("  ")
	b.WriteString(statusBadge(m.status))
	if m.snapshot != nil {
		b.WriteString("   ")
		b.WriteString(renderPressureInline(m.snapshot.Pressure))
	}
	b.WriteString("\n\n")

	if m.snapshot == nil {
		b.WriteString(mutedStyle.Render("ожидание данных indicators..."))
		if m.lastErr != "" {
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(colorSOS).Render("ошибка разбора: " + m.lastErr))
		}
		return borderStyle.Render(b.String())
	}

	b.WriteString(renderTimeframeBlocks(*m.snapshot))

	if m.lastErr != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(colorSOS).Render("ошибка разбора последнего сообщения: " + m.lastErr))
	}

	return borderStyle.Render(b.String())
}

// statusBadge — цветной индикатор состояния соединения, тот же
// набор цветов (OK/WARNING/SOS), что использует раздел 11 для
// SERV/EXCH статусов в header — здесь применяем его к статусу
// самого WS-соединения TUI.
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

// timeframeBlockWidth — ширина одного ТФ-блока в лейбл-значение
// формате (см. renderTimeframeBlocks). Достаточно для самой длинной
// строки-значения ("SELL   994399.0 🔥") с запасом.
const timeframeBlockWidth = 40

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

		b.WriteString(blockLabelStyle.Render("BUY"))
		b.WriteString(dataStyle.Render(fmt.Sprintf("%.1f", v.BuyVol)))
		b.WriteString("\n")

		b.WriteString(blockLabelStyle.Render("SELL"))
		b.WriteString(volCellWithSpike(v.SellVol, v.Spike))
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

// angleMaxDegrees — верхняя граница нормализации для мини-шкалы угла.
// Не теоретический предел angle (у него формально такого нет) — это
// практический потолок для шкалы 0..bar: в реальных прод-данных
// (см. internal/tui/symbol_test.go, снятые с прода 2026-08-17)
// наблюдались значения примерно до ±80. 90 — круглое число с небольшим
// запасом сверх этого. Значения за пределами зажимаются (clamp), а
// не растягивают шкалу — иначе один резкий выброс "сплющил" бы
// шкалу для всех обычных, не экстремальных значений в этом же
// столбце и в последующих кадрах.
const angleMaxDegrees = 90.0

// angleBarWidth — количество ячеек мини-шкалы (███░░).
const angleBarWidth = 5

// angleBar рисует мини-шкалу силы угла тренда (0..angleMaxDegrees,
// по модулю — направление уже показано отдельно в TREND) плюс само
// число рядом, раскрашенную тем же цветом, что direction (bar сам
// по себе не говорит вверх или вниз — это делает соседняя колонка
// TREND, здесь только цвет как визуальная связка с ней).
func angleBar(angle float64) string {
	abs := angle
	if abs < 0 {
		abs = -abs
	}
	if abs > angleMaxDegrees {
		abs = angleMaxDegrees
	}
	filled := int(abs / angleMaxDegrees * angleBarWidth)
	if filled > angleBarWidth {
		filled = angleBarWidth
	}

	color := colorNeutral
	switch {
	case angle > 0:
		color = colorOK
	case angle < 0:
		color = colorSOS
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", angleBarWidth-filled)
	return lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("%s %.0f°", bar, angle))
}

// emaSpreadMaxPercent — верхняя граница нормализации для мини-шкалы
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

// emaSpreadBar рисует мини-шкалу силы расхождения EMA fast/slow как
// процент от slow (спред в % = (fast-slow)/slow*100 — стандартный
// способ сделать разницу цен сопоставимой между разными по цене
// символами, а не сравнивать абсолютные дельты в валюте).
func emaSpreadBar(fast, slow float64) string {
	if slow == 0 {
		return mutedStyle.Render(strings.Repeat("░", angleBarWidth) + " n/a")
	}
	spreadPct := (fast - slow) / slow * 100

	abs := spreadPct
	if abs < 0 {
		abs = -abs
	}
	if abs > emaSpreadMaxPercent {
		abs = emaSpreadMaxPercent
	}
	filled := int(abs / emaSpreadMaxPercent * angleBarWidth)
	if filled > angleBarWidth {
		filled = angleBarWidth
	}

	color := colorNeutral
	switch {
	case spreadPct > 0:
		color = colorOK // fast > slow — восходящее расхождение
	case spreadPct < 0:
		color = colorSOS
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", angleBarWidth-filled)
	return lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("%s %+.2f%%", bar, spreadPct))
}

// volCellWithSpike — то же оформление, что buy (dataStyle), плюс
// пометка всплеска: spike относится к паре buy/sell в целом
// (indicators.Volume.Spike — одно bool-поле на весь ТФ, не отдельно
// на buy и на sell), показываем его при sell, а не дублируем на
// обеих колонках — так его видно один раз, не два.
func volCellWithSpike(sellVol float64, spike bool) string {
	text := dataStyle.Render(fmt.Sprintf("%.1f", sellVol))
	if spike {
		text += " 🔥"
	}
	return text
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

// renderPressureInline показывает bid/ask давление и imbalance —
// единственный P-индикатор без разбивки по ТФ (не привязан ни к
// одному конкретному таймфрейму), поэтому живёт в шапке вкладки
// рядом с именем символа/статусом соединения, а не в одном из
// ТФ-блоков (решение из чата).
func renderPressureInline(p indicators.Pressure) string {
	imbColor := colorNeutral
	switch {
	case p.Imbalance > 1.05:
		imbColor = colorOK // bid > ask ощутимо — давление покупателей
	case p.Imbalance < 0.95:
		imbColor = colorSOS // ask > bid ощутимо — давление продавцов
	}
	return fmt.Sprintf(
		"%s %s/%s imb=%s",
		mutedStyle.Render("P:"),
		dataStyle.Render(fmt.Sprintf("%.0f", p.BidVol)),
		dataStyle.Render(fmt.Sprintf("%.0f", p.AskVol)),
		lipgloss.NewStyle().Foreground(imbColor).Bold(true).Render(fmt.Sprintf("%.3f", p.Imbalance)),
	)
}
