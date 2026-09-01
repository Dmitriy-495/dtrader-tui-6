// Файл app.go — модель главного лайаута приложения (см. раздел 9
// CHECKPOINT.md dtrader-6): header + tabs + [content|rightbar] +
// footer. Единственный получатель сообщений из ws.Client — сама App
// не рисует данные символов напрямую, а держит по одной symbol.Model
// на каждый торгуемый символ и рассылает им входящие WS-сообщения
// через их Update() (см. комментарий у symbol.New про причину этого
// решения: несколько вкладок не могут каждая сама слушать один
// небуферизованный канал).
package tui

import (
	"encoding/json"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dmitriy-495/dtrader-tui-6/internal/indicators"
	"github.com/Dmitriy-495/dtrader-tui-6/internal/ws"
)

// appWsMsg/appWsStatusMsg — обёртки над ws.Message/ws.Status для
// главного цикла App, аналогично wsMsg/wsStatusMsg в symbol.go.
// Отдельный тип (не переиспользуем wsMsg напрямую), потому что App
// сама читает канал через свой собственный tea.Cmd — жизненный цикл
// её ожидания сообщений независим от отдельных вкладок (которые
// теперь вообще не читают канал, см. symbol.New).
type appWsMsg ws.Message
type appWsStatusMsg ws.Status

// App — модель главного лайаута.
type App struct {
	client *ws.Client

	status ws.Status
	system *indicators.SystemMsg // nil, пока не пришло ни одного system-сообщения

	tabs         []string         // "Dashboard" + символы, см. tabLabels
	symbolModels map[string]Model // по одной вкладке на символ, ключ — сам символ (без "Dashboard")
	activeIndex  int              // индекс в tabs; 0 = Dashboard

	logs []LogEntry

	width, height int

	// dashboardVP — прокрутка для Dashboard (вкладка №0), отдельная
	// от viewport каждой вкладки символа — решение из чата: список
	// блоков символов на Dashboard не ограничен по высоте (сколько
	// символов, столько блоков), при достаточном числе символов не
	// поместится на обычный терминал, та же причина, что уже приводила
	// к добавлению viewport во вкладку одного символа.
	dashboardVP      viewport.Model
	dashboardVPReady bool

	// rightbarVP — прокрутка ТОЛЬКО для LOGS-части rightbar, не для
	// Positions (тот остаётся отдельным всегда-видимым блоком снизу,
	// см. renderPositionsBlock). Решение из чата: за долгую работу
	// (часы, ночь) логов накапливается больше, чем помещается по
	// высоте — без ограничения rightbar физически вырастал выше
	// отведённого места и раздувал вниз весь кадр, из-за чего
	// header/tabs/content уезжали за пределы видимой области терминала
	// (реальный баг, обнаруженный на скриншоте после ночной работы).
	// Автоскролл всегда к последней записи (GotoBottom при каждом
	// addLog) — решение из чата: "не давать ручной прокрутки, всегда
	// показывать последние N".
	rightbarVP      viewport.Model
	rightbarVPReady bool

	// settings — пропорции панелей лайаута (см. settings.go). Решение
	// из чата: "вынести все пропорции панелей в настройки интерфейса".
	settings LayoutSettings
}

// NewApp создаёт модель главного лайаута. client уже должен быть
// запущен (client.Run(ctx) в отдельной горутине) — App только читает
// из client.Messages/client.Status.
// NewApp создаёт модель главного лайаута. client уже должен быть
// запущен (client.Run(ctx) в отдельной горутине) — App только читает
// из client.Messages/client.Status. settings — пропорции панелей,
// см. settings.go/LoadLayoutSettings; вызывающий код (cmd/main.go)
// сам решает, откуда их взять (layout.yaml или DefaultLayoutSettings()
// как запасной вариант при ошибке чтения).
func NewApp(client *ws.Client, settings LayoutSettings) App {
	return App{
		client:       client,
		status:       ws.StatusConnecting,
		tabs:         []string{dashboardTabLabel},
		symbolModels: make(map[string]Model),
		settings:     settings,
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(waitForAppMessage(a.client), waitForAppStatus(a.client))
}

func waitForAppMessage(client *ws.Client) tea.Cmd {
	return func() tea.Msg {
		return appWsMsg(<-client.Messages)
	}
}

func waitForAppStatus(client *ws.Client) tea.Cmd {
	return func() tea.Msg {
		return appWsStatusMsg(<-client.Status)
	}
}

// addLog добавляет запись в лог, обрезая буфер до maxLogLines с
// начала (см. rightbar.go) — самые старые записи вытесняются самыми
// новыми, а не растут неограниченно при долгой работе. Обновляет
// rightbarVP и сразу скроллит к низу (GotoBottom) — решение из чата:
// автоскролл к последней записи без ручной прокрутки.
func (a *App) addLog(level LogLevel, text string) {
	a.logs = append(a.logs, LogEntry{
		Time:  time.Now().Format("15:04:05"),
		Text:  text,
		Level: level,
	})
	if len(a.logs) > maxLogLines {
		a.logs = a.logs[len(a.logs)-maxLogLines:]
	}
	if a.rightbarVPReady {
		a.rightbarVP.SetContent(renderLogsContent(a.logs))
		a.rightbarVP.GotoBottom()
	}
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.resizeDashboardViewport()
		a.resizeRightbarViewport()
		return a.propagateWindowSize(msg)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return a, tea.Quit
		case "ctrl+l":
			return a, tea.ClearScreen
		case "tab":
			a.activeIndex = (a.activeIndex + 1) % len(a.tabs)
			return a, nil
		case "shift+tab":
			a.activeIndex = (a.activeIndex - 1 + len(a.tabs)) % len(a.tabs)
			return a, nil
		}
		// Ctrl+1..9 — прямой переход к вкладке по номеру (решение из
		// чата: "Ctrl+1..N на конкретный символ"). "ctrl+1" соответствует
		// индексу 0 (Dashboard), "ctrl+2" — индексу 1 (первый символ),
		// и так далее — та же нумерация, что видна пользователю в
		// строке вкладок (первая позиция слева = 1).
		if idx, ok := ctrlDigitIndex(msg.String()); ok && idx < len(a.tabs) {
			a.activeIndex = idx
			return a, nil
		}

		if a.activeIndex == dashboardTabIndex {
			// Прокрутка Dashboard — решение из чата: список блоков
			// символов не ограничен по высоте, тот же принцип
			// viewport, что уже есть у каждой вкладки символа.
			if a.dashboardVPReady {
				var cmd tea.Cmd
				a.dashboardVP, cmd = a.dashboardVP.Update(msg)
				return a, cmd
			}
			return a, nil
		}

		return a.updateActiveSymbolModel(msg)

	case appWsStatusMsg:
		a.status = ws.Status(msg)
		switch ws.Status(msg) {
		case ws.StatusConnected:
			a.addLog(LogInfo, "подключено к ws-server")
		case ws.StatusReconnecting:
			a.addLog(LogWarn, "соединение потеряно, переподключение...")
		}
		return a, waitForAppStatus(a.client)

	case appWsMsg:
		cmd := waitForAppMessage(a.client)
		return a.handleWsMsg(ws.Message(msg), cmd)
	}

	return a, nil
}

// resizeDashboardViewport (пере)создаёт dashboardVP под текущий
// contentSize() — вызывается при WindowSizeMsg. Отдельная функция, а
// не инлайн в Update, потому что тот же расчёт размера нужен и при
// первом создании (WindowSizeMsg может прийти раньше или позже
// первого system-сообщения — оба порядка валидны).
func (a *App) resizeDashboardViewport() {
	fullWidth, fullHeight := a.contentSize()
	// Вычитаем размер borderStyle (та же рамка, что оборачивает
	// вкладки символов и теперь Dashboard, см. renderContent) —
	// viewport должен знать РЕАЛЬНУЮ ширину/высоту под контент, не
	// полную выделенную область, иначе рамка вокруг viewport.View()
	// превысила бы contentSize() (та же ошибка, что уже находили и
	// чинили для header/footer).
	width := fullWidth - borderStyle.GetHorizontalFrameSize()
	height := fullHeight - borderStyle.GetVerticalFrameSize()
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	if !a.dashboardVPReady {
		a.dashboardVP = viewport.New(width, height)
		a.dashboardVPReady = true
	} else {
		a.dashboardVP.Width = width
		a.dashboardVP.Height = height
	}
	a.dashboardVP.SetContent(renderDashboard(a.system, a.symbolModels, width))
}

// resizeRightbarViewport (пере)создаёт rightbarVP под доступную
// высоту LOGS-части rightbar — та же схема, что resizeDashboardViewport.
// Ширина панели и пропорция Logs/Positions по высоте берутся из
// a.settings (см. settings.go), не константы. PositionsHeightPercent —
// решение из чата: "фиксированные 40% высоты rightbar всегда" (не
// подстраивается под реальное число позиций — если позиций мало,
// остаётся пустое место; если много, блок Positions обрежется по
// высоте, не растянет rightbar).
func (a *App) resizeRightbarViewport() {
	_, fullHeight := a.contentSize()

	rbWidth := a.settings.rightbarWidth(a.width)
	width := rbWidth - rightbarBorderStyle.GetHorizontalFrameSize()
	rightbarInnerHeight := fullHeight - rightbarBorderStyle.GetVerticalFrameSize()

	positionsHeight := a.settings.positionsHeight(rightbarInnerHeight)
	height := rightbarInnerHeight - positionsHeight
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	if !a.rightbarVPReady {
		a.rightbarVP = viewport.New(width, height)
		a.rightbarVPReady = true
	} else {
		a.rightbarVP.Width = width
		a.rightbarVP.Height = height
	}
	a.rightbarVP.SetContent(renderLogsContent(a.logs))
	a.rightbarVP.GotoBottom()
}

// ctrlDigitIndex разбирает строки вида "ctrl+1".."ctrl+9" в индекс
// вкладки (0-based). Отдельная функция — чище, чем строковый switch
// на 9 одинаковых веток внутри Update.
func ctrlDigitIndex(key string) (int, bool) {
	if len(key) != len("ctrl+1") {
		return 0, false
	}
	prefix, digit := key[:len(key)-1], key[len(key)-1]
	if prefix != "ctrl+" || digit < '1' || digit > '9' {
		return 0, false
	}
	return int(digit - '1'), true
}

// contentSize вычисляет размер, реально доступный под контент вкладки
// (после вычета header/tabs/footer/rightbar общего лайаута) — то же
// вычисление, что использует View() для самого рендера (см. bodyHeight/
// contentWidth там), вынесено сюда как отдельная функция, чтобы
// вкладки символов получали WindowSizeMsg с ЭТИМ размером, а не с
// полным размером терминала. Раньше (баг, найденный при первом
// визуальном превью главного лайаута) вкладки получали a.width/a.height
// целиком — их viewport считал себя больше, чем реально отведённое
// им место, и итоговая высота content не совпадала с rightbar,
// рамки визуально "разъезжались".
func (a App) contentSize() (width, height int) {
	header := renderHeader(a.system, a.width)
	tabsBar := renderTabs(a.tabsSymbolsOnly(), a.activeIndex, a.width)
	footer := renderFooter(a.width)

	width = a.width - a.settings.rightbarWidth(a.width)
	if width < 1 {
		width = 1
	}
	height = a.height - lipgloss.Height(header) - lipgloss.Height(tabsBar) - lipgloss.Height(footer)
	if height < 1 {
		height = 1
	}
	return width, height
}

// propagateWindowSize рассылает СКОРРЕКТИРОВАННЫЙ размер (см.
// contentSize) во все существующие вкладки символов — не полный
// размер терминала, а именно то, что реально останется под контент
// после вычета header/tabs/footer/rightbar общего лайаута.
func (a App) propagateWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	width, height := a.contentSize()
	contentMsg := tea.WindowSizeMsg{Width: width, Height: height}
	for symbol, m := range a.symbolModels {
		updated, _ := m.Update(contentMsg)
		a.symbolModels[symbol] = updated.(Model)
	}
	return a, nil
}

func (a App) updateActiveSymbolModel(msg tea.Msg) (tea.Model, tea.Cmd) {
	symbol := a.activeSymbol()
	if symbol == "" {
		return a, nil // Dashboard активен — нет вкладки символа, которой можно переслать клавишу
	}
	m, ok := a.symbolModels[symbol]
	if !ok {
		return a, nil
	}
	updated, cmd := m.Update(msg)
	a.symbolModels[symbol] = updated.(Model)
	return a, cmd
}

// activeSymbol возвращает символ активной вкладки, или "" если
// активна вкладка Dashboard (индекс 0).
func (a App) activeSymbol() string {
	if a.activeIndex == dashboardTabIndex {
		return ""
	}
	if a.activeIndex < len(a.tabs) {
		return a.tabs[a.activeIndex]
	}
	return ""
}

// handleWsMsg обрабатывает одно входящее WS-сообщение: system —
// обновляет состояние самой App (баланс, позиции, список символов —
// и по факту прихода первого system-сообщения создаёт вкладки для
// каждого символа, решение из чата: "формируется при запуске по
// полученным от сервера торгуемым парам"); остальные каналы
// (indicators/orderbook/...) — рассылаются во ВСЕ вкладки символов
// разом, каждая сама решает, её ли это символ (см. symbol.go:
// msg.Symbol != m.symbol).
func (a App) handleWsMsg(msg ws.Message, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	if msg.Channel == "system" {
		var sys indicators.SystemMsg
		if err := json.Unmarshal(msg.Data, &sys); err != nil {
			a.addLog(LogError, "не удалось разобрать system: "+err.Error())
			return a, cmd
		}
		a.ensureSymbolTabs(sys.Symbols)
		a.system = &sys
		a.refreshDashboardContent()
		if a.rightbarVPReady {
			// Positions приходят в system — их изменение влияет на
			// высоту positionsBlock, значит на доступную высоту под
			// LOGS-viewport (см. resizeRightbarViewport). Пересчитываем
			// каждый раз, когда system обновляется.
			a.resizeRightbarViewport()
		}
		return a, cmd
	}

	// Не system — рассылаем во все вкладки символов. Каждая сама
	// фильтрует по msg.Symbol (см. symbol.go Update, case wsMsg).
	for symbol, m := range a.symbolModels {
		updated, _ := m.Update(wsMsg(msg))
		a.symbolModels[symbol] = updated.(Model)
	}
	// Dashboard показывает те же snapshot/orderbook, что и вкладки
	// символов (см. renderDashboard — читает их из symbolModels) —
	// любое indicators/orderbook-сообщение может изменить то, что
	// должно быть видно на Dashboard, не только system.
	a.refreshDashboardContent()
	return a, cmd
}

// refreshDashboardContent перестраивает содержимое dashboardVP из
// текущего состояния — вызывается при каждом WS-сообщении, которое
// могло изменить то, что видно на Dashboard (system/indicators/
// orderbook). Ничего не делает, если viewport ещё не создан (до
// первого WindowSizeMsg) — тогда используется актуальное состояние
// на момент resizeDashboardViewport.
func (a *App) refreshDashboardContent() {
	if !a.dashboardVPReady {
		return
	}
	// a.dashboardVP.Width уже учитывает вычет рамки (см.
	// resizeDashboardViewport) — переиспользуем его, не пересчитываем
	// заново от contentSize(), чтобы не дублировать вычитание рамки в
	// двух местах.
	a.dashboardVP.SetContent(renderDashboard(a.system, a.symbolModels, a.dashboardVP.Width))
}

// ensureSymbolTabs создаёт вкладки (symbol.Model + запись в a.tabs)
// для новых символов, которых ещё нет — решение из чата: список
// формируется один раз по факту прихода symbols от сервера, не
// пересоздаётся заново при каждом system-сообщении (это была бы
// потеря состояния уже открытых вкладок — снапшоты/скролл сбросились
// бы каждые 10s на пустое место).
func (a *App) ensureSymbolTabs(symbols []string) {
	for _, symbol := range symbols {
		if _, exists := a.symbolModels[symbol]; exists {
			continue
		}
		m := New(symbol)
		if a.width > 0 && a.height > 0 {
			width, height := a.contentSize()
			updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
			m = updated.(Model)
		}
		a.symbolModels[symbol] = m
		a.tabs = append(a.tabs, symbol)
		a.addLog(LogInfo, "добавлена вкладка "+symbol)
	}
}

func (a App) View() string {
	header := renderHeader(a.system, a.width)
	tabsBar := renderTabs(a.tabsSymbolsOnly(), a.activeIndex, a.width)

	contentWidth, bodyHeight := a.contentSize()

	content := a.renderContent(contentWidth, bodyHeight)
	var positions []indicators.Position
	if a.system != nil {
		positions = a.system.Positions
	}

	// rightbar: LOGS (свой viewport, автоскролл к низу — решение из
	// чата: ограничение по высоте, чтобы накопленные за долгую работу
	// логи не раздували весь кадр вниз) + POSITIONS (отдельный
	// всегда-видимый блок снизу с разделителем сверху, фиксированная
	// высота из a.settings.PositionsHeightPercent).
	var rightbarInner string
	if a.rightbarVPReady {
		rightbarInner = a.rightbarVP.View()
	} else {
		rightbarInner = mutedStyle.Render("инициализация...")
	}

	rbWidth := a.settings.rightbarWidth(a.width)
	positionsWidth := rbWidth - rightbarBorderStyle.GetHorizontalFrameSize()
	rightbarInnerHeight := bodyHeight - rightbarBorderStyle.GetVerticalFrameSize()
	positionsHeight := a.settings.positionsHeight(rightbarInnerHeight)
	// Защита от отрицательных/нулевых значений на маленьких терминалах —
	// найден реальный краш на проде: strings.Repeat паникует на
	// отрицательном count, если rbWidth/rightbarInnerHeight меньше,
	// чем размер рамки+паддинга rightbarBorderStyle (тот же класс
	// защиты, что уже есть в symbol.go/dashboard.go для аналогичных
	// вычислений, здесь была пропущена).
	if positionsWidth < 1 {
		positionsWidth = 1
	}
	if positionsHeight < 1 {
		positionsHeight = 1
	}

	positionsBlock := renderPositionsBlock(positions, positionsWidth, positionsHeight)
	rightbarContent := rightbarInner + "\n" + positionsBlock
	rightbar := rightbarBorderStyle.Render(rightbarContent)

	body := lipgloss.JoinHorizontal(lipgloss.Top, content, rightbar)
	footer := renderFooter(a.width)

	return lipgloss.JoinVertical(lipgloss.Left, header, tabsBar, body, footer)
}

// tabsSymbolsOnly возвращает a.tabs без "Dashboard" — renderTabs сам
// добавляет Dashboard первым элементом (см. tabLabels), передавать
// его дважды дало бы дублирование в строке вкладок.
func (a App) tabsSymbolsOnly() []string {
	if len(a.tabs) <= 1 {
		return nil
	}
	return a.tabs[1:]
}

// renderContent рисует активную вкладку: Dashboard (viewport + рамка,
// та же обёртка, что и у вкладок символов) или конкретный символ.
func (a App) renderContent(width, height int) string {
	symbol := a.activeSymbol()
	if symbol == "" {
		if !a.dashboardVPReady {
			return mutedStyle.Render("инициализация...")
		}
		return borderStyle.Render(a.dashboardVP.View())
	}
	m, ok := a.symbolModels[symbol]
	if !ok {
		return mutedStyle.Render("вкладка не найдена: " + symbol)
	}
	return m.View()
}
