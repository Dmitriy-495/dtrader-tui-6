package ui

import (
	"fmt"
	"time"
)

// addLog — добавляет запись в лог sidebar
func (m *Model) addLog(entry string) {
	ts := GrayStyle.Render(time.Now().Format("15:04:05"))
	line := fmt.Sprintf("%s %s", ts, GrayStyle.Render(entry))
	m.logs = append(m.logs, line)
	if len(m.logs) > 200 {
		m.logs = m.logs[len(m.logs)-200:]
	}
	m.logView.SetContent(joinLines(m.logs))
	m.logView.GotoBottom()
}

// renderSidebar — правая панель с логом событий
func (m Model) renderSidebar() string {
	return SidebarStyle.
		Width(sidebarWidth).
		Height(m.height - 6).
		Render(m.logView.View())
}
