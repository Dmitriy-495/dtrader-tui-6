// Пакет indicators описывает формат канала "indicators" от ws-server —
// объединённый T/V/P снапшот для одного символа (см.
// ws-server/internal/reader/redis.go: IndicatorsMsg).
//
// Поля и их набор сняты с реального прода (см. первый успешный запуск
// TUI шаг 1, 2026-08-17 23:33 MSK), а не из документации — так что
// расхождений между "что задокументировано" и "что реально приходит"
// здесь по построению быть не должно.
package indicators

// Trend — T-индикатор на одном таймфрейме.
type Trend struct {
	EMAFast       float64 `json:"ema_fast"`
	EMASlow       float64 `json:"ema_slow"`
	Direction     string  `json:"direction"` // "up" / "down" / "neutral"
	Angle         float64 `json:"angle"`
	RSI           float64 `json:"rsi"`
	MACDHistogram float64 `json:"macd_histogram"`
	Ts            int64   `json:"ts"`
}

// Volume — V-индикатор на одном таймфрейме.
type Volume struct {
	BuyVol  float64 `json:"buy_vol"`
	SellVol float64 `json:"sell_vol"`
	Delta   float64 `json:"delta"`
	Spike   bool    `json:"spike"`
	Ts      int64   `json:"ts"`
}

// Pressure — P-индикатор, единый на весь символ (без разбивки по ТФ).
type Pressure struct {
	BidVol    float64 `json:"bid_vol"`
	AskVol    float64 `json:"ask_vol"`
	Imbalance float64 `json:"imbalance"`
	Ts        int64   `json:"ts"`
}

// Snapshot — объединённый T/V/P снапшот одного символа, ровно то, что
// приходит в data канала "indicators".
//
// Trend/Volume — map по таймфрейму ("1m"/"8m"/"24m"), а не фиксированные
// поля структуры: набор таймфреймов определяется analyzer-ом (ws-server
// сам берёт список из своей константы indicatorTimeframes), TUI не
// должен захардкоживать ровно три конкретных ключа на случай, если
// список изменится — достаточно итерировать по map с уже известным,
// но не жёстко зашитым здесь порядком отображения.
type Snapshot struct {
	Trend    map[string]Trend  `json:"trend"`
	Volume   map[string]Volume `json:"volume"`
	Pressure Pressure          `json:"pressure"`
}

// Timeframes — порядок отображения таймфреймов в таблице TUI.
// Захардкожен здесь, в TUI, отдельно от того же списка в ws-server —
// это порядок ПРЕДСТАВЛЕНИЯ (UI-забота), а не источник истины о том,
// какие ТФ реально считаются (это ws-server/analyzer). Если Snapshot
// придёт с ТФ, которого нет в этом списке, View должен показать и его
// тоже (см. tui.timeframesToShow) — список ниже задаёт порядок для
// известных ТФ, а не фильтр.
var Timeframes = []string{"1m", "8m", "24m"}
