package indicators

import "testing"

func TestOrderBook_MidPrice(t *testing.T) {
	ob := OrderBook{
		Bids: []OrderBookLevel{{P: "100.0", S: "1"}},
		Asks: []OrderBookLevel{{P: "102.0", S: "1"}},
	}
	mid, ok := ob.MidPrice()
	if !ok {
		t.Fatal("MidPrice() ok=false для непустого стакана с обеих сторон")
	}
	if mid != 101.0 {
		t.Errorf("MidPrice() = %v, ожидалось 101.0", mid)
	}
}

func TestOrderBook_MidPrice_EmptyBidsReturnsNotOK(t *testing.T) {
	ob := OrderBook{
		Bids: nil,
		Asks: []OrderBookLevel{{P: "102.0", S: "1"}},
	}
	_, ok := ob.MidPrice()
	if ok {
		t.Error("MidPrice() должен вернуть ok=false при пустых bids")
	}
}

func TestOrderBook_MidPrice_EmptyAsksReturnsNotOK(t *testing.T) {
	ob := OrderBook{
		Bids: []OrderBookLevel{{P: "100.0", S: "1"}},
		Asks: nil,
	}
	_, ok := ob.MidPrice()
	if ok {
		t.Error("MidPrice() должен вернуть ok=false при пустых asks")
	}
}
