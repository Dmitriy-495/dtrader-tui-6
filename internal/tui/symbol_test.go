package tui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Dmitriy-495/dtrader-tui-6/internal/ws"
)

// realIndicatorsPayload — реальный JSON из живого прода (см. лог TUI
// шаг 1, 2026-08-17 23:33 MSK, символ BTC_USDT), а не сконструированный
// вручную — чтобы тест проверял разбор именно того формата, который
// реально приходит, а не идеализированную версию.
const realIndicatorsPayload = `{"trend":{"1m":{"ema_fast":64336.7819116163,"ema_slow":64316.64902869316,"direction":"neutral","angle":79.5419837407385,"rsi":62.968099861304914,"macd_histogram":0,"ts":1786998812534},"24m":{"ema_fast":64227.10070441352,"ema_slow":64217.998080054225,"direction":"neutral","angle":80.36118722179849,"rsi":0,"macd_histogram":0,"ts":1786998812534},"8m":{"ema_fast":64312.13943956376,"ema_slow":64314.06340467373,"direction":"neutral","angle":-78.79546235408986,"rsi":39.48923194206264,"macd_histogram":-6.312523906260582,"ts":1786998812534}},"volume":{"1m":{"buy_vol":17718,"sell_vol":0,"delta":17718,"spike":false,"ts":1786998812534},"24m":{"buy_vol":994399,"sell_vol":683812,"delta":310587,"spike":false,"ts":1786998812534},"8m":{"buy_vol":58279,"sell_vol":68733,"delta":-10454,"spike":false,"ts":1786998812534}},"pressure":{"bid_vol":41053,"ask_vol":79193,"imbalance":0.518391777051002,"ts":1786998812534}}`

func TestUpdate_ParsesRealIndicatorsPayload(t *testing.T) {
	client := ws.New("ws://unused", "unused")
	m := New("BTC_USDT", client)

	msg := wsMsg{
		Channel: "indicators",
		Symbol:  "BTC_USDT",
		Data:    json.RawMessage(realIndicatorsPayload),
	}

	updated, _ := m.Update(msg)
	got := updated.(Model)

	if got.snapshot == nil {
		t.Fatal("snapshot не должен быть nil после успешного разбора")
	}
	if got.lastErr != "" {
		t.Fatalf("lastErr должен быть пустым, получено: %q", got.lastErr)
	}
	if got.snapshot.Trend["1m"].Direction != "neutral" {
		t.Errorf("Trend[1m].Direction = %q, ожидалось neutral", got.snapshot.Trend["1m"].Direction)
	}
	if got.snapshot.Volume["1m"].Delta != 17718 {
		t.Errorf("Volume[1m].Delta = %v, ожидалось 17718", got.snapshot.Volume["1m"].Delta)
	}
	if got.snapshot.Pressure.Imbalance != 0.518391777051002 {
		t.Errorf("Pressure.Imbalance = %v, ожидалось 0.518391777051002", got.snapshot.Pressure.Imbalance)
	}
}

func TestUpdate_IgnoresOtherSymbol(t *testing.T) {
	client := ws.New("ws://unused", "unused")
	m := New("BTC_USDT", client)

	msg := wsMsg{
		Channel: "indicators",
		Symbol:  "ETH_USDT", // другой символ — не наша вкладка
		Data:    json.RawMessage(realIndicatorsPayload),
	}

	updated, _ := m.Update(msg)
	got := updated.(Model)

	if got.snapshot != nil {
		t.Error("snapshot должен остаться nil — сообщение для другого символа не должно применяться")
	}
}

func TestUpdate_IgnoresOtherChannel(t *testing.T) {
	client := ws.New("ws://unused", "unused")
	m := New("BTC_USDT", client)

	msg := wsMsg{
		Channel: "trades", // не indicators
		Symbol:  "BTC_USDT",
		Data:    json.RawMessage(`{"symbol":"BTC_USDT"}`),
	}

	updated, _ := m.Update(msg)
	got := updated.(Model)

	if got.snapshot != nil {
		t.Error("snapshot должен остаться nil — канал trades не должен обрабатываться как indicators")
	}
}

func TestUpdate_HandlesMalformedJSON(t *testing.T) {
	client := ws.New("ws://unused", "unused")
	m := New("BTC_USDT", client)

	msg := wsMsg{
		Channel: "indicators",
		Symbol:  "BTC_USDT",
		Data:    json.RawMessage(`{not valid json`),
	}

	updated, _ := m.Update(msg)
	got := updated.(Model)

	if got.lastErr == "" {
		t.Error("lastErr должен быть заполнен при битом JSON, не молчать об ошибке")
	}
	if got.snapshot != nil {
		t.Error("snapshot не должен обновиться при ошибке разбора")
	}
}

func TestView_RendersWithoutPanicBeforeFirstData(t *testing.T) {
	client := ws.New("ws://unused", "unused")
	m := New("BTC_USDT", client)

	out := m.View()
	if !strings.Contains(out, "BTC_USDT") {
		t.Errorf("View() должен содержать имя символа, получено: %q", out)
	}
	if !strings.Contains(out, "ожидание данных") {
		t.Errorf("View() до получения данных должен показывать статус ожидания, получено: %q", out)
	}
}

func TestView_RendersTableAfterData(t *testing.T) {
	client := ws.New("ws://unused", "unused")
	m := New("BTC_USDT", client)

	msg := wsMsg{
		Channel: "indicators",
		Symbol:  "BTC_USDT",
		Data:    json.RawMessage(realIndicatorsPayload),
	}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	out := m.View()
	for _, want := range []string{"1m", "8m", "24m", "TREND", "ANGLE", "RSI", "EMA", "BUY", "SELL", "P:"} {
		if !strings.Contains(out, want) {
			t.Errorf("View() после данных должен содержать %q, получено:\n%s", want, out)
		}
	}
}

func TestAngleBar_ClampsAndDirection(t *testing.T) {
	// Значение за пределами angleMaxDegrees должно зажиматься
	// (clamp), не растягивать шкалу — иначе, например, angle=500
	// выглядел бы так же "заполненно", как и angle=90, и это
	// нормально (обе экстремальны), но проверяем, что заполнение
	// не превышает ширину бара и не паникует на экстремальных
	// значениях.
	extreme := angleBar(500)
	if strings.Count(extreme, "█") > angleBarWidth {
		t.Errorf("angleBar(500) содержит больше %d закрашенных ячеек: %q", angleBarWidth, extreme)
	}

	neg := angleBar(-45)
	if !strings.Contains(neg, "-45°") {
		t.Errorf("angleBar(-45) должен показывать отрицательное число, получено: %q", neg)
	}

	zero := angleBar(0)
	if strings.Count(zero, "█") != 0 {
		t.Errorf("angleBar(0) не должен иметь закрашенных ячеек, получено: %q", zero)
	}
}

func TestEMASpreadBar_HandlesZeroSlow(t *testing.T) {
	// slow=0 — деление на ноль, редкий, но возможный случай на старте
	// прогрева индикатора (см. реальные данные: rsi иногда 0 в 24m
	// сразу после старта analyzer). Не должно паниковать.
	out := emaSpreadBar(100, 0)
	if !strings.Contains(out, "n/a") {
		t.Errorf("emaSpreadBar с slow=0 должен вернуть n/a, получено: %q", out)
	}
}
