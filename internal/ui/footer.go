package ui

import (
	"fmt"
	"strings"
)

func (m Model) renderFooter() string {
	mainH  := m.height - 6
	rightW := m.width * rightbarPct / 100
	leftW  := m.width - rightW
	left   := m.renderContent(leftW, mainH)
	right  := m.renderRightbar(rightW, mainH)
	leftH  := strings.Count(left, "\n") + 1
	rightH := strings.Count(right, "\n") + 1
	debug  := fmt.Sprintf("[H=%d mainH=%d left=%d right=%d]",
		m.height, mainH, leftH, rightH)
	return FooterStyle.Width(m.width).Render(
		fmt.Sprintf("❯ %s  %s", m.input.View(), debug),
	)
}
