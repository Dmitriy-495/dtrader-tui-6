# DTrader TUI 6 — Подробный чекпоинт (2026-03-26)

## Репозитории
- **TUI**: github.com/Dmitriy-495/dtrader-tui-6 (ветка main) — ПУБЛИЧНЫЙ
- **Bot+WS**: github.com/Dmitriy-495/dtrader-6 (ветка master)

## Окружение

### Локальные машины (разработка)
- OS: Ubuntu 22, zsh, Go 1.22.3
- Терминал: Kitty (установлен, настроен)
- Путь: /home/tda/code/dtrader/dtrader-tui-6
- Запуск TUI: алиас `tui` в ~/.zshrc
```bash
  alias tui='cd ~/code/dtrader/dtrader-tui-6 && go build -o ./bin/tui ./cmd/main.go && ./bin/tui'
```

### VDS (продакшн) — vm-tda495
- OS: Ubuntu 22.04.5 LTS
- Go 1.22.3, Redis 6.0.16, PostgreSQL 14.22
- IP: 88.218.67.93
- Путь: /home/tda495/code/dtrader/dtrader-6
- Хостинг: console.cloud.ru

## Архитектура системы
```
Gate.io Futures WebSocket (Сингапур, ~220ms RTT)
        ↓
   [dtrader-bot] systemd: dtrader-bot.service
   - Подключается к Gate.io futures WebSocket
   - Получает: trades, orderbook, candles, liquidations, stats, balance
   - Считает EMA латентности (α=2/101) ping/pong каждые 10s
   - Пишет всё в Redis
        ↓ Redis (localhost:6379)
   [dtrader-ws-server] systemd: dtrader-ws.service  
   - Читает Redis streams и keys
   - Агрегирует trades каждые 500ms
   - Отправляет heartbeat каждые 10s (system канал)
   - Раздаёт данные по WebSocket на порту 9000
   - Аутентификация по X-API-Key заголовку
        ↓ WebSocket ws://88.218.67.93:9000/ws
   [dtrader-tui-6] — локальный TUI клиент
   - Подключается к ws-server
   - Показывает данные в реальном времени
   - Получает новости с Cointelegraph RU RSS каждые 5 мин
```

## Управление сервисами на VDS
```bash
# Статус
sudo systemctl status dtrader-bot dtrader-ws dtrader-watcher

# Перезапуск
sudo systemctl restart dtrader-bot dtrader-ws dtrader-watcher

# Логи в реальном времени
sudo journalctl -u dtrader-bot -f
sudo journalctl -u dtrader-ws -f

# Деплой (с локалки)
cd ~/code/dtrader/dtrader-6
./deploy.sh "commit message"
# watcher.sh на VDS подхватывает через ~30s и перезапускает нужный сервис
```

## Redis схема (VDS)
| Ключ | Тип | TTL | Содержимое |
|---|---|---|---|
| `market:trades:{symbol}` | Stream | — | тики сделок {price, size, ts} |
| `market:orderbook:{symbol}` | String | — | JSON снапшот стакана |
| `market:candles:1m:{symbol}` | List | — | закрытые свечи (макс 200) |
| `market:liquidations:{symbol}` | Stream | — | ликвидации |
| `market:stats:{symbol}` | String | — | JSON {lsr_taker, open_interest_usd, ...} |
| `system:exchange_ping` | String | 60s | JSON {"current":222,"ema":288} — RTT биржи |
| `account:balance` | String | — | JSON {"total":"25.27","margin":"0","leverage":"3"} |

## Протокол ws-server → TUI (JSON через WebSocket)
```json
// Каждые 10 секунд — heartbeat
{"channel":"system","symbol":"","data":{
  "server_ts": 1773359082497,
  "exchange_ping": {"current": 222, "ema": 288},
  "balance": {"total":"25.27","margin":"0","leverage":"3"}
}}

// Каждые 500ms — агрегированные трейды
{"channel":"trades","symbol":"BTC_USDT","data":{
  "buy_vol": 1234.5, "sell_vol": 987.3,
  "buy_count": 15, "sell_count": 12,
  "last_price": "70500.5", "ts": 1773359082497
}}

// При изменении — статистика
{"channel":"stats","symbol":"BTC_USDT","data":{
  "lsr_taker": 1.25, "open_interest_usd": 4250000000
}}

// При появлении — ликвидации
{"channel":"liquidations","symbol":"BTC_USDT","data":{...}}

// При закрытии свечи — 1m candle
{"channel":"candles","symbol":"BTC_USDT","data":{...}}
```

## Структура TUI проекта
```
dtrader-tui-6/
├── cmd/main.go                    — точка входа, запуск news+ws+tui
├── internal/
│   ├── config/config.go           — загрузка .env
│   ├── news/client.go             — RSS клиент Cointelegraph RU
│   ├── ws/client.go               — WebSocket клиент с автореконнектом
│   └── ui/
│       ├── app.go                 — главная Model + Update + View
│       ├── styles.go              — ВСЕ стили и цвета
│       ├── header.go              — компонент header
│       ├── footer.go              — компонент footer  
│       ├── layout.go              — renderMain (НЕЗАВЕРШЕНО)
│       ├── tabs.go                — powerline вкладки
│       ├── news.go                — RSS лента новостей
│       ├── rightbar.go            — стили rightbar
│       ├── sidebar.go             — addLog()
│       ├── settings.go            — иконка ⚙ (заглушка)
│       └── screens/
│           ├── dashboard.go       — 📊 сводная таблица пар
│           └── pair.go            — детальный экран пары
├── .env                           — секреты (не в git!)
├── go.mod
└── CHECKPOINT.md
```

## Дизайн-система
- **Фирменный цвет**: оранжевый `lipgloss.Color("214")`
- **Все рамки**: оранжевые `colorBorder = "214"`
- **Статусные цвета**: green="82" (OK), yellow="226" (WARNING), red="196" (SOS/OFF)
- **Текст**: white="255" (важное), orange="214" (данные), gray="239" (вспомогательное)
- **Новости**: синий `lipgloss.Color("39")`

## Header (3 строки с рамкой)
```
╭─────────────────────────────────────────────────────────────────────────────────╮
│ ⚡ DTrader 6   09:19:05 UTC   💰 $25.27 USDT   ↑+$0.17   ↑+$2.43   ● SERV 15ms  ● EXCH 222ms (avg 288ms)  ⚙ │
╰─────────────────────────────────────────────────────────────────────────────────╯
```

## Layout структура (НЕЗАВЕРШЕНО — главная задача!)
```
╭── header ────────────────────────────────────────────────────────────────────╮
│ ⚡ DTrader 6  |  время  |  баланс  |  PnL  |  SERV  |  EXCH  |  ⚙          │
╰──────────────────────────────────────────────────────────────────────────────╯
[ Dashboard ][ ₿ BTC_USDT ][ Ξ ETH_USDT ][ ◎ SOL_USDT ]────────────────────╮
│                                                        │ 📋 Logs            │
│  content (infoBox без верхней рамки)                   │ 09:19:05 💹 BTC    │
│                                                        │ ...                │
│                                                        ├────────────────────┤
│                                                        │ 📈 Positions       │
│                                                        │ нет позиций        │
╰────────────────────────────────────────────────────────╯────────────────────╯
╭── News ─────────────────────────────────────────────────────────────────────╮
│ 1h ● Биткоин...    2h ● Ethereum...                                         │
╰─────────────────────────────────────────────────────────────────────────────╯
❯ введите команду...
```

## Текущие проблемы layout (нужно исправить в новом чате!)
1. Правые границы Logs и Positions не видны (уходят за экран)
2. Tabs на 1 символ короче чем News
3. Разрыв между tabs и rightbar на 1 символ
4. News не прилегает к footer (разрыв 1 строка)
5. Header иногда пропадает при изменении высоты

## Формулы высот (текущие, требуют отладки)
```go
mainH   = m.height - 5
totalW  = m.width - 2        // рабочая ширина
rightW  = totalW * 20 / 100  // 20% под rightbar
leftW   = totalW - rightW    // 80% под content
newsH   = 8 + 2 = 10         // news с border
infoH   = mainH - 2 - newsH  // content без tabs
vpH     = infoH - 2          // viewport внутри infoBox
rightbar total = mainH - 2 - 4
```

## .env (на каждой машине создавать вручную!)
```
WS_SERVER_URL=ws://88.218.67.93:9000/ws
WS_API_KEY=dtrader6_ws_secret
CRYPTOPANIC_API_KEY=79f2be56e48ea3978d8992bcd57791c14554a505
```

## Горячие клавиши TUI
| Клавиша | Действие |
|---|---|
| `Tab` | следующая вкладка |
| `Shift+Tab` | предыдущая вкладка |
| `Ctrl+1..5` | прямой переход к вкладке |
| `Ctrl+C` | выход |

## Стратегия TVP-Sniper (будущее)
- **T** — мульти таймфреймы (1m, 8m, 24m)
- **V** — объёмы (рост)
- **P** — давление в стакане
- **Sniper** — точный вход. 200ms латентность некритична для 1m свечей.

## PENDING задачи (приоритет)
1. **СЕЙЧАС** — исправить layout (правые borders, выравнивание ±1 символ)
2. Dashboard — сброс накопленных объёмов каждую минуту (сейчас накапливаются бесконечно)
3. Детальный экран пары — стакан, свечи, ликвидации  
4. PnL реальный (сейчас заглушка +$0.17)
5. indicator-engine микросервис (Redis → индикаторы)
6. Стратегия TVP-Sniper
7. Закрыть порт 9000 до конкретных IP в console.cloud.ru
