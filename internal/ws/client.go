package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// Message — входящее сообщение от ws-server
type Message struct {
	Channel string          `json:"channel"`
	Symbol  string          `json:"symbol"`
	Data    json.RawMessage `json:"data"`
}

// Client — WS клиент для подключения к ws-server
type Client struct {
	url    string
	apiKey string
	conn   *websocket.Conn
	MsgCh  chan Message // канал входящих сообщений → TUI
}

func New(url, apiKey string) *Client {
	return &Client{
		url:    url,
		apiKey: apiKey,
		MsgCh:  make(chan Message, 256),
	}
}

// Connect подключается к ws-server с автореконнектом
func (c *Client) Connect() {
	for {
		if err := c.connect(); err != nil {
			log.Printf("❌ WS connect error: %v — повтор через 5s", err)
			time.Sleep(5 * time.Second)
			continue
		}
		c.readLoop()
		log.Println("🔄 WS реконнект через 5s...")
		time.Sleep(5 * time.Second)
	}
}

func (c *Client) connect() error {
	header := http.Header{
		"X-API-Key": []string{c.apiKey},
	}
	conn, _, err := websocket.DefaultDialer.Dial(c.url, header)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *Client) readLoop() {
	defer c.conn.Close()
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("⚠️ WS read error: %v", err)
			return
		}
		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		select {
		case c.MsgCh <- msg:
		default:
		}
	}
}
