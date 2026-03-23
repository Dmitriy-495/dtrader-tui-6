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

func (m Model) renderMain() string {
	mainH  := m.height - 6
	totalW := m.width - 2           // рабочая ширина без border header
	rightW := totalW * rightbarPct / 100
	leftW  := totalW - rightW

	left  := m.renderContent(leftW, mainH)
	right := m.renderRightbar(rightW, mainH)

	// tabs на полную рабочую ширину
	tabs := m.renderTabs(totalW)

	main := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	return lipgloss.JoinVertical(lipgloss.Left, tabs, main)
}

func (m Model) renderContent(w, h int) string {
	newsH := newsContent + 2
	infoH := h - newsH - 2 // -2 для выравнивания

	var screen string
	vpH := infoH - 2
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

	infoBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorOrange).
		BorderTop(false).
		Width(w - 2).
		Height(infoH).
		PaddingLeft(1).
		Render(vp.View())

	newsBox := m.renderNews(w, newsH)

	return lipgloss.JoinVertical(lipgloss.Left, infoBox, newsBox)
}

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
