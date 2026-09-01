package indicators

// ExchangePing — латентность до биржи, current + EMA. Точная схема
// с ws-server/internal/reader/redis.go: ExchangePing.
type ExchangePing struct {
	Current int64 `json:"current"`
	Ema     int64 `json:"ema"`
}

// Balance — баланс аккаунта. Точная схема с ws-server: Balance.
type Balance struct {
	Total    string `json:"total"`
	Margin   string `json:"margin"`
	Leverage string `json:"leverage"`
}

// SystemMsg — весь канал "system" целиком (см. ws-server/internal/
// reader/redis.go: SystemMsg). В отличие от ws-server (который
// ретранслирует Positions как []json.RawMessage, не зная их
// структуру), TUI — конечный потребитель, поэтому здесь сразу
// []Position: нам нужно реально отображать позиции (для rightbar/
// header), а не просто передавать их дальше.
type SystemMsg struct {
	ServerTs     int64        `json:"server_ts"`
	ExchangePing ExchangePing `json:"exchange_ping"`
	Balance      Balance      `json:"balance"`
	Symbols      []string     `json:"symbols"`
	Positions    []Position   `json:"positions"`
}

// TotalPnL суммирует нереализованный PnL всех открытых позиций —
// используется в шапке (header.go) для агрегированного показателя
// "PnL↑/↓", который не приходит с сервера готовым значением, а
// вычисляется на стороне TUI из списка позиций.
func (s SystemMsg) TotalPnL() float64 {
	var total float64
	for _, p := range s.Positions {
		total += p.PnL()
	}
	return total
}
