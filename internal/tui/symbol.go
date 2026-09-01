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

	"github.com/charmbracelet/bubbles/viewport"
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
	colorText    = lipgloss.Color("252") // текст важный — приглушённый светло-серый, решение из чата: "чуть светлее, но не белый" (было 255 — чистый белый)
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

	status    ws.Status
	snapshot  *indicators.Snapshot  // nil, пока ничего не пришло по этому символу
	orderbook *indicators.OrderBook // nil, пока не пришло ни одного orderbook-сообщения для этого символа
	lastErr   string                // последняя ошибка разбора JSON, если была — не молчим о битых данных

	width, height int

	// vp — прокручиваемая область для тела вкладки (две колонки:
	// стакан + ТФ-блоки). Решение из чата: контент не помещается по
	// высоте на обычном терминале (карточки-индикаторы с border-bottom
	// заметно раздули высоту), а сжатие по высоте решили не делать —
	// вместо этого прокрутка. Шапка (имя символа, bid/ask) вне vp —
	// остаётся видна всегда, не должна "уезжать вверх" при скролле,
	// то, на что и жаловался запрос ("шапка уехала вверх").
	vp      viewport.Model
	vpReady bool // true после первого WindowSizeMsg — viewport нельзя использовать до реального размера терминала
}

// New создаёт модель для одного символа.
//
// Не принимает *ws.Client (в отличие от более ранней версии) —
// решение из чата про архитектуру главного лайаута: при нескольких
// вкладках (по числу символов) только App (см. app.go) должен читать
// из client.Messages/client.Status напрямую, рассылая полученные
// сообщения во все вкладки через их Update(); если бы каждая Model
// сама слушала общий небуферизованный канал, вкладки конкурировали бы
// друг с другом за каждое входящее сообщение.
func New(symbol string) Model {
	return Model{
		symbol: symbol,
		status: ws.StatusConnecting,
	}
}

func (m Model) Init() tea.Cmd {
	// Пустой Init: раньше здесь модель сама слушала client.Messages/
	// client.Status через waitForMessage/waitForStatus. С появлением
	// главного лайаута (app.go) вкладок стало несколько (по числу
	// символов) — если бы каждая Model продолжала сама читать из
	// ОДНОГО общего (небуферизованного) канала client.Messages,
	// вкладки конкурировали бы за каждое входящее сообщение: только
	// одна выиграла бы гонку, остальные вообще не увидели бы свои
	// данные. Теперь единственный читатель канала — App (см. app.go),
	// который сам рассылает каждое полученное сообщение во все
	// активные вкладки через их Update() — как обычный tea.Msg, а не
	// через отдельное чтение канала внутри каждой вкладки.
	return nil
}

// headerHeight — количество строк, занимаемых шапкой вкладки (имя
// символа + bid/ask + пустая строка-отступ) плюс верх/низ внешней
// рамки borderStyle. Используется, чтобы посчитать, сколько высоты
// терминала реально достаётся под прокручиваемое тело (viewport) —
// не всю высоту терминала целиком, иначе шапка вытеснялась бы телом
// или обрезалась.
//
// Подобрано под фактическую структуру View(): 2 строки паддинга рамки
// сверху + 1 строка заголовка + 1 пустая строка + 2 строки паддинга
// рамки снизу = 6. Не идеальный универсальный расчёт (например
// удлинится, если m.lastErr показывает ошибку под телом), но
// достаточен как база — небольшой запас лучше точного попадания.
const headerHeight = 6

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		vpHeight := m.height - headerHeight
		if vpHeight < 1 {
			vpHeight = 1
		}
		// vpWidth: вычитаем ширину, которую съедает внешняя рамка
		// borderStyle (RoundedBorder — по 1 символу слева/справа = 2,
		// плюс Padding(1,2) — по 2 символа слева/справа = 4). Без
		// этого vp.Width равнялся полной ширине терминала, а сверху
		// ещё добавлялась рамка — итоговый вывод превышал ширину
		// экрана и растягивался/переносился некрасиво (решение из
		// чата: "оранжевая рамка виджета растянута на весь экран").
		vpWidth := m.width - 6
		if vpWidth < 1 {
			vpWidth = 1
		}
		if !m.vpReady {
			m.vp = viewport.New(vpWidth, vpHeight)
			m.vpReady = true
		} else {
			m.vp.Width = vpWidth
			m.vp.Height = vpHeight
		}
		m.vp.SetContent(m.renderBody())
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "ctrl+l":
			// Решение из чата: сторонний Ctrl+L (стандартная функция
			// самого терминала "очистить экран") может рассинхронизировать
			// внутренний кэш строк рендерера bubbletea с тем, что
			// реально видно в терминале — терминал стирает экран сам,
			// в обход bubbletea, а рендерер продолжает думать, что
			// "неизменившиеся" строки (включая рамку) уже на месте, и
			// пропускает их перерисовку до тех пор, пока их содержимое
			// реально не изменится хоть на символ. Явно перехватываем
			// Ctrl+L здесь и просим bubbletea принудительно перерисовать
			// всё (tea.ClearScreen сбрасывает внутренний кэш строк
			// рендерера, см. standardRenderer.repaint) — так наша
			// программа сама управляет полной перерисовкой, а не
			// полагается на то, что терминал синхронизирует своё
			// состояние с bubbletea корректно.
			return m, tea.ClearScreen
		}
		// Остальные клавиши (стрелки, PageUp/PageDown, home/end) —
		// отдаём viewport.Update, он сам знает, какие клавиши двигают
		// прокрутку (см. bubbles/viewport.DefaultKeyMap).
		if m.vpReady {
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return m, cmd
		}
		return m, nil

	case wsStatusMsg:
		m.status = ws.Status(msg)
		return m, nil

	case wsMsg:
		if msg.Symbol != m.symbol {
			// Не наш символ — игнорируем оба интересующих нас канала
			// (indicators и orderbook).
			return m, nil
		}

		changed := false
		switch msg.Channel {
		case "indicators":
			var snap indicators.Snapshot
			if err := json.Unmarshal(msg.Data, &snap); err != nil {
				m.lastErr = err.Error()
				return m, nil
			}
			m.snapshot = &snap
			m.lastErr = ""
			changed = true

		case "orderbook":
			var ob indicators.OrderBook
			if err := json.Unmarshal(msg.Data, &ob); err != nil {
				m.lastErr = err.Error()
				return m, nil
			}
			m.orderbook = &ob
			m.lastErr = ""
			changed = true
		}

		// Обновляем содержимое viewport при каждом новом snapshot/
		// orderbook — иначе прокручиваемая область продолжала бы
		// показывать устаревшие данные несмотря на то, что m.snapshot/
		// m.orderbook уже обновились. SetContent сбрасывает позицию
		// прокрутки на верх — компромисс: живые данные важнее, чем
		// сохранение точной позиции скролла между обновлениями раз в
		// несколько секунд (см. интервалы pollIndicators/pollOrderBook
		// в ws-server).
		if changed && m.vpReady {
			m.vp.SetContent(m.renderBody())
		}

		return m, nil
	}

	return m, nil
}

// renderBody строит содержимое прокручиваемой области (обе колонки:
// стакан + ТФ-блоки), без шапки — шапка рисуется отдельно в View()
// и остаётся видна всегда, вне viewport (см. комментарий у поля vp
// в Model). Вынесено в отдельный метод, потому что нужно в двух
// местах: при инициализации/ресайзе viewport и при получении новых
// данных (Update: case wsMsg) — оба раза вызывают m.vp.SetContent(...).
// scrollIndicator показывает позицию прокрутки (например "▼ 42%") в
// шапке, когда контент не помещается целиком — сам viewport из bubbles
// не рисует scrollbar по умолчанию, только реагирует на клавиши, так
// что без явного индикатора пользователь не может понять, что часть
// контента скрыта ниже/выше (решение из чата: "полосы прокрутки нет,
// надо бы предусмотреть").
//
// Пустая строка, если весь контент и так помещается (AtTop && AtBottom
// одновременно) — незачем показывать "100%" там, где скроллить нечего.
func scrollIndicator(vp viewport.Model) string {
	if vp.AtTop() && vp.AtBottom() {
		return ""
	}
	pct := int(vp.ScrollPercent() * 100)
	arrow := "↕"
	switch {
	case vp.AtTop():
		arrow = "▼" // есть куда скроллить только вниз
	case vp.AtBottom():
		arrow = "▲" // есть куда скроллить только вверх
	}
	return mutedStyle.Render(fmt.Sprintf("%s %d%%", arrow, pct))
}

func (m Model) renderBody() string {
	if m.snapshot == nil {
		var b strings.Builder
		b.WriteString(mutedStyle.Render("ожидание данных indicators..."))
		if m.lastErr != "" {
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(colorSOS).Render("ошибка разбора: " + m.lastErr))
		}
		return b.String()
	}

	var right strings.Builder
	right.WriteString(renderPressureBlock(m.snapshot.Pressure))
	right.WriteString("\n")
	right.WriteString(blockDividerStyle.Render(strings.Repeat("─", indicatorCardWidth+2)))
	right.WriteString("\n")
	right.WriteString(renderTimeframeBlocks(*m.snapshot))

	left := renderOrderbookColumn(m.orderbook)

	// Две колонки бок о бок: стакан (20 строк, приближенный к
	// биржевому виду) слева, ТФ-блоки справа — решение из чата
	// ("разделим карточку символа на две части по горизонтали").
	// lipgloss.JoinHorizontal выравнивает обе колонки по высоте сам,
	// не нужно вручную считать разницу строк между ними.
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, "   ", right.String())

	if m.lastErr != "" {
		body += "\n" + lipgloss.NewStyle().Foreground(colorSOS).Render("ошибка разбора последнего сообщения: "+m.lastErr)
	}

	return body
}

func (m Model) View() string {
	var head strings.Builder

	head.WriteString(titleStyle.Render(fmt.Sprintf("  %s  ", m.symbol)))
	head.WriteString("  ")
	head.WriteString(bestPricesText(m.orderbook))

	if m.vpReady {
		head.WriteString("  ")
		head.WriteString(scrollIndicator(m.vp))
	}
	head.WriteString("\n\n")

	if !m.vpReady {
		// Первый WindowSizeMsg ещё не пришёл — bubbletea гарантирует,
		// что он придёт до первого реального взаимодействия, так что
		// это состояние видно доли секунды на самом старте программы,
		// не постоянно.
		head.WriteString(mutedStyle.Render("инициализация..."))
		return borderStyle.Render(head.String())
	}

	head.WriteString(m.vp.View())

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

		b.WriteString(renderIndicatorCard("TREND", directionText(t.Direction)))
		b.WriteString("\n\n")

		b.WriteString(renderIndicatorCard("ANGLE", angleBar(t.Angle)))
		b.WriteString("\n\n")

		b.WriteString(renderIndicatorCard("RSI", dataStyle.Render(fmt.Sprintf("%.1f", t.RSI))))
		b.WriteString("\n\n")

		b.WriteString(renderIndicatorCard("EMA Δ%", emaSpreadBar(t.EMAFast, t.EMASlow)))
		b.WriteString("\n\n")

		b.WriteString(renderIndicatorCard("VOL Δ%", volumeDeltaBar(v.BuyVol, v.SellVol, v.Spike)))
		b.WriteString("\n")

		// Разделитель между ТФ-блоками (не между показателями внутри
		// блока — те теперь просто пустой строкой, решение из чата:
		// "попробуй убрать бордеры между индикаторами, оставив просто
		// пустые строки"): не после последнего блока в ленте — там
		// уже есть внешняя рамка borderStyle.
		if i < len(tfs)-1 {
			b.WriteString(blockDividerStyle.Render(strings.Repeat("─", indicatorCardWidth+2)))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderIndicatorCard рисует один показатель как простую строку
// "лейбл значение", без рамки вокруг — решение из чата: сначала
// пробовали полную рамку вокруг каждого индикатора, потом border-
// bottom-only, в итоге отказались от любых рамок между показателями
// в пользу пустых строк-разделителей (см. renderTimeframeBlocks) —
// визуально легче, чем любой вариант с линиями.
func renderIndicatorCard(label, value string) string {
	return blockLabelStyle.Render(label) + value
}

// indicatorCardWidth — видимая ширина самой длинной строки-показателя
// внутри ТФ-блока (лейбл+шкала+число), используется для расчёта длины
// разделителя между ТФ-блоками (indicatorCardWidth+2 в
// renderTimeframeBlocks) — раньше эту роль играла отдельная константа
// timeframeBlockWidth, рассчитанная под старую раскладку с рамками
// вокруг каждого индикатора; после отказа от рамок (решение из чата:
// "убрать бордеры между индикаторами, оставив пустые строки")
// разделители оказались длиннее реального контента — используем
// единый источник истины вместо двух рассинхронизирующихся констант.
//
// Измерено через lipgloss.Width на реальных строках (не подсчитано
// на бумаге вручную — такой расчёт для orderbookRowWidth расходился
// с реальностью ранее в этой же сессии): самая длинная строка
// (VOL Δ%, например "VOL Δ%   <шкала> +100.0%") даёт 48 видимых
// символов, +4 запаса.
const indicatorCardWidth = 52

// barWidth — количество ячеек двусторонней мини-шкалы. Увеличено
// вдвое от предыдущей версии (15 → ~31) по запросу — "увеличь ширину
// двухнаправленных шкал индикаторов ещё в два раза". Нечётное число —
// есть единственная центральная ячейка для нуля, шкала визуально
// симметрична влево/вправо.
const barWidth = 31

// barCenter — индекс центральной ячейки (ноль), 0-based.
const barCenter = barWidth / 2 // 15 при barWidth=31

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
	cellsRunes[barCenter] = '┆' // центральная отметка нуля — всегда
	// видна, даже когда filled==0, чтобы шкала не выглядела как
	// "просто пустая полоса", а явно показывала, где проходит ноль.
	// '┆' (лёгкая пунктирная), не '│' (сплошная) — решение из чата:
	// сплошная линия визуально сливалась с соседним закрашенным '█'
	// в подобие "двойного бордера", выглядело тяжеловесно.

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
// (bid==ask), влево — давление продавцов, вправо — давление покупателей.
// bid/ask объёмы вынесены отдельной строкой ПОД шкалой, центрированы
// по её ширине (решение из чата: "перенеси bid/ask объёмы под шкалу
// pressure с выравниванием по центру" — было в одной строке со шкалой
// и процентом, теперь разнесено на две строки).
// pressureBar считает и рисует только шкалу+процент давления
// (bid-ask)/(bid+ask)*100, без подписи "PRESSURE" и без строки
// bid/ask объёмов — общая часть, переиспользуемая и в
// renderPressureBlock (вкладка символа, две строки), и в Dashboard
// (dashboard.go, одна компактная строка в заголовке блока символа).
func pressureBar(p indicators.Pressure) (bar string, imbalancePct float64) {
	total := p.BidVol + p.AskVol
	if total != 0 {
		imbalancePct = (p.BidVol - p.AskVol) / total * 100
	}

	rawBar, right, left := bipolarBar(imbalancePct, volumeDeltaMaxPercent)

	color := colorNeutral
	switch {
	case right > 0:
		color = colorOK // bid > ask — давление покупателей
	case left > 0:
		color = colorSOS // ask > bid — давление продавцов
	}

	return lipgloss.NewStyle().Foreground(color).Render(rawBar), imbalancePct
}

func renderPressureBlock(p indicators.Pressure) string {
	bar, imbalancePct := pressureBar(p)

	color := colorNeutral
	switch {
	case imbalancePct > 0:
		color = colorOK
	case imbalancePct < 0:
		color = colorSOS
	}

	line1 := fmt.Sprintf(
		"%s%s %s",
		blockLabelStyle.Render("PRESSURE"),
		bar,
		lipgloss.NewStyle().Foreground(color).Bold(true).Render(fmt.Sprintf("%+.1f%%", imbalancePct)),
	)

	volumesText := fmt.Sprintf(
		"%s %s/%s",
		mutedStyle.Render("bid/ask"),
		dataStyle.Render(formatNumber(p.BidVol)),
		dataStyle.Render(formatNumber(p.AskVol)),
	)
	// Центрируем по ширине лейбла+шкалы (blockLabelStyle.GetWidth()+
	// barWidth), а не по всей ширине правой колонки — bid/ask должен
	// визуально центрироваться относительно самой шкалы, которую он
	// поясняет, а не относительно всего блока, который шире (в нём
	// есть ещё правая часть под число вроде "-32.5%").
	pressureLineWidth := blockLabelStyle.GetWidth() + barWidth
	line2 := lipgloss.NewStyle().Width(pressureLineWidth).Align(lipgloss.Center).Render(volumesText)

	return line1 + "\n" + line2
}
