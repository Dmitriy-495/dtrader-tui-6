package ui

import (
	"fmt"
	"strings"
)

func (m Model) renderFooter() string {
	main := m.renderMain()
	mainLines := strings.Count(main, "\n") + 1
	debug := fmt.Sprintf("[H=%d mainH=%d renderMain=%d строк]",
		m.height, m.height-2, mainLines)
	return FooterStyle.Width(m.width).Render(
		fmt.Sprintf("❯ %s  %s", m.input.View(), debug),
	)
}
