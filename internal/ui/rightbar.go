package ui

import "github.com/charmbracelet/lipgloss"

var (
	rightbarBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorOrange)

	sectionTitleStyle = lipgloss.NewStyle().
				Foreground(colorOrange).
				Bold(true)
)
