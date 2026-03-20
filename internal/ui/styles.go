package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorOrange = lipgloss.Color("214")
	colorGreen  = lipgloss.Color("82")
	colorYellow = lipgloss.Color("226")
	colorRed    = lipgloss.Color("196")
	colorGray   = lipgloss.Color("239")
	colorBg     = lipgloss.Color("236")
	colorBorder = lipgloss.Color("214")
)

var (
	OrangeStyle = lipgloss.NewStyle().Foreground(colorOrange).Bold(true)
	GreenStyle  = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	YellowStyle = lipgloss.NewStyle().Foreground(colorYellow).Bold(true)
	RedStyle    = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	GrayStyle   = lipgloss.NewStyle().Foreground(colorGray)
)

var (
	HeaderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Bold(true)

	FooterStyle = lipgloss.NewStyle().
			Background(colorBg).
			Foreground(colorOrange).
			PaddingLeft(1)

	// ContentStyle — только левая, правая и нижняя границы
	// верхняя граница заменяется вкладками
	ContentStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			BorderTop(false).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true).
			PaddingLeft(1).
			PaddingRight(1)

	SidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			PaddingLeft(1).
			PaddingRight(1)
)
