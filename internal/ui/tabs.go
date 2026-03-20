package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Tab struct {
	Icon  string
	Label string
	Index int
}

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

var (
	inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder   = tabBorderWithBottom("┘", " ", "└")

	inactiveTabStyle = lipgloss.NewStyle().
				Border(inactiveTabBorder, true).
				BorderForeground(colorOrange).
				Foreground(colorGray).
				Padding(0, 1)

	activeTabStyle = lipgloss.NewStyle().
			Border(activeTabBorder, true).
			BorderForeground(colorOrange).
			Foreground(colorOrange).
			Bold(true).
			Padding(0, 1)

	tabLineStyle = lipgloss.NewStyle().Foreground(colorOrange)
)

func (m Model) buildTabs() []Tab {
	tabs := []Tab{
		{Icon: "📊", Label: "Dashboard", Index: 0},
	}
	icons := map[string]string{
		"BTC_USDT": "₿",
		"ETH_USDT": "Ξ",
		"SOL_USDT": "◎",
	}
	for i, sym := range m.dashboard.Symbols {
		icon := icons[sym]
		if icon == "" {
			icon = "◆"
		}
		tabs = append(tabs, Tab{
			Icon:  icon,
			Label: sym,
			Index: i + 1,
		})
	}
	return tabs
}

func (m Model) renderTabs(w int) string {
	tabs := m.buildTabs()
	total := len(tabs)

	var rendered []string
	for i, tab := range tabs {
		isFirst  := i == 0
		isLast   := i == total-1
		isActive := m.activeTab == tab.Index

		var style lipgloss.Style
		if isActive {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}

		border, _, _, _, _ := style.GetBorder()
		if isFirst && isActive {
			border.BottomLeft = "│"
		} else if isFirst && !isActive {
			border.BottomLeft = "├"
		} else if isLast && isActive {
			border.BottomRight = "│"
		} else if isLast && !isActive {
			border.BottomRight = "┤"
		}
		style = style.Border(border)
		rendered = append(rendered, style.Render(tab.Icon+" "+tab.Label))
	}

	row  := lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
	rowW := lipgloss.Width(row)

	// Линия до конца + скруглённый уголок ╮
	lineW := w - rowW - 1
	if lineW < 0 {
		lineW = 0
	}
	line := tabLineStyle.Render(strings.Repeat("─", lineW) + "╮")

	return row + line
}
