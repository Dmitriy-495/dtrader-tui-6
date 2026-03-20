package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

const (
	newsContent  = 10
	rightbarPct  = 18
	logsPct      = 60
	paddingRight = 4 // отступ справа чтобы бордер был виден
)

func (m Model) renderMain() string {
	mainH  := m.height - 5
	w      := m.width - paddingRight // рабочая ширина с отступом справа
	rightW := w * rightbarPct / 100
	leftW  := w - rightW - 1

	left  := m.renderLeft(leftW, mainH)
	right := m.renderRight(rightW, mainH)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Model) renderLeft(w, h int) string {
	screenH := h - newsContent - 4
	if screenH < 1 {
		screenH = 1
	}
	vpH := screenH - 2
	if vpH < 1 {
		vpH = 1
	}

	var screen string
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

	screenBox := ContentStyle.
		Width(w).
		Height(screenH).
		Render(m.renderTabs(w) + "\n" + vp.View())

	newsBox := m.renderNews(w, newsContent)

	return lipgloss.JoinVertical(lipgloss.Left, screenBox, newsBox)
}

func (m Model) renderRight(w, h int) string {
	total := h - 4 - 4
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

