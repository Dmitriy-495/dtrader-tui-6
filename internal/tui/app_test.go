package tui

import (
	"encoding/json"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dmitriy-495/dtrader-tui-6/internal/ws"
)

const systemPayload = `{"server_ts":1786998815513,"exchange_ping":{"current":264,"ema":263},"balance":{"total":"25.259071788618","margin":"0","leverage":"3"},"symbols":["BTC_USDT","ETH_USDT","SOL_USDT"],"positions":[]}`

func newTestApp() App {
	client := ws.New("ws://unused", "unused")
	return NewApp(client, DefaultLayoutSettings())
}

func TestApp_SystemMsgCreatesTabsForEachSymbol(t *testing.T) {
	a := newTestApp()

	msg := appWsMsg(ws.Message{Channel: "system", Symbol: "", Data: json.RawMessage(systemPayload)})
	updated, _ := a.Update(msg)
	a = updated.(App)

	// tabs = ["Dashboard", "BTC_USDT", "ETH_USDT", "SOL_USDT"]
	if len(a.tabs) != 4 {
		t.Fatalf("ожидалось 4 вкладки (Dashboard+3 символа), получено %d: %v", len(a.tabs), a.tabs)
	}
	for _, symbol := range []string{"BTC_USDT", "ETH_USDT", "SOL_USDT"} {
		if _, ok := a.symbolModels[symbol]; !ok {
			t.Errorf("не создана вкладка для %s", symbol)
		}
	}
}

func TestApp_SystemMsgDoesNotResetExistingTabState(t *testing.T) {
	// Решение из чата: вкладки формируются ОДИН РАЗ по факту прихода
	// symbols, повторный system-msg не должен пересоздавать уже
	// существующие вкладки (иначе накопленный snapshot/orderbook/
	// позиция скролла терялись бы каждые ~10s).
	a := newTestApp()

	msg := appWsMsg(ws.Message{Channel: "system", Symbol: "", Data: json.RawMessage(systemPayload)})
	updated, _ := a.Update(msg)
	a = updated.(App)

	indicatorsMsg := appWsMsg(ws.Message{Channel: "indicators", Symbol: "BTC_USDT", Data: json.RawMessage(realIndicatorsPayload)})
	updated, _ = a.Update(indicatorsMsg)
	a = updated.(App)

	if a.symbolModels["BTC_USDT"].snapshot == nil {
		t.Fatal("snapshot должен быть установлен после indicators")
	}

	// Повторный system с тем же списком символов.
	updated, _ = a.Update(msg)
	a = updated.(App)

	if a.symbolModels["BTC_USDT"].snapshot == nil {
		t.Error("повторный system-msg не должен сбрасывать уже накопленный snapshot вкладки")
	}
}

func TestApp_IndicatorsMsgRoutedOnlyToMatchingSymbol(t *testing.T) {
	a := newTestApp()
	msg := appWsMsg(ws.Message{Channel: "system", Symbol: "", Data: json.RawMessage(systemPayload)})
	updated, _ := a.Update(msg)
	a = updated.(App)

	indicatorsMsg := appWsMsg(ws.Message{Channel: "indicators", Symbol: "BTC_USDT", Data: json.RawMessage(realIndicatorsPayload)})
	updated, _ = a.Update(indicatorsMsg)
	a = updated.(App)

	if a.symbolModels["BTC_USDT"].snapshot == nil {
		t.Error("BTC_USDT должен получить snapshot")
	}
	if a.symbolModels["ETH_USDT"].snapshot != nil {
		t.Error("ETH_USDT не должен получить snapshot, предназначенный для BTC_USDT")
	}
	if a.symbolModels["SOL_USDT"].snapshot != nil {
		t.Error("SOL_USDT не должен получить snapshot, предназначенный для BTC_USDT")
	}
}

func TestApp_TabNavigation_CyclesForwardAndBackward(t *testing.T) {
	a := newTestApp()
	msg := appWsMsg(ws.Message{Channel: "system", Symbol: "", Data: json.RawMessage(systemPayload)})
	updated, _ := a.Update(msg)
	a = updated.(App)

	if a.activeIndex != 0 {
		t.Fatalf("изначально должна быть активна вкладка 0 (Dashboard), получено %d", a.activeIndex)
	}

	updated, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = updated.(App)
	if a.activeIndex != 1 {
		t.Errorf("после Tab ожидался индекс 1, получено %d", a.activeIndex)
	}

	updated, _ = a.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	a = updated.(App)
	if a.activeIndex != 0 {
		t.Errorf("после Shift+Tab ожидался возврат к индексу 0, получено %d", a.activeIndex)
	}

	// Shift+Tab из 0 должен обернуться на последнюю вкладку (индекс 3).
	updated, _ = a.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	a = updated.(App)
	if a.activeIndex != len(a.tabs)-1 {
		t.Errorf("Shift+Tab из индекса 0 должен обернуться на последний индекс %d, получено %d", len(a.tabs)-1, a.activeIndex)
	}
}

func TestCtrlDigitIndex_ParsesValidAndRejectsInvalid(t *testing.T) {
	cases := []struct {
		key     string
		wantIdx int
		wantOK  bool
	}{
		{"ctrl+1", 0, true},
		{"ctrl+2", 1, true},
		{"ctrl+9", 8, true},
		{"ctrl+0", 0, false}, // 0 не входит в 1..9
		{"tab", 0, false},
		{"ctrl+a", 0, false},
		{"ctrl+10", 0, false}, // длина не совпадает с "ctrl+N"
	}
	for _, c := range cases {
		idx, ok := ctrlDigitIndex(c.key)
		if ok != c.wantOK {
			t.Errorf("ctrlDigitIndex(%q) ok = %v, ожидалось %v", c.key, ok, c.wantOK)
			continue
		}
		if ok && idx != c.wantIdx {
			t.Errorf("ctrlDigitIndex(%q) = %d, ожидалось %d", c.key, idx, c.wantIdx)
		}
	}
}

func TestApp_View_ContainsHeaderTabsAndFooter(t *testing.T) {
	a := newTestApp()
	updated, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = updated.(App)

	msg := appWsMsg(ws.Message{Channel: "system", Symbol: "", Data: json.RawMessage(systemPayload)})
	updated, _ = a.Update(msg)
	a = updated.(App)

	out := a.View()
	for _, want := range []string{"DTrader 6", "Dashboard", "BTC_USDT", "LOGS", "POSITIONS", "Tab/Shift+Tab"} {
		if !strings.Contains(out, want) {
			t.Errorf("View() должен содержать %q, получено:\n%s", want, out)
		}
	}
}

func TestApp_View_DoesNotPanicBeforeAnyData(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("View() запаниковал до получения каких-либо данных: %v", r)
		}
	}()
	a := newTestApp()
	updated, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	a = updated.(App)
	_ = a.View()
}

func TestApp_View_ContentAndRightbarHaveSameHeight(t *testing.T) {
	// Решение из чата: рамки content/rightbar "разъезжались" визуально
	// на первом превью главного лайаута — вкладке символа передавался
	// полный размер терминала вместо реально выделенной ей области
	// (после вычета header/tabs/footer/rightbar), из-за чего её
	// viewport считал себя больше, чем нужно.
	a := newTestApp()
	updated, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 45})
	a = updated.(App)

	msg := appWsMsg(ws.Message{Channel: "system", Symbol: "", Data: json.RawMessage(systemPayload)})
	updated, _ = a.Update(msg)
	a = updated.(App)

	// Переключаемся на вкладку символа — там используется viewport,
	// именно там баг проявлялся (Dashboard рендерится по-другому).
	updated, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = updated.(App)

	indMsg := appWsMsg(ws.Message{Channel: "indicators", Symbol: "BTC_USDT", Data: json.RawMessage(realIndicatorsPayload)})
	updated, _ = a.Update(indMsg)
	a = updated.(App)

	contentWidth, bodyHeight := a.contentSize()
	content := a.renderContent(contentWidth, bodyHeight)

	updated, _ = a.Update(tea.WindowSizeMsg{Width: 160, Height: 45}) // пересчитать rightbarVP под текущее состояние
	a = updated.(App)

	// Собираем rightbar тем же способом, что и реальный View() —
	// через a.settings, а не константу (rightbarWidth больше не
	// существует как отдельная константа, см. settings.go).
	rbWidth := a.settings.rightbarWidth(a.width)
	rightbarInnerHeight := bodyHeight - rightbarBorderStyle.GetVerticalFrameSize()
	positionsHeight := a.settings.positionsHeight(rightbarInnerHeight)

	rightbarInner := a.rightbarVP.View()
	positionsBlock := renderPositionsBlock(nil, rbWidth-rightbarBorderStyle.GetHorizontalFrameSize(), positionsHeight)
	rightbar := rightbarBorderStyle.Render(rightbarInner + "\n" + positionsBlock)

	contentLines := strings.Count(content, "\n") + 1
	rightbarLines := strings.Count(rightbar, "\n") + 1

	if contentLines != rightbarLines {
		t.Errorf("content и rightbar должны иметь одинаковую высоту в строках: content=%d, rightbar=%d", contentLines, rightbarLines)
	}
}

func TestApp_View_HeaderFooterWidthMatchesBody(t *testing.T) {
	// Решение из чата: "правый бордер сместился влево" — header/footer
	// были УЖЕ, чем body (content+rightbar), из-за неверной формулы
	// вычитания рамки/паддинга при .Width(). Проверяем на нескольких
	// реалистичных ширинах терминала, что header/footer ровно
	// совпадают по ширине с a.width (а значит и с body, который тоже
	// строится от a.width).
	for _, width := range []int{80, 100, 120, 160, 200} {
		a := newTestApp()
		updated, _ := a.Update(tea.WindowSizeMsg{Width: width, Height: 45})
		a = updated.(App)

		msg := appWsMsg(ws.Message{Channel: "system", Symbol: "", Data: json.RawMessage(systemPayload)})
		updated, _ = a.Update(msg)
		a = updated.(App)

		header := renderHeader(a.system, a.width)
		footer := renderFooter(a.width)

		if got := lipgloss.Width(header); got != width {
			t.Errorf("width=%d: header width = %d, ожидалось %d", width, got, width)
		}
		if got := lipgloss.Width(footer); got != width {
			t.Errorf("width=%d: footer width = %d, ожидалось %d", width, got, width)
		}
	}
}

func TestApp_RightbarHeightStaysBoundedWithManyLogs(t *testing.T) {
	// Решение из чата: за ночь работы накопилось сотни строк
	// реконнектов, rightbar рос без ограничения и раздувал весь кадр
	// вниз, из-за чего header/tabs/content уезжали за пределы видимой
	// области терминала (реальный баг с прод-скриншота). Проверяем,
	// что после добавления сотен логов итоговая высота View() не
	// растёт пропорционально числу логов — viewport должен держать
	// LOGS в границах выделенной высоты.
	a := newTestApp()
	updated, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 45})
	a = updated.(App)

	sysMsg := appWsMsg(ws.Message{Channel: "system", Symbol: "", Data: json.RawMessage(systemPayload)})
	updated, _ = a.Update(sysMsg)
	a = updated.(App)

	heightBefore := lipgloss.Height(a.View())

	// Симулируем сотни реконнектов подряд.
	for i := 0; i < 300; i++ {
		a.addLog(LogWarn, "соединение потеряно, переподключение...")
	}

	heightAfter := lipgloss.Height(a.View())

	if heightAfter != heightBefore {
		t.Errorf("высота View() изменилась после добавления 300 логов: было %d, стало %d — rightbar не ограничен по высоте", heightBefore, heightAfter)
	}
}

func TestApp_View_DoesNotPanicOnVerySmallTerminal(t *testing.T) {
	// Решение из чата: реальный краш на проде — "strings: negative
	// Repeat count" в renderPositionsBlock. Причина: rightbarWidth()
	// защищена минимумом 1, но App.View() затем вычитал размер рамки
	// (rightbarBorderStyle.GetHorizontalFrameSize()) БЕЗ повторной
	// защиты — на маленьком терминале (rbWidth=1, рамка=4) результат
	// уходил в отрицательные числа и падал в strings.Repeat.
	// Проверяем на нескольких маленьких, но реалистичных размерах
	// терминала (в том числе экстремально узких/низких), что View()
	// не паникует.
	sizes := []struct{ w, h int }{
		{10, 10}, {20, 15}, {1, 1}, {5, 5}, {40, 20}, {80, 24},
	}
	for _, size := range sizes {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("View() запаниковал при width=%d height=%d: %v", size.w, size.h, r)
				}
			}()
			a := newTestApp()
			updated, _ := a.Update(tea.WindowSizeMsg{Width: size.w, Height: size.h})
			a = updated.(App)

			msg := appWsMsg(ws.Message{Channel: "system", Symbol: "", Data: json.RawMessage(systemPayload)})
			updated, _ = a.Update(msg)
			a = updated.(App)

			_ = a.View()
		}()
	}
}
