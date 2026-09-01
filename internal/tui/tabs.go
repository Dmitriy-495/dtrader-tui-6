// Файл tabs.go — строка вкладок главного лайаута: "Dashboard" (вкладка
// №0, общий обзор всех символов) + по одной вкладке на каждый
// торгуемый символ (формируются при старте по списку symbols из
// первого system-сообщения — решение из чата: "формируется при
// запуске по полученным от сервера торгуемым парам", не хардкод и не
// динамическое добавление/удаление на лету).
//
// Стиль — не буквальный powerline со скошенными Nerd Font глифами:
// такие иконки требуют специального шрифта на стороне пользователя и
// выглядят как битые квадраты без него. Вместо этого — активная
// вкладка выделена ярким фоном, неактивные приглушены текстом, что
// даёт тот же эффект "явно видно активную вкладку", не рискуя
// сломаться на терминале без нужного шрифта.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// dashboardTabIndex — индекс вкладки Dashboard в общем списке вкладок,
// всегда первая (индекс 0) — решение из чата: "вкладка №0".
const dashboardTabIndex = 0

const dashboardTabLabel = "Dashboard"

var (
	activeTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")). // чёрный текст на ярком фоне — контраст важнее общей цветовой схемы здесь
			Background(colorBorder).
			Bold(true).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Padding(0, 2)

	// tabsBarStyle — без border-bottom (решение из чата: "убрать
	// border bottom у наименований вкладок" — линия под вкладками
	// визуально дублировала уже достаточно чёткое разделение через
	// цвет активной вкладки, выглядела избыточно).
	tabsBarStyle = lipgloss.NewStyle()
)

// tabLabels возвращает подписи вкладок в порядке отображения:
// Dashboard первой, затем символы в том порядке, в котором они
// пришли в system.symbols (сохраняем порядок сервера, не сортируем
// заново — если ws-server/bot решат прислать их в осмысленном
// порядке, TUI не должен его ломать).
func tabLabels(symbols []string) []string {
	labels := make([]string, 0, len(symbols)+1)
	labels = append(labels, dashboardTabLabel)
	labels = append(labels, symbols...)
	return labels
}

// renderTabs рисует строку вкладок. activeIndex — индекс текущей
// активной вкладки (0 = Dashboard, 1..N = символы в порядке symbols).
// width — полная ширина терминала.
// renderTabs рисует строку вкладок. activeIndex — индекс текущей
// активной вкладки (0 = Dashboard, 1..N = символы в порядке symbols).
// width — полная ширина терминала.
func renderTabs(symbols []string, activeIndex int, width int) string {
	labels := tabLabels(symbols)

	var b strings.Builder
	for i, label := range labels {
		text := fmt.Sprintf(" %s ", label)
		if i == activeIndex {
			b.WriteString(activeTabStyle.Render(text))
		} else {
			b.WriteString(inactiveTabStyle.Render(text))
		}
	}

	// width-1: решение из чата — "оторви на один символ панель
	// закладок от правой границы для лучшего восприятия". Полоса
	// вкладок не растягивается вплотную до правого края терминала,
	// остаётся один пустой символ отступа справа.
	return tabsBarStyle.Width(width - 1).Render(b.String())
}
