package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/Dmitriy-495/dtrader-tui-6/internal/news"
	"github.com/Dmitriy-495/dtrader-tui-6/internal/ws"
)

const (
	sidebarWidth = 35
	minWidth     = 120
	minHeight    = 30
)

type tickMsg time.Time
type wsMsg ws.Message
type newsMsg []news.NewsItem

type SystemMsg struct {
	ServerTs     int64 `json:"server_ts"`
	ExchangePing struct {
		Current int64 `json:"current"`
		Ema     int64 `json:"ema"`
	} `json:"exchange_ping"`
	Balance struct {
		Total    string `json:"total"`
		Margin   string `json:"margin"`
		Leverage string `json:"leverage"`
	} `json:"balance"`
}

type TradeMsg struct {
	BuyVol    float64 `json:"buy_vol"`
	SellVol   float64 `json:"sell_vol"`
	BuyCount  int     `json:"buy_count"`
	SellCount int     `json:"sell_count"`
	LastPrice string  `json:"last_price"`
	Ts        int64   `json:"ts"`
}

type StatsMsg struct {
	OpenInterest    float64 `json:"open_interest"`
	OpenInterestUSD float64 `json:"open_interest_usd"`
	LsrTaker        float64 `json:"lsr_taker"`
	LsrAccount      float64 `json:"lsr_account"`
}

type Model struct {
	width  int
	height int

	clockTime time.Time
	balance   string
	servMs    int64
	exchCurMs int64
	exchEmaMs int64
	connected bool

	activeTab int
	symbols   []string
	pairs     map[string]*PairData
	newsItems []news.NewsItem

	logs    []string
	logView viewport.Model

	contentView viewport.Model

	input  textinput.Model
	msgCh  <-chan ws.Message
	newsCh <-chan []news.NewsItem
}

func New(msgCh <-chan ws.Message, newsCh <-chan []news.NewsItem) Model {
	input := textinput.New()
	input.Placeholder = "введите команду..."
	input.Focus()
	return Model{
		msgCh:     msgCh,
		newsCh:    newsCh,
		balance:   "загрузка...",
		clockTime: time.Now(),
		input:     input,
		logs:      []string{},
		pairs:     make(map[string]*PairData),
		symbols:   []string{},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), waitForMsg(m.msgCh), waitForNews(m.newsCh))
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func waitForMsg(ch <-chan ws.Message) tea.Cmd {
	return func() tea.Msg { return wsMsg(<-ch) }
}

func waitForNews(ch <-chan []news.NewsItem) tea.Cmd {
	return func() tea.Msg { return newsMsg(<-ch) }
}

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
	case newsMsg:
		m.newsItems = []news.NewsItem(msg)
		cmds = append(cmds, waitForNews(m.newsCh))
	case wsMsg:
		m.handleWS(ws.Message(msg))
		cmds = append(cmds, waitForMsg(m.msgCh))
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.activeTab = (m.activeTab + 1) % (len(m.symbols) + 1)
		case "shift+tab":
			m.activeTab = (m.activeTab - 1 + len(m.symbols) + 1) % (len(m.symbols) + 1)
		case "0":
			m.activeTab = 0
		case "1", "2", "3", "4", "5":
			idx := int(msg.String()[0] - '0')
			if idx <= len(m.symbols) {
				m.activeTab = idx
			}
		case "enter":
			if cmd := m.input.Value(); cmd != "" {
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
		m.exchCurMs = sys.ExchangePing.Current
		m.exchEmaMs = sys.ExchangePing.Ema
		if sys.Balance.Total != "" {
			m.balance = fmt.Sprintf("$%.2f USDT", parseFloat(sys.Balance.Total))
		}
	case "trades":
		var t TradeMsg
		if err := json.Unmarshal(msg.Data, &t); err != nil {
			return
		}
		m.ensurePair(msg.Symbol)
		p := m.pairs[msg.Symbol]
		p.Price = t.LastPrice
		p.BuyVol += t.BuyVol
		p.SellVol += t.SellVol
		m.addLog(fmt.Sprintf("💹 %s %s", msg.Symbol, t.LastPrice))
	case "stats":
		var s StatsMsg
		if err := json.Unmarshal(msg.Data, &s); err != nil {
			return
		}
		m.ensurePair(msg.Symbol)
		p := m.pairs[msg.Symbol]
		p.LSR = s.LsrTaker
		p.OI = s.OpenInterestUSD
		m.addLog(fmt.Sprintf("📊 stats %s", msg.Symbol))
	case "liquidations":
		m.addLog(fmt.Sprintf("💥 LIQ %s", msg.Symbol))
	case "candles":
		m.addLog(fmt.Sprintf("🕯️  candle %s", msg.Symbol))
	}
}

func (m *Model) ensurePair(symbol string) {
	if _, ok := m.pairs[symbol]; !ok {
		m.pairs[symbol] = &PairData{Symbol: symbol}
		m.symbols = append(m.symbols, symbol)
	}
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func joinLines(lines []string) string {
	return strings.Join(lines, "\n")
}

func (m *Model) recalcSizes() {
	rightW   := m.width * rightbarPct / 100
	leftW    := m.width - rightW
	mainH    := m.height - 4
	newsTotal := newsContent + 2
	vpH      := mainH - newsTotal - 3
	if vpH < 1 {
		vpH = 1
	}
	m.contentView = viewport.New(leftW-2, vpH)
	m.logView = viewport.New(rightW-2, mainH*logsPct/100-2)
}

// =============================================================================
// View
// =============================================================================
func (m Model) View() string {
	if m.width == 0 {
		return "загрузка..."
	}
	if m.width < minWidth || m.height < minHeight {
		return fmt.Sprintf(
			"\n\n  ❌ Терминал слишком мал!\n\n"+
				"  Текущий размер:    %d x %d\n"+
				"  Минимальный размер: %d x %d\n\n"+
				"  Увеличьте окно терминала.",
			m.width, m.height, minWidth, minHeight,
		)
	}
	return strings.Join([]string{
		m.renderHeader(),
		m.renderMain(),
		m.renderFooter(),
	}, "\n")
}
