package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

const (
	newsContent = 8
	rightbarPct = 20
	logsPct     = 60
)

// renderMain — [content | rightbar]
func (m Model) renderMain() string {
	mainH  := m.height - 5 // header(3) + footer(1) + \n(1)
	rightW := m.width * rightbarPct / 100
	leftW  := m.width - rightW

	left  := m.renderContent(leftW, mainH)
	right := m.renderRightbar(rightW, mainH)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

// renderContent — tabs + info + news
func (m Model) renderContent(w, h int) string {
	newsH  := newsContent + 2 // +2 border
	tabsH  := 2               // tabs строки (без нижней границы активной)
	infoH  := h - newsH - tabsH

	// Активный экран
	var screen string
	vpH := infoH - 2 // -2 border left+right overhead
	if vpH < 1 {
		vpH = 1
	}
	if m.activeTab == 0 {
		screen = m.dashboard.Render(w-4, vpH)
	} else if m.activeTab <= len(m.dashboard.Symbols) {
		sym := m.dashboard.Symbols[m.activeTab-1]
		if pm, ok := m.pairModels[sym]; ok {
			screen = pm.Render(w-4, vpH)
		}
	}

	vp := viewport.New(w-4, vpH)
	vp.SetContent(screen)

	// tabs section
	tabs := m.renderTabs(w)

	// info section — рамка без верхней границы
	infoBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorOrange).
		BorderTop(false).
		Width(w - 2).
		Height(infoH).
		PaddingLeft(1).
		Render(vp.View())

	// news section
	newsBox := m.renderNews(w, newsH)

	return lipgloss.JoinVertical(lipgloss.Left, tabs, infoBox, newsBox)
}

// renderRightbar — logs + positions
func (m Model) renderRightbar(w, h int) string {
	total := h - 4
	logsH := total * logsPct / 100
	posH  := total - logsH

	logs := rightbarBorderStyle.
		Width(w - 1).
		Height(logsH).
		Render(sectionTitleStyle.Render("📋 Logs") + "\n" + m.logView.View())

	positions := rightbarBorderStyle.
		Width(w - 1).
		Height(posH).
		Render(sectionTitleStyle.Render("📈 Positions") + "\n" +
			GrayStyle.Render("  нет открытых позиций"))

	return lipgloss.JoinVertical(lipgloss.Left, logs, positions)
}
