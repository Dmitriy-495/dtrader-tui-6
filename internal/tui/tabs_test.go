package tui

import (
	"strings"
	"testing"
)

func TestTabLabels_DashboardFirst(t *testing.T) {
	labels := tabLabels([]string{"BTC_USDT", "ETH_USDT", "SOL_USDT"})
	want := []string{"Dashboard", "BTC_USDT", "ETH_USDT", "SOL_USDT"}
	if len(labels) != len(want) {
		t.Fatalf("tabLabels() вернул %d меток, ожидалось %d: %v", len(labels), len(want), labels)
	}
	for i, w := range want {
		if labels[i] != w {
			t.Errorf("labels[%d] = %q, ожидалось %q", i, labels[i], w)
		}
	}
}

func TestTabLabels_EmptySymbols_OnlyDashboard(t *testing.T) {
	// Решение из чата: вкладки формируются по symbols из первого
	// system-сообщения — до его прихода symbols пуст, но Dashboard
	// должен быть виден всегда (это не вкладка символа, ей не нужны
	// данные символов для существования).
	labels := tabLabels(nil)
	if len(labels) != 1 || labels[0] != dashboardTabLabel {
		t.Errorf("tabLabels(nil) = %v, ожидался только [%q]", labels, dashboardTabLabel)
	}
}

func TestRenderTabs_ContainsAllLabels(t *testing.T) {
	out := renderTabs([]string{"BTC_USDT", "ETH_USDT"}, 0, 100)
	for _, want := range []string{"Dashboard", "BTC_USDT", "ETH_USDT"} {
		if !strings.Contains(out, want) {
			t.Errorf("renderTabs() должен содержать %q, получено: %q", want, out)
		}
	}
}

func TestRenderTabs_DoesNotPanicOnZeroWidth(t *testing.T) {
	// Защита от паники при некорректном/ещё не установленном размере
	// терминала (например, до первого WindowSizeMsg).
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("renderTabs() запаниковал при width=0: %v", r)
		}
	}()
	renderTabs([]string{"BTC_USDT"}, 0, 0)
}
