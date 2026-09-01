// Пакет indicators: типы для канала "system" — баланс, статусы,
// список торгуемых пар и открытые позиции (см. ws-server/internal/
// reader/redis.go: SystemMsg).
package indicators

import "strconv"

// Position — одна открытая позиция, формат в точности соответствует
// bot/internal/gateway/rest.go (Gate.io REST Position) — bot публикует
// позиции как есть, без переформатирования (см.
// bot/internal/publisher/redis.go: PublishPositions), ws-server
// ретранслирует их как json.RawMessage (не парсит), так что TUI —
// единственное место, где эта структура реально нужна для отображения.
type Position struct {
	Contract         string `json:"contract"`
	Size             int64  `json:"size"`
	EntryPrice       string `json:"entry_price"`
	MarkPrice        string `json:"mark_price"`
	UnrealisedPnl    string `json:"unrealised_pnl"`
	Margin           string `json:"margin"`
	LiquidationPrice string `json:"liq_price"`
	Leverage         int64  `json:"leverage"`
	Mode             string `json:"mode"`
}

// PnL возвращает UnrealisedPnl как float64 (0, если поле не парсится —
// то же соглашение "тихого нуля при мусорных данных", что уже
// используется в OrderBookLevel.PriceFloat/SizeFloat в этом пакете).
func (p Position) PnL() float64 {
	v, _ := strconv.ParseFloat(p.UnrealisedPnl, 64)
	return v
}

// IsLong возвращает true для длинной позиции (Size > 0), false для
// короткой (Size < 0) — на Gate.io Size со знаком кодирует направление,
// отдельного поля "side" в структуре нет (см. bot/cmd/main.go: тот
// же принцип уже используется там — direction по знаку p.Size).
func (p Position) IsLong() bool {
	return p.Size > 0
}
