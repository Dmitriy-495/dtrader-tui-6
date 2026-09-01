package tui

import (
	"strings"
	"testing"

	"github.com/Dmitriy-495/dtrader-tui-6/internal/indicators"
)

func TestRenderPositions_EmptyShowsPlaceholder(t *testing.T) {
	out := renderPositions(nil)
	if !strings.Contains(out, "нет открытых позиций") {
		t.Errorf("пустой список позиций должен показывать плейсхолдер, получено: %q", out)
	}
}

func TestRenderPositions_ShowsLongAndShort(t *testing.T) {
	positions := []indicators.Position{
		{Contract: "BTC_USDT", Size: 1, UnrealisedPnl: "12.5"},
		{Contract: "ETH_USDT", Size: -1, UnrealisedPnl: "-3.2"},
	}
	out := renderPositions(positions)
	if !strings.Contains(out, "BTC_USDT") || !strings.Contains(out, "LONG") {
		t.Errorf("должна быть видна LONG позиция BTC_USDT, получено: %q", out)
	}
	if !strings.Contains(out, "ETH_USDT") || !strings.Contains(out, "SHORT") {
		t.Errorf("должна быть видна SHORT позиция ETH_USDT, получено: %q", out)
	}
}

func TestRenderLogs_EmptyShowsPlaceholder(t *testing.T) {
	out := renderLogs(nil)
	if !strings.Contains(out, "пока пусто") {
		t.Errorf("пустой лог должен показывать плейсхолдер, получено: %q", out)
	}
}

func TestRenderLogs_ShowsAllEntries(t *testing.T) {
	logs := []LogEntry{
		{Time: "12:00:00", Text: "подключено", Level: LogInfo},
		{Time: "12:00:05", Text: "ошибка сети", Level: LogError},
	}
	out := renderLogs(logs)
	if !strings.Contains(out, "подключено") || !strings.Contains(out, "ошибка сети") {
		t.Errorf("должны быть видны обе записи лога, получено: %q", out)
	}
}

func TestRenderLogsContent_DoesNotPanicOnEmptyInput(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("renderLogsContent запаниковал: %v", r)
		}
	}()
	renderLogsContent(nil)
}

func TestRenderPositionsBlock_DoesNotPanicOnSmallWidth(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("renderPositionsBlock запаниковал при маленькой ширине: %v", r)
		}
	}()
	renderPositionsBlock(nil, 5, 5)
}
