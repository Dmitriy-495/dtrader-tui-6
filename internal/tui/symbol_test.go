package tui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Dmitriy-495/dtrader-tui-6/internal/indicators"
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
	for _, want := range []string{"1m", "8m", "24m", "TREND", "ANGLE", "RSI", "EMA", "VOL Δ%", "PRESSURE"} {
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
	if strings.Count(extreme, "█") > barWidth {
		t.Errorf("angleBar(500) содержит больше %d закрашенных ячеек: %q", barWidth, extreme)
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

func TestBipolarBar_NegativeGoesLeftPositiveGoesRight(t *testing.T) {
	// Ядро решения из чата: "за ноль принять середину, отрицательные
	// значения влево, положительные вправо" — проверяем именно это
	// через возвращаемые filledRight/filledLeft (не через побайтовый
	// разбор отрендеренной строки — там смешаны многобайтовые руны
	// █/░/│, индекс через strings.Index был бы ненадёжен).
	_, right, left := bipolarBar(50, 90)
	if right == 0 {
		t.Error("положительное значение должно заполнять правую сторону (filledRight > 0)")
	}
	if left != 0 {
		t.Error("положительное значение не должно заполнять левую сторону")
	}

	_, right2, left2 := bipolarBar(-50, 90)
	if left2 == 0 {
		t.Error("отрицательное значение должно заполнять левую сторону (filledLeft > 0)")
	}
	if right2 != 0 {
		t.Error("отрицательное значение не должно заполнять правую сторону")
	}

	_, right3, left3 := bipolarBar(0, 90)
	if right3 != 0 || left3 != 0 {
		t.Errorf("нулевое значение не должно заполнять ни одну сторону, right=%d left=%d", right3, left3)
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

func TestFormatNumber(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0,00"},
		{1, "1,00"},
		{12, "12,00"},
		{123, "123,00"},
		{1234, "1 234,00"},
		{12345, "12 345,00"},
		{123456, "123 456,00"},
		{1234567, "1 234 567,00"},
		{994399, "994 399,00"},
		{994399.5, "994 399,50"},
		{-1234.5, "-1 234,50"},
		{0.1, "0,10"},
		{999.995, "1 000,00"}, // округление на границе
	}
	for _, c := range cases {
		got := formatNumber(c.in)
		if got != c.want {
			t.Errorf("formatNumber(%v) = %q, ожидалось %q", c.in, got, c.want)
		}
	}
}

// realOrderbookPayload — реальный orderbook для BTC_USDT из лога
// прода (см. TestUpdate_ParsesRealIndicatorsPayload — тот же сеанс,
// 2026-08-17 23:33 MSK), обрезан до первых 3 уровней с каждой
// стороны — для проверки BestBid/BestAsk достаточно первого уровня,
// остальные не нужны.
const realOrderbookPayload = `{"a":[{"p":"64327.7","s":"40413"},{"p":"64328.7","s":"1855"},{"p":"64329.9","s":"1555"}],"b":[{"p":"64327.6","s":"24469"},{"p":"64327","s":"3518"},{"p":"64326.9","s":"2364"}],"s":"BTC_USDT","t":1786998815289}`

func TestUpdate_ParsesOrderbookAndComputesBestPrices(t *testing.T) {
	client := ws.New("ws://unused", "unused")
	m := New("BTC_USDT", client)

	msg := wsMsg{
		Channel: "orderbook",
		Symbol:  "BTC_USDT",
		Data:    json.RawMessage(realOrderbookPayload),
	}

	updated, _ := m.Update(msg)
	got := updated.(Model)

	if got.orderbook == nil {
		t.Fatal("orderbook не должен быть nil после успешного разбора")
	}

	bid, ok := got.orderbook.BestBid()
	if !ok || bid != 64327.6 {
		t.Errorf("BestBid() = %v, %v — ожидалось 64327.6, true", bid, ok)
	}
	ask, ok := got.orderbook.BestAsk()
	if !ok || ask != 64327.7 {
		t.Errorf("BestAsk() = %v, %v — ожидалось 64327.7, true", ask, ok)
	}

	out := got.View()
	if !strings.Contains(out, "64 327,60") {
		t.Errorf("View() должен содержать отформатированный best bid 64 327,60, получено:\n%s", out)
	}
	if !strings.Contains(out, "64 327,70") {
		t.Errorf("View() должен содержать отформатированный best ask 64 327,70, получено:\n%s", out)
	}
	// statusBadge не должен вызываться в шапке (решение из чата:
	// статус связи переехал в общий header, не на вкладку символа).
	if strings.Contains(out, "connected") || strings.Contains(out, "connecting") {
		t.Errorf("View() не должен содержать индикатор статуса соединения, получено:\n%s", out)
	}
}

func TestOrderBook_EmptySideReturnsFalse(t *testing.T) {
	ob := indicators.OrderBook{Asks: nil, Bids: []indicators.OrderBookLevel{{P: "100", S: "1"}}}
	if _, ok := ob.BestAsk(); ok {
		t.Error("BestAsk() с пустым Asks должен вернуть ok=false")
	}
	if _, ok := ob.BestBid(); !ok {
		t.Error("BestBid() с непустым Bids должен вернуть ok=true")
	}
}

func TestBestPricesText_NilOrderbookShowsWaiting(t *testing.T) {
	out := bestPricesText(nil)
	if !strings.Contains(out, "ожидание") {
		t.Errorf("bestPricesText(nil) должен показывать статус ожидания, получено: %q", out)
	}
}

func TestVolumeDeltaBar_ComputesSymmetricPercent(t *testing.T) {
	// buy=17718, sell=0 (реальные данные 1m из realIndicatorsPayload)
	// → (17718-0)/(17718+0)*100 = +100.0%, весь объём — покупки.
	out := volumeDeltaBar(17718, 0, false)
	if !strings.Contains(out, "+100.0%") {
		t.Errorf("volumeDeltaBar(17718,0) должен показать +100.0%%, получено: %q", out)
	}

	// buy=58279, sell=68733 → (58279-68733)/(58279+68733)*100 ≈ -8.2%
	out2 := volumeDeltaBar(58279, 68733, false)
	if !strings.Contains(out2, "-8.2%") {
		t.Errorf("volumeDeltaBar(58279,68733) должен показать -8.2%%, получено: %q", out2)
	}

	// total=0 — оба нулевые, не должно паниковать/делить на ноль.
	out3 := volumeDeltaBar(0, 0, false)
	if !strings.Contains(out3, "n/a") {
		t.Errorf("volumeDeltaBar(0,0) должен показать n/a, получено: %q", out3)
	}

	// spike=true должен добавлять пометку.
	out4 := volumeDeltaBar(100, 50, true)
	if !strings.Contains(out4, "🔥") {
		t.Error("volumeDeltaBar с spike=true должен содержать пометку 🔥")
	}
}

func TestRenderPressureBlock_ComputesSymmetricPercent(t *testing.T) {
	// bid=41053, ask=79193 (реальные данные) →
	// (41053-79193)/(41053+79193)*100 ≈ -31.7%
	p := indicators.Pressure{BidVol: 41053, AskVol: 79193, Imbalance: 0.518391777051002}
	out := renderPressureBlock(p)
	if !strings.Contains(out, "-31.7%") {
		t.Errorf("renderPressureBlock должен показать -31.7%%, получено: %q", out)
	}
	// Старый коэффициент 0.518 не должен фигурировать как отдельная
	// метрика — заменён процентом полностью, не добавлен рядом с ним.
	if strings.Contains(out, "0.518") {
		t.Errorf("renderPressureBlock не должен содержать старый коэффициент imbalance отдельно от процента, получено: %q", out)
	}
}

// realOrderbook20LevelsPayload — полный 20-уровневый (10+10) стакан
// BTC_USDT с прода (тот же сеанс 2026-08-17 23:33 MSK), нужен для
// проверки полной картины стакана (realOrderbookPayload выше обрезан
// до 3 уровней и годится только для BestBid/BestAsk).
const realOrderbook20LevelsPayload = `{"a":[{"p":"64327.7","s":"40413"},{"p":"64328.7","s":"1855"},{"p":"64329.9","s":"1555"},{"p":"64330.3","s":"89"},{"p":"64331.4","s":"5"},{"p":"64331.5","s":"155"},{"p":"64331.7","s":"2"},{"p":"64331.8","s":"2"},{"p":"64332.4","s":"36"},{"p":"64333.5","s":"310"},{"p":"64334.2","s":"1"},{"p":"64334.4","s":"1"},{"p":"64335.3","s":"594"},{"p":"64335.9","s":"392"},{"p":"64336.1","s":"6"},{"p":"64336.3","s":"2"},{"p":"64337.3","s":"10"},{"p":"64337.6","s":"249"},{"p":"64337.7","s":"10"},{"p":"64337.8","s":"5"}],"b":[{"p":"64327.6","s":"24469"},{"p":"64327","s":"3518"},{"p":"64326.9","s":"2364"},{"p":"64326.5","s":"3852"},{"p":"64326.4","s":"5645"},{"p":"64325.4","s":"1555"},{"p":"64323.2","s":"1270"},{"p":"64322.8","s":"1556"},{"p":"64322.3","s":"1556"},{"p":"64321.4","s":"187"},{"p":"64321.1","s":"1"},{"p":"64321","s":"608"},{"p":"64320.7","s":"156"},{"p":"64320.3","s":"6"},{"p":"64320.2","s":"300"},{"p":"64320","s":"6756"},{"p":"64319.1","s":"36"},{"p":"64319","s":"1556"},{"p":"64318.7","s":"1"},{"p":"64318.3","s":"780"}],"s":"BTC_USDT","t":1786998815289}`

func TestRenderOrderbookColumn_NilShowsWaiting(t *testing.T) {
	out := renderOrderbookColumn(nil)
	if !strings.Contains(out, "ожидание") {
		t.Errorf("renderOrderbookColumn(nil) должен показывать статус ожидания, получено: %q", out)
	}
}

func TestRenderOrderbookColumn_RendersExpectedLineCount(t *testing.T) {
	var ob indicators.OrderBook
	if err := json.Unmarshal([]byte(realOrderbook20LevelsPayload), &ob); err != nil {
		t.Fatalf("не удалось разобрать тестовый orderbook: %v", err)
	}

	out := renderOrderbookColumn(&ob)
	lines := strings.Split(out, "\n")

	// 10 asks + 1 строка спреда + 10 bids = 21 строка (решение из
	// чата: "10 asks сверху + 10 bids снизу").
	want := orderbookLevelsPerSide*2 + 1
	if len(lines) != want {
		t.Errorf("renderOrderbookColumn выдал %d строк, ожидалось %d:\n%s", len(lines), want, out)
	}
}

func TestRenderOrderbookColumn_ContainsBestPricesNearSpread(t *testing.T) {
	var ob indicators.OrderBook
	if err := json.Unmarshal([]byte(realOrderbook20LevelsPayload), &ob); err != nil {
		t.Fatalf("не удалось разобрать тестовый orderbook: %v", err)
	}

	out := renderOrderbookColumn(&ob)
	// best ask (64327.7) должен быть последней ask-строкой перед
	// спредом, best bid (64327.6) — первой bid-строкой после него
	// (порядок из чата: "10 asks сверху + 10 bids снизу").
	if !strings.Contains(out, formatNumber(64327.7)) {
		t.Errorf("вывод должен содержать best ask %s, получено:\n%s", formatNumber(64327.7), out)
	}
	if !strings.Contains(out, formatNumber(64327.6)) {
		t.Errorf("вывод должен содержать best bid %s, получено:\n%s", formatNumber(64327.6), out)
	}
	if !strings.Contains(out, "spread") {
		t.Errorf("вывод должен содержать строку spread, получено:\n%s", out)
	}
}

func TestRenderOrderbookColumn_LimitsToTenLevelsPerSide(t *testing.T) {
	var ob indicators.OrderBook
	if err := json.Unmarshal([]byte(realOrderbook20LevelsPayload), &ob); err != nil {
		t.Fatalf("не удалось разобрать тестовый orderbook: %v", err)
	}
	// В исходных данных ровно 20 уровней с каждой стороны — проверяем,
	// что дальний уровень (20-й ask, самый дальний от спреда) НЕ
	// попадает в вывод, раз лимит — 10 (решение из чата).
	out := renderOrderbookColumn(&ob)
	farAsk := ob.Asks[orderbookLevelsPerSide].Price() // 11-й уровень, за пределами лимита 10
	if strings.Contains(out, formatNumber(farAsk)) {
		t.Errorf("вывод не должен содержать цену за пределами лимита %d уровней (%s):\n%s",
			orderbookLevelsPerSide, formatNumber(farAsk), out)
	}
}
