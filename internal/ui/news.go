package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Dmitriy-495/dtrader-tui-6/internal/news"
	"github.com/charmbracelet/lipgloss"
)

var newsBorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorOrange)

// renderNews — лента биржевых новостей от CryptoPanic
func (m Model) renderNews(w, h int) string {
	title := sectionTitleStyle.Render("📰 News")

	var sb strings.Builder
	if len(m.newsItems) == 0 {
		sb.WriteString(GrayStyle.Render("  загрузка новостей..."))
	} else {
		for _, item := range m.newsItems {
			age := time.Since(item.PublishedAt).Round(time.Minute)
			sentiment := renderSentiment(item)
			line := fmt.Sprintf("%s %s  %s",
				GrayStyle.Render(formatAge(age)),
				sentiment,
				lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(truncate(item.Title, w-20)),
			)
			sb.WriteString(line + "\n")
		}
	}

	return newsBorderStyle.
		Width(w).
		Height(h).
		Render(title + "\n" + sb.String())
}

// renderSentiment — цветной индикатор тональности новости
func renderSentiment(item news.NewsItem) string {
	pos := item.Votes.Positive
	neg := item.Votes.Negative
	if pos > neg*2 {
		return GreenStyle.Render("▲")
	} else if neg > pos*2 {
		return RedStyle.Render("▼")
	}
	return GrayStyle.Render("●")
}

// formatAge — форматирует возраст новости
func formatAge(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%2dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%2dh", int(d.Hours()))
}

// truncate — обрезает строку до maxLen символов
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
