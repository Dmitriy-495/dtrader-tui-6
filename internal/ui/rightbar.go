package ui

import "github.com/charmbracelet/lipgloss"

var (
	rightbarBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238"))

	sectionTitleStyle = lipgloss.NewStyle().
				Foreground(colorOrange).
				Bold(true)
)
