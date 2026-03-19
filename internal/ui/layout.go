package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

const (
	newsContent = 10
	rightbarPct = 20
	logsPct     = 60
)

func (m Model) renderMain() string {
	mainH  := m.height - 4
	rightW := m.width * rightbarPct / 100
	leftW  := m.width - rightW - 1

	left  := m.renderLeft(leftW, mainH)
	right := m.renderRight(rightW, mainH)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Model) renderLeft(w, h int) string {
	screenH := h - newsContent - 2
	if screenH < 1 {
		screenH = 1
	}
	vpH := screenH - 2
	if vpH < 1 {
		vpH = 1
	}

	var screen string
	if m.activeTab == 0 {
		screen = m.renderDashboard()
	} else if m.activeTab <= len(m.symbols) {
		screen = m.renderPairScreen(m.symbols[m.activeTab-1])
	}

	vp := viewport.New(w-4, vpH)
	vp.SetContent(screen)

	screenBox := ContentStyle.
		Width(w).
		Height(screenH).
		Render(m.renderTabs(w) + "\n" + vp.View())

	newsBox := m.renderNews(w, newsContent)

	return lipgloss.JoinVertical(lipgloss.Left, screenBox, newsBox)
}

func (m Model) renderRight(w, h int) string {
	total := h - 4 // logsH + posH = total, каждый блок +2 border = h
	logsH := total * logsPct / 100
	posH  := total - logsH

	logs := rightbarBorderStyle.
		Width(w).
		Height(logsH).
		Render(sectionTitleStyle.Render("📋 Logs") + "\n" + m.logView.View())

	positions := rightbarBorderStyle.
		Width(w).
		Height(posH).
		Render(sectionTitleStyle.Render("📈 Positions") + "\n" +
			GrayStyle.Render("  нет открытых позиций"))

	return lipgloss.JoinVertical(lipgloss.Left, logs, positions)
}

func (m Model) renderTabs(w int) string {
	activeStyle   := lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Underline(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(colorGray)

	result := ""
	if m.activeTab == 0 {
		result += activeStyle.Render("[ Dashboard ]")
	} else {
		result += inactiveStyle.Render("  Dashboard  ")
	}
	for i, sym := range m.symbols {
		if m.activeTab == i+1 {
			result += activeStyle.Render("[ " + sym + " ]")
		} else {
			result += inactiveStyle.Render("  " + sym + "  ")
		}
	}
	result += GrayStyle.Render("  Tab/0-5")
	return result
}
