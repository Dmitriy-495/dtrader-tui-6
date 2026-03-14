package ui

import "github.com/charmbracelet/lipgloss"

// =============================================================================
// Цветовая палитра
// =============================================================================
var (
	colorOrange = lipgloss.Color("214")
	colorGreen  = lipgloss.Color("82")
	colorYellow = lipgloss.Color("226")
	colorRed    = lipgloss.Color("196")
	colorGray   = lipgloss.Color("239")
	colorBg     = lipgloss.Color("236")
)

// =============================================================================
// Базовые стили
// =============================================================================
var (
	OrangeStyle = lipgloss.NewStyle().Foreground(colorOrange).Bold(true)
	GreenStyle  = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	YellowStyle = lipgloss.NewStyle().Foreground(colorYellow).Bold(true)
	RedStyle    = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	GrayStyle   = lipgloss.NewStyle().Foreground(colorGray)
)

// =============================================================================
// Стили компонентов
// =============================================================================
var (
	// HeaderStyle — без фона, просто Bold
	HeaderStyle = lipgloss.NewStyle().Bold(true)

	FooterStyle = lipgloss.NewStyle().
			Background(colorBg).
			Foreground(colorOrange).
			PaddingLeft(1)

	ContentStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			PaddingLeft(1).
			PaddingRight(1)

	SidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			PaddingLeft(1).
			PaddingRight(1)
)
