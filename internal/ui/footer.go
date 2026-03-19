package ui

import (
	"fmt"
	"strings"
)

func (m Model) renderFooter() string {
	mainH    := m.height - 4
	left     := m.renderLeft(m.width - m.width*rightbarPct/100 - 1, mainH)
	right    := m.renderRight(m.width*rightbarPct/100, mainH)
	leftH    := strings.Count(left, "\n") + 1
	rightH   := strings.Count(right, "\n") + 1
	debug    := fmt.Sprintf("[H=%d mainH=%d left=%d right=%d renderMain=%d]",
		m.height, mainH, leftH, rightH, strings.Count(m.renderMain(), "\n")+1)
	return FooterStyle.Width(m.width).Render(
		fmt.Sprintf("❯ %s  %s", m.input.View(), debug),
	)
}
