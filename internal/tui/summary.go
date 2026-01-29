package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type SummaryRow struct {
	Label string
	Value string
}

func RenderSummary(rows []SummaryRow) string {
	labelWidth := 0
	valueWidth := 0
	for _, row := range rows {
		if len(row.Label) > labelWidth {
			labelWidth = len(row.Label)
		}
		if len(row.Value) > valueWidth {
			valueWidth = len(row.Value)
		}
	}

	hline := strings.Repeat("-", labelWidth+valueWidth+3)
	lines := []string{dimStyle.Render(hline)}

	for _, row := range rows {
		label := padRight(row.Label, labelWidth)
		value := padRight(row.Value, valueWidth)
		line := fmt.Sprintf("%s %s %s", labelStyle.Render(label), dimStyle.Render("|"), summaryValueStyle.Render(value))
		lines = append(lines, line)
	}

	lines = append(lines, dimStyle.Render(hline))
	return strings.Join(lines, "\n")
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

var (
	labelStyle        = lipgloss.NewStyle().Foreground(ColorAccentAlt)
	summaryValueStyle = lipgloss.NewStyle().Foreground(ColorInk).Bold(true)
)
