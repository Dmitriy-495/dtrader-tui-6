package indicators

import "strconv"

// OrderBookLevel — один уровень стакана (цена + размер), формат снят
// с реального прода (канал "orderbook", см. лог TUI шаг 1). Оба
// поля — строки в исходном JSON (биржевой API отдаёт их так, чтобы
// не терять точность через float), парсятся через Price()/Size().
type OrderBookLevel struct {
	P string `json:"p"`
	S string `json:"s"`
}

// Price парсит цену уровня. Возвращает 0 при ошибке разбора — вызывающий
// код (см. tui.bestPrice) не должен падать на единичном кривом
// значении, а просто пропустить/показать n/a для него.
func (l OrderBookLevel) Price() float64 {
	v, _ := strconv.ParseFloat(l.P, 64)
	return v
}

// Size парсит объём уровня, тот же принцип, что Price (0 при ошибке
// разбора, не паника — вызывающий код решает, что делать с нулём).
func (l OrderBookLevel) Size() float64 {
	v, _ := strconv.ParseFloat(l.S, 64)
	return v
}

// OrderBook — снапшот стакана для одного символа. Asks (a) и Bids (b)
// — уже отсортированы биржей так, что первый элемент — лучшая цена
// (ask по возрастанию, bid по убыванию, см. реальные данные с прода:
// a[0].p < a[1].p < ..., b[0].p > b[1].p > ...) — TUI на это
// полагается (BestAsk/BestBid берут просто [0]-й элемент), не
// пересортировывает сам.
type OrderBook struct {
	Asks   []OrderBookLevel `json:"a"`
	Bids   []OrderBookLevel `json:"b"`
	Symbol string           `json:"s"`
	Ts     int64            `json:"t"`
}

// BestAsk возвращает лучшую (минимальную) цену продажи, ok=false если
// стакан пуст с этой стороны (случай, который стакан теоретически
// может прислать, например в момент экстремальной волатильности —
// не должен приводить к панике или показу нуля как настоящей цены).
func (ob OrderBook) BestAsk() (price float64, ok bool) {
	if len(ob.Asks) == 0 {
		return 0, false
	}
	return ob.Asks[0].Price(), true
}

// BestBid возвращает лучшую (максимальную) цену покупки, тот же
// принцип ok=false, что BestAsk.
func (ob OrderBook) BestBid() (price float64, ok bool) {
	if len(ob.Bids) == 0 {
		return 0, false
	}
	return ob.Bids[0].Price(), true
}
