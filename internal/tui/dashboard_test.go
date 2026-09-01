package tui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Dmitriy-495/dtrader-tui-6/internal/indicators"
)

func newSymbolModelWithData(t *testing.T, symbol string) Model {
	t.Helper()
	m := New(symbol)

	updated, _ := m.Update(wsMsg{Channel: "indicators", Symbol: symbol, Data: json.RawMessage(realIndicatorsPayload)})
	m = updated.(Model)

	updated, _ = m.Update(wsMsg{Channel: "orderbook", Symbol: symbol, Data: json.RawMessage(realOrderbook20LevelsPayload)})
	m = updated.(Model)

	return m
}

func TestRenderDashboard_NilSystemShowsPlaceholder(t *testing.T) {
	out := renderDashboard(nil, nil, 100)
	if !strings.Contains(out, "ожидание списка торгуемых пар") {
		t.Errorf("nil system должен показывать плейсхолдер, получено: %q", out)
	}
}

func TestRenderDashboard_ShowsAllThreeTimeframesPerSymbol(t *testing.T) {
	// Решение из чата: "разбивка блоков каждого символа на три строки,
	// где каждая — инфа по 1-8-24 мин".
	sys := &indicators.SystemMsg{Symbols: []string{"BTC_USDT"}}
	models := map[string]Model{
		"BTC_USDT": newSymbolModelWithData(t, "BTC_USDT"),
	}

	out := renderDashboard(sys, models, 120)

	for _, want := range []string{"BTC_USDT", "1m", "8m", "24m", "mid:", "pressure:"} {
		if !strings.Contains(out, want) {
			t.Errorf("renderDashboard() должен содержать %q, получено:\n%s", want, out)
		}
	}
}

func TestRenderDashboard_MultipleSymbolsSeparatedByDivider(t *testing.T) {
	sys := &indicators.SystemMsg{Symbols: []string{"BTC_USDT", "ETH_USDT"}}
	models := map[string]Model{
		"BTC_USDT": newSymbolModelWithData(t, "BTC_USDT"),
		"ETH_USDT": newSymbolModelWithData(t, "ETH_USDT"),
	}

	out := renderDashboard(sys, models, 120)

	if !strings.Contains(out, "BTC_USDT") || !strings.Contains(out, "ETH_USDT") {
		t.Errorf("должны быть видны оба символа, получено:\n%s", out)
	}
	if !strings.Contains(out, "─") {
		t.Error("между блоками символов должен быть разделитель")
	}
}

func TestRenderDashboardBlock_NoSnapshotShowsWaiting(t *testing.T) {
	// Символ уже есть во вкладках (появился по symbols), но indicators
	// для него ещё не пришли — валидное состояние, не ошибка.
	m := New("BTC_USDT")
	out := renderDashboardBlock("BTC_USDT", m, 100)

	if !strings.Contains(out, "BTC_USDT") {
		t.Error("заголовок блока должен быть виден даже без snapshot")
	}
	if !strings.Contains(out, "ожидание данных indicators") {
		t.Errorf("должен быть виден плейсхолдер ожидания, получено: %q", out)
	}
}

func TestRenderDashboardHeader_ShowsMidPriceWhenOrderbookAvailable(t *testing.T) {
	m := newSymbolModelWithData(t, "BTC_USDT")
	out := renderDashboardHeader("BTC_USDT", m)

	if strings.Contains(out, "mid: n/a") {
		t.Errorf("mid price должен быть посчитан при наличии orderbook, получено: %q", out)
	}
}

func TestRenderDashboardHeader_ShowsNAWithoutOrderbook(t *testing.T) {
	m := New("BTC_USDT") // без orderbook
	out := renderDashboardHeader("BTC_USDT", m)

	if !strings.Contains(out, "mid: n/a") {
		t.Errorf("без orderbook mid price должен быть n/a, получено: %q", out)
	}
}
