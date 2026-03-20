package news

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"time"
)

const (
	rssURL   = "https://ru.cointelegraph.com/rss"
	interval = 5 * time.Minute
)

// NewsItem — одна новость
type NewsItem struct {
	Title       string
	PublishedAt time.Time
	Votes       struct {
		Positive int
		Negative int
	}
}

// rssItem — XML структура одной новости
type rssItem struct {
	Title   string `xml:"title"`
	PubDate string `xml:"pubDate"`
}

// rssFeed — XML структура RSS ленты
type rssFeed struct {
	Items []rssItem `xml:"channel>item"`
}

// Client — RSS клиент Cointelegraph
type Client struct {
	UpdateCh chan []NewsItem
}

func New(_ string) *Client {
	return &Client{
		UpdateCh: make(chan []NewsItem, 10),
	}
}

func (c *Client) Start() {
	go func() {
		c.fetch()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			c.fetch()
		}
	}()
}

func (c *Client) fetch() {
	resp, err := http.Get(rssURL)
	if err != nil {
		fmt.Printf("⚠️ RSS ошибка: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var feed rssFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		fmt.Printf("⚠️ RSS parse ошибка: %v\n", err)
		return
	}

	var items []NewsItem
	for _, item := range feed.Items {
		t, _ := time.Parse(time.RFC1123Z, item.PubDate)
		items = append(items, NewsItem{
			Title:       item.Title,
			PublishedAt: t,
		})
		if len(items) >= 8 {
			break
		}
	}

	select {
	case c.UpdateCh <- items:
	default:
		<-c.UpdateCh
		c.UpdateCh <- items
	}
}
