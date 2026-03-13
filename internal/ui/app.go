package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Dmitriy-495/dtrader-tui-6/internal/ws"
)

// =============================================================================
// Стили — все цвета и отступы в одном месте
// =============================================================================
var (
	headerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("255")).
			Bold(true)

	liveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)

	offStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	exchFreshStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true)

	exchStaleStyle = lipgloss.NewStyle().
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

// =============================================================================
// Сообщения bubbletea
// =============================================================================

type tickMsg time.Time
type wsMsg ws.Message

// SystemMsg — структура system канала от ws-server
type SystemMsg struct {
	ServerTs   int64 `json:"server_ts"`
	ExchangeLatMs int64 `json:"exchange_lat_ms"`
	Balance    struct {
		Total    string `json:"total"`
		Margin   string `json:"margin"`
		Leverage string `json:"leverage"`
	} `json:"balance"`
}

// =============================================================================
// Model
// =============================================================================
type Model struct {
	width  int
	height int

	clockTime time.Time
	balance   string
	servMs    int64
	exchLatMs int64
	connected bool

	logs    []string
	logView viewport.Model

	contentView viewport.Model

	input textinput.Model
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

// =============================================================================
// Init
// =============================================================================
func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), waitForMsg(m.msgCh))
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func waitForMsg(ch <-chan ws.Message) tea.Cmd {
	return func() tea.Msg {
		return wsMsg(<-ch)
	}
}

// =============================================================================
// Update
// =============================================================================
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

func (m *Model) handleWS(msg ws.Message) {
	m.connected = true

	switch msg.Channel {
	case "system":
		var sys SystemMsg
		if err := json.Unmarshal(msg.Data, &sys); err != nil {
			return
		}
		m.servMs = time.Now().UnixMilli() - sys.ServerTs
		m.exchLatMs = sys.ExchangeLatMs
		if sys.Balance.Total != "" {
			m.balance = fmt.Sprintf("$%.2f USDT", parseFloat(sys.Balance.Total))
		}
	case "trades":
		m.addLog(fmt.Sprintf("💹 trade %s", msg.Symbol))
	case "stats":
		m.addLog(fmt.Sprintf("📊 stats %s", msg.Symbol))
	case "liquidations":
		m.addLog(fmt.Sprintf("💥 LIQ %s", msg.Symbol))
	case "candles":
		m.addLog(fmt.Sprintf("🕯️ candle %s", msg.Symbol))
	}
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func (m *Model) addLog(entry string) {
	ts := logTimeStyle.Render(time.Now().Format("15:04:05"))
	line := fmt.Sprintf("%s %s", ts, logEntryStyle.Render(entry))
	m.logs = append(m.logs, line)
	if len(m.logs) > 200 {
		m.logs = m.logs[len(m.logs)-200:]
	}
	m.logView.SetContent(strings.Join(m.logs, "\n"))
	m.logView.GotoBottom()
}

func (m *Model) recalcSizes() {
	rightbarW := 35
	contentW := m.width - rightbarW - 4
	contentH := m.height - 1 - 3 - 4 // header + footer + borders

	m.contentView = viewport.New(contentW, contentH)
	m.contentView.SetContent("📊 Dashboard\n\nДанные загружаются...")

	m.logView = viewport.New(rightbarW-4, contentH)
	m.logView.SetContent("")
}

// =============================================================================
// View
// =============================================================================
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

// renderHeader — одна строка на всю ширину:
// [название]   [время UTC]   [баланс]   [● SERV Xms  ● EXCH Xs]
func (m Model) renderHeader() string {
	title   := "⚡ DTrader 6  v0.1"
	clock   := m.clockTime.UTC().Format("15:04:05 UTC")
	balance := fmt.Sprintf("💰 %s", m.balance)

	// SERV — latency между TUI и ws-server
	var serv string
	if !m.connected {
		serv = offStyle.Render("● SERV OFF")
	} else if m.servMs < 100 {
		serv = liveStyle.Render(fmt.Sprintf("● SERV %dms", m.servMs))
	} else {
		serv = exchFreshStyle.Render(fmt.Sprintf("● SERV %dms", m.servMs))
	}

	// EXCH — латентность ping-pong с биржей
	var exch string
	if m.exchLatMs == 0 {
		// нет данных
		exch = offStyle.Render("● EXCH OFF")
	} else if m.exchLatMs < 300 {
		// < 300ms — хорошая задержка, зелёный
		exch = liveStyle.Render(fmt.Sprintf("● EXCH %dms", m.exchLatMs))
	} else if m.exchLatMs < 1000 {
		// 300-1000ms — WARNING, жёлтый
		exch = exchFreshStyle.Render(fmt.Sprintf("● EXCH %dms", m.exchLatMs))
	} else {
		// > 1000ms — SOS, красный
		exch = exchStaleStyle.Render(fmt.Sprintf("● EXCH %dms SOS", m.exchLatMs))
	}

	// Оба индикатора рядом — прижаты к правому краю
	indicators := serv + "  " + exch

	// Считаем пробелы для равномерного распределения 4 блоков
	usedW := len(title) + len(clock) + len(balance) +
		lipgloss.Width(indicators)
	totalGap := m.width - usedW - 2 // -2 небольшой запас
	if totalGap < 4 {
		totalGap = 4
	}
	gap := strings.Repeat(" ", totalGap/3)

	line := title + gap + clock + gap + balance + gap + indicators
	return headerStyle.Width(m.width).Render(line)
}

// renderMain — контент слева + логи справа
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
	return footerStyle.Width(m.width).Render(
		fmt.Sprintf("❯ %s", m.input.View()),
	)
}
