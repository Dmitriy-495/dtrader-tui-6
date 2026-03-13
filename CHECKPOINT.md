# DTrader TUI 6 — Чекпоинт v0.4 (2026-03-13)

## Репозиторий
- github.com/Dmitriy-495/dtrader-tui-6 (ветка main)

## Запуск
alias tui='cd ~/code/dtrader/dtrader-tui-6 && go build -o ./bin/tui ./cmd/main.go && ./bin/tui'
tui

## Header
⚡ DTrader 6  v0.4   09:19:05 UTC   💰 $25.27 USDT   ● SERV 15ms  ● EXCH 222ms (avg 288ms)

## Индикаторы
- SERV: зелёный <100ms, жёлтый >=100ms, красный OFF
- EXCH: зелёный <300ms, жёлтый 300-1000ms, красный >=1000ms SOS

## Зависимости
- bubbletea, bubbles, lipgloss
- gorilla/websocket
- godotenv

## PENDING
1. Dashboard — строки по парам — СЛЕДУЮЩИЙ
2. PnL в header
3. Навигация по вкладкам (BTC/ETH/SOL)
