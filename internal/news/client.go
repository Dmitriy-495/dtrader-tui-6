package news

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	apiURL   = "https://cryptopanic.com/api/developer/v2/posts/"
	interval = 5 * time.Minute
)

// NewsItem — одна новость из CryptoPanic
type NewsItem struct {
	Title       string    `json:"title"`
	PublishedAt time.Time `json:"published_at"`
	Source      struct {
		Title string `json:"title"`
	} `json:"source"`
	Votes struct {
		Positive int `json:"positive"`
		Negative int `json:"negative"`
	} `json:"votes"`
}

// response — ответ CryptoPanic API
type response struct {
	Results []NewsItem `json:"results"`
}

// Client — клиент CryptoPanic
type Client struct {
	apiKey   string
	items    []NewsItem
	UpdateCh chan []NewsItem
}

func New(apiKey string) *Client {
	return &Client{
		apiKey:   apiKey,
		UpdateCh: make(chan []NewsItem, 1),
	}
}

// Start — запускает фоновое обновление новостей каждые 5 минут
func (c *Client) Start() {
	go func() {
		c.fetch() // сразу при старте
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			c.fetch()
		}
	}()
}

// fetch — получает свежие новости по BTC/ETH/SOL
func (c *Client) fetch() {
	url := fmt.Sprintf("%s?auth_token=%s&currencies=BTC,ETH,SOL&kind=news&public=true",
		apiURL, c.apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var r response
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return
	}

	// Берём последние 8 новостей
	items := r.Results
	if len(items) > 8 {
		items = items[:8]
	}

	// Отправляем в канал без блокировки
	select {
	case c.UpdateCh <- items:
	default:
		<-c.UpdateCh
		c.UpdateCh <- items
	}
}
