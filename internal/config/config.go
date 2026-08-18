// Пакет config отвечает за загрузку конфигурации TUI из .env.
//
// В отличие от bot/ws-server, у TUI нет config.yaml — по разделу 10
// CHECKPOINT.md dtrader-6 весь конфиг TUI умещается в один .env файл
// (WS_SERVER_URL, WS_API_KEY, CRYPTOPANIC_API_KEY). Никакого отдельного
// YAML не заводим, чтобы не плодить два источника конфигурации там,
// где хватает одного.
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config — вся конфигурация TUI, единственный источник настроек.
// Создаётся один раз в main() и передаётся в модули (ws, news, ui).
type Config struct {
	// WSServerURL — адрес ws-server, например ws://1.2.3.4:9000/ws.
	// Полный URL с путём /ws, а не только host:port — чтобы можно
	// было переключаться между msk/sgp просто сменой .env, без
	// изменений в коде клиента.
	WSServerURL string

	// WSAPIKey — секрет для заголовка X-API-Key при подключении
	// к ws-server (см. ws-server/internal/handler/ws.go: ServeHTTP
	// сравнивает r.Header.Get("X-API-Key") с этим значением).
	WSAPIKey string

	// CryptoPanicAPIKey — ключ для RSS/новостного модуля (раздел 9,
	// internal/news/client.go). В MVP не используется (новостей нет),
	// но поле уже здесь — значит .env для MVP и для будущего полного
	// TUI формально совместимы, менять формат .env при расширении
	// не придётся.
	CryptoPanicAPIKey string
}

// Load загружает .env и собирает Config.
//
// Порядок:
//  1. godotenv.Load() — читает .env в переменные окружения (если файла
//     нет, не паникуем: на деплое секреты могут быть заданы иначе,
//     тем же способом, что bot/config.go игнорирует ошибку godotenv).
//  2. Читаем нужные переменные.
//  3. Валидируем то, что реально нужно MVP прямо сейчас — WSServerURL
//     и WSAPIKey. CryptoPanicAPIKey не валидируем: он не используется
//     в MVP, и требовать его сейчас значило бы блокировать первый
//     запуск TUI ради функциональности, которой ещё нет в коде.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		WSServerURL:       os.Getenv("WS_SERVER_URL"),
		WSAPIKey:          os.Getenv("WS_API_KEY"),
		CryptoPanicAPIKey: os.Getenv("CRYPTOPANIC_API_KEY"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.WSServerURL == "" {
		return fmt.Errorf("WS_SERVER_URL не задан в .env файле")
	}
	if c.WSAPIKey == "" {
		return fmt.Errorf("WS_API_KEY не задан в .env файле")
	}
	return nil
}
