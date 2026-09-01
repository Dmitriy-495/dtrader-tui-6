// Файл footer.go — нижняя панель главного лайаута.
//
// Раздел 9 CHECKPOINT.md описывает footer.go как "командную строку" —
// полноценный интерпретатор команд (ввод текста, парсинг, история) там
// не описан детально и не обсуждался в чате отдельно, так что для
// этого шага footer показывает подсказки активных горячих клавиш
// (то, что уже реально реализовано и работает: Tab/Shift+Tab,
// Ctrl+1..9, q/Ctrl+C) — минимальная полезная версия footer,
// расширяемая позже до реального ввода команд, когда/если появится
// конкретная спецификация, что эти команды должны делать.
//
// Стиль — тот же, что у header (решение из чата: "footer в стиле
// header, в три строки с рамкой orange" — то есть RoundedBorder
// оранжевого цвета вокруг одной строки контента, что визуально даёт
// верх рамки + контент + низ рамки = 3 печатные строки).
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// footerHints — статичная строка подсказок. Не зависит от текущего
// состояния приложения (в отличие от header/tabs) — горячие клавиши
// одинаковы независимо от того, какая вкладка активна.
const footerHints = "Tab/Shift+Tab: вкладки  •  Ctrl+1-9: перейти к вкладке  •  ↑↓/PgUp/PgDn: прокрутка  •  q/Ctrl+C: выход"

// renderFooter рисует нижнюю панель на всю ширину терминала.
//
// Паттерн — единый с header.go/symbol.go/rightbar.go: .Width()/
// .Padding() применяются на отдельном внутреннем стиле контента,
// внешняя рамка (Border) — без .Width() на себе, просто оборачивает
// уже готовый контент. Экспериментально проверено (см. header.go):
// content := Padding(0,2).Width(N) даёt видимую ширину N (паддинг уже
// включён внутрь N), внешний Border добавляет ровно 2 символа —
// значит N = width - 2 даёт итоговую ширину == width, совпадающую с
// body (content+rightbar) в App.View() (решение из чата: "правый
// бордер сместился влево").
func renderFooter(width int) string {
	textWidth := width - 2
	content := lipgloss.NewStyle().Padding(0, 2).Width(textWidth).Render(mutedStyle.Render(footerHints))
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Render(content)
}
