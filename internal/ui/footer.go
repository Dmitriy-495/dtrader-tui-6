package ui

import "fmt"

func (m Model) renderFooter() string {
	return FooterStyle.Width(m.width).Render(
		fmt.Sprintf("❯ %s", m.input.View()),
	)
}
