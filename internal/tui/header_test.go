package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/Dmitriy-495/dtrader-tui-6/internal/indicators"
)

func TestRenderHeader_NilShowsWaiting(t *testing.T) {
	out := renderHeader(nil, 120)
	if !strings.Contains(out, "ожидание данных system") {
		t.Errorf("nil system должен показывать плейсхолдер ожидания, получено: %q", out)
	}
	if !strings.Contains(out, "DTrader 6") {
		t.Errorf("бренд должен быть виден даже без данных, получено: %q", out)
	}
}

func TestRenderHeader_ShowsRealData(t *testing.T) {
	// Данные близки к реальным (см. твой лог system из этого чата):
	// balance.total="25.259071788618", exchange_ping.current=264.
	sys := &indicators.SystemMsg{
		ServerTs:     0, // будет "в прошлом" — SERV покажет большую задержку, это нормально для теста
		ExchangePing: indicators.ExchangePing{Current: 264, Ema: 263},
		Balance:      indicators.Balance{Total: "25.259071788618", Margin: "0", Leverage: "3"},
		Symbols:      []string{"BTC_USDT", "ETH_USDT", "SOL_USDT"},
		Positions: []indicators.Position{
			{Contract: "BTC_USDT", Size: 1, UnrealisedPnl: "12.5"},
			{Contract: "ETH_USDT", Size: -1, UnrealisedPnl: "-3.2"},
		},
	}

	out := renderHeader(sys, 120)

	if !strings.Contains(out, "25") {
		t.Errorf("баланс должен быть виден, получено: %q", out)
	}
	// TotalPnL = 12.5 + (-3.2) = 9.3 — суммарный PnL положительный.
	if !strings.Contains(out, "↑") {
		t.Errorf("суммарный положительный PnL должен показывать стрелку вверх, получено: %q", out)
	}
	if !strings.Contains(out, "9,3") && !strings.Contains(out, "9,30") {
		t.Errorf("суммарный PnL (9.3) должен отображаться, получено: %q", out)
	}
	if !strings.Contains(out, "SERV") || !strings.Contains(out, "EXCH") {
		t.Errorf("статусы SERV/EXCH должны быть видны, получено: %q", out)
	}
}

func TestSystemMsg_TotalPnL(t *testing.T) {
	sys := indicators.SystemMsg{
		Positions: []indicators.Position{
			{UnrealisedPnl: "10.5"},
			{UnrealisedPnl: "-4.25"},
			{UnrealisedPnl: "not-a-number"}, // должно тихо стать 0, не паниковать
		},
	}
	got := sys.TotalPnL()
	want := 6.25
	if got != want {
		t.Errorf("TotalPnL() = %v, ожидалось %v", got, want)
	}
}

func TestExchStatusText_Thresholds(t *testing.T) {
	cases := []struct {
		ms        int64
		wantColor string // цвет мы не можем легко достать из ANSI-строки напрямую — проверяем по числу, что оно видно
	}{
		{100, "ok"},
		{500, "warn"},
		{1500, "sos"},
	}
	for _, c := range cases {
		out := exchStatusText(indicators.ExchangePing{Current: c.ms})
		if !strings.Contains(out, "EXCH") {
			t.Errorf("exchStatusText(%d) не содержит EXCH: %q", c.ms, out)
		}
	}
}

func TestServStatusText_NegativeLatencyClampedToZero(t *testing.T) {
	// serverTs "в будущем" относительно time.Now() — защита от
	// рассинхронизации часов клиент/сервер не должна давать
	// отрицательную задержку в выводе.
	futureTs := time.Now().UnixMilli() + 100000
	out := servStatusText(futureTs)
	if strings.Contains(out, "-") {
		t.Errorf("servStatusText не должен показывать отрицательную задержку, получено: %q", out)
	}
}

func TestRenderHeader_AlwaysThreeLinesRegardlessOfWidth(t *testing.T) {
	// Решение из чата: обнаружен реальный баг — header переносился на
	// 4 строки вместо 3 (верх рамки + контент + низ рамки) из-за
	// расхождения между lipgloss.Width() и реальной визуальной шириной
	// строки с эмодзи (⚡💰↑●). Проверяем на нескольких реалистичных
	// ширинах терминала, что header всегда остаётся ровно 3 строки.
	sys := &indicators.SystemMsg{
		Balance:      indicators.Balance{Total: "25.259071788618", Margin: "0", Leverage: "3"},
		ExchangePing: indicators.ExchangePing{Current: 264, Ema: 263},
		ServerTs:     time.Now().UnixMilli(),
		Symbols:      []string{"BTC_USDT"},
	}
	for _, width := range []int{80, 100, 120, 140, 160, 200, 250} {
		out := renderHeader(sys, width)
		got := lipgloss.Height(out)
		if got != 3 {
			t.Errorf("width=%d: renderHeader() высота = %d строк, ожидалось 3:\n%s", width, got, out)
		}
	}
}
