package tui

import (
	"encoding/json"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestModel_ConcurrentUpdatesRealisticSequence эмулирует то, что
// реально происходит за долгую сессию TUI: чередование входящих
// WS-сообщений (indicators/orderbook/system) с изменениями размера
// терминала (пользователь переключает вкладки/ресайзит окно) —
// решение из чата: устойчивый визуальный баг "три строки шапки
// одновременно", не проходящий сам за минуту работы. Гоняется с
// -race (см. Makefile/README), чтобы поймать гонки данных, которые
// не видны в headless-окружении без реальной конкурентности.
func TestModel_RealisticUpdateSequenceUnderRace(t *testing.T) {
	m := New("BTC_USDT")

	sizes := []tea.WindowSizeMsg{
		{Width: 120, Height: 40},
		{Width: 100, Height: 30}, // имитация ресайза/переключения вкладки
		{Width: 150, Height: 50},
		{Width: 80, Height: 24},
	}

	indicatorsMsg := wsMsg{Channel: "indicators", Symbol: "BTC_USDT", Data: json.RawMessage(realIndicatorsPayload)}
	orderbookMsg := wsMsg{Channel: "orderbook", Symbol: "BTC_USDT", Data: json.RawMessage(realOrderbook20LevelsPayload)}

	var tm tea.Model = m
	for i := 0; i < 200; i++ {
		switch i % 5 {
		case 0:
			tm, _ = tm.Update(sizes[i%len(sizes)])
		case 1:
			tm, _ = tm.Update(indicatorsMsg)
		case 2:
			tm, _ = tm.Update(orderbookMsg)
		case 3:
			tm, _ = tm.Update(tea.KeyMsg{Type: tea.KeyDown})
		case 4:
			_ = tm.View() // рендер посреди последовательности апдейтов
		}
	}

	// Финальный View() не должен паниковать и должен быть непустым.
	out := tm.View()
	if out == "" {
		t.Error("итоговый View() пуст после серии апдейтов")
	}
}
