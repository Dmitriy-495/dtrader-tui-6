package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Dmitriy-495/dtrader-tui-6/internal/ws"
)

// --- Стили ---
var (
	headerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("255")).
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1)

	activeIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)

	inactiveIndicator = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)

	footerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("255")).
			PaddingLeft(1)

	logStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			PaddingLeft(1).
			PaddingRight(1)

	contentStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			PaddingLeft(1).
			PaddingRight(1)

	logEntryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	logTimeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("239"))
)

// tickMsg — сообщение таймера для обновления времени в header
type tickMsg time.Time

// wsMsg — входящее сообщение от WS
type wsMsg ws.Message

// Model — главная модель TUI (bubbletea)
type Model struct {
	width  int
	height int

	// Header
	connected bool
	pingMs    int64
	balance   string
	clockTime time.Time

	// Логи (rightbar)
	logs     []string
	logView  viewport.Model

	// Контент
	contentView viewport.Model

	// Footer — командная строка
	input textinput.Model

	// WS канал
	msgCh <-chan ws.Message
}

func New(msgCh <-chan ws.Message) Model {
	input := textinput.New()
	input.Placeholder = "введите команду..."
	input.Focus()

	return Model{
		msgCh:     msgCh,
		balance:   "загрузка...",
		clockTime: time.Now(),
		input:     input,
		logs:      []string{},
	}
}

// --- Init ---
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		waitForMsg(m.msgCh),
	)
}

// tickCmd — таймер обновления каждую секунду
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// waitForMsg — ждём сообщение от WS
func waitForMsg(ch <-chan ws.Message) tea.Cmd {
	return func() tea.Msg {
		return wsMsg(<-ch)
	}
}

// --- Update ---
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcSizes()

	case tickMsg:
		m.clockTime = time.Time(msg)
		cmds = append(cmds, tickCmd())

	case wsMsg:
		m.handleWS(ws.Message(msg))
		cmds = append(cmds, waitForMsg(m.msgCh))

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			cmd := m.input.Value()
			if cmd != "" {
				m.addLog(fmt.Sprintf("> %s", cmd))
				m.input.SetValue("")
			}
		default:
			var inputCmd tea.Cmd
			m.input, inputCmd = m.input.Update(msg)
			cmds = append(cmds, inputCmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// handleWS обрабатывает входящие WS сообщения
func (m *Model) handleWS(msg ws.Message) {
	m.connected = true
	switch msg.Channel {
	case "trades":
		m.addLog(fmt.Sprintf("💹 trade %s", msg.Symbol))
	case "stats":
		m.addLog(fmt.Sprintf("📊 stats %s", msg.Symbol))
	case "liquidations":
		m.addLog(fmt.Sprintf("💥 liquidation %s", msg.Symbol))
	case "candles":
		m.addLog(fmt.Sprintf("🕯️ candle %s", msg.Symbol))
	}
}

// addLog добавляет строку в лог с временной меткой
func (m *Model) addLog(entry string) {
	ts := logTimeStyle.Render(time.Now().Format("15:04:05"))
	line := fmt.Sprintf("%s %s", ts, logEntryStyle.Render(entry))
	m.logs = append(m.logs, line)
	// храним последние 200 строк
	if len(m.logs) > 200 {
		m.logs = m.logs[len(m.logs)-200:]
	}
	m.logView.SetContent(strings.Join(m.logs, "\n"))
	m.logView.GotoBottom()
}

// recalcSizes пересчитывает размеры панелей при изменении окна
func (m *Model) recalcSizes() {
	headerH := 1
	footerH := 3
	rightbarW := 35

	contentW := m.width - rightbarW - 4
	contentH := m.height - headerH - footerH - 4

	m.contentView = viewport.New(contentW, contentH)
	m.contentView.SetContent("📊 Dashboard\n\nДанные загружаются...")

	m.logView = viewport.New(rightbarW-4, contentH)
	m.logView.SetContent("")
}

// --- View ---
func (m Model) View() string {
	if m.width == 0 {
		return "загрузка..."
	}

	return strings.Join([]string{
		m.renderHeader(),
		m.renderMain(),
		m.renderFooter(),
	}, "\n")
}

// renderHeader — одна строка с названием, временем, балансом, индикатором
func (m Model) renderHeader() string {
	// Левая часть
	left := headerStyle.Render("⚡ DTrader 6  v0.1")

	// Время биржи (UTC)
	clock := headerStyle.Render(m.clockTime.UTC().Format("15:04:05 UTC"))

	// Баланс
	balance := headerStyle.Render(fmt.Sprintf("💰 %s", m.balance))

	// Индикатор связи
	var indicator string
	if m.connected {
		indicator = headerStyle.Render(activeIndicator.Render("● LIVE"))
	} else {
		indicator = headerStyle.Render(inactiveIndicator.Render("● OFF"))
	}

	// Заполняем пространство между элементами
	usedW := lipgloss.Width(left) + lipgloss.Width(clock) +
		lipgloss.Width(balance) + lipgloss.Width(indicator)
	gap := m.width - usedW
	if gap < 0 {
		gap = 0
	}
	spacer := headerStyle.Render(strings.Repeat(" ", gap))

	return lipgloss.JoinHorizontal(lipgloss.Top,
		left, spacer, clock, balance, indicator)
}

// renderMain — основной контент + rightbar логи
func (m Model) renderMain() string {
	rightbarW := 35
	contentW := m.width - rightbarW - 4

	content := contentStyle.
		Width(contentW).
		Height(m.height - 6).
		Render(m.contentView.View())

	logs := logStyle.
		Width(rightbarW).
		Height(m.height - 6).
		Render(m.logView.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, content, logs)
}

// renderFooter — командная строка
func (m Model) renderFooter() string {
	prompt := footerStyle.Width(m.width).Render(
		fmt.Sprintf("❯ %s", m.input.View()),
	)
	return prompt
}
