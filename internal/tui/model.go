package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"bleach/internal/processor"
)

type Model struct {
	updates    <-chan processor.ProgressUpdate
	started    time.Time
	width      int
	total      int
	processed  int
	errors     int
	leaks      int
	bytesSaved int64
	quitting   bool
}

type doneMsg struct{}

type updateMsg processor.ProgressUpdate

func NewModel(updates <-chan processor.ProgressUpdate) Model {
	return Model{updates: updates, started: time.Now()}
}

func (m Model) Init() tea.Cmd {
	return listenForUpdates(m.updates)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateMsg:
		m.total += msg.TotalDelta
		m.processed += msg.ProcessedDelta
		m.errors += msg.ErrorDelta
		m.leaks += msg.LeakDelta
		m.bytesSaved += msg.BytesSavedDelta
		return m, listenForUpdates(m.updates)
	case doneMsg:
		m.quitting = true
		return m, tea.Quit
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	barWidth := 40
	if m.width > 0 {
		barWidth = int(math.Min(60, float64(m.width-10)))
		if barWidth < 20 {
			barWidth = 20
		}
	}

	ratio := 0.0
	if m.total > 0 {
		ratio = float64(m.processed) / float64(m.total)
		if ratio > 1 {
			ratio = 1
		}
	}

	bar := renderBar(barWidth, ratio)
	elapsed := time.Since(m.started).Round(time.Millisecond)

	lines := []string{
		titleStyle.Render("bleach 🧼"),
		fmt.Sprintf("%s %s %s", keyStyle.Render("Files"), valueStyle.Render(fmt.Sprintf("%d/%d", m.processed, m.total)), dimStyle.Render(fmt.Sprintf("errors:%d", m.errors))),
		fmt.Sprintf("%s %s", keyStyle.Render("Leaks plugged"), valueStyle.Render(fmt.Sprintf("%d", m.leaks))),
		fmt.Sprintf("%s %s", keyStyle.Render("Bytes saved"), valueStyle.Render(fmt.Sprintf("%d", m.bytesSaved))),
		dimStyle.Render(fmt.Sprintf("Elapsed: %s", elapsed)),
		renderBarLine(bar),
	}

	return strings.Join(lines, "\n")
}

func listenForUpdates(updates <-chan processor.ProgressUpdate) tea.Cmd {
	return func() tea.Msg {
		update, ok := <-updates
		if !ok {
			return doneMsg{}
		}
		return updateMsg(update)
	}
}

func renderBar(width int, ratio float64) string {
	filled := int(math.Round(ratio * float64(width)))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return barFillStyle.Render(strings.Repeat("=", filled)) + barEmptyStyle.Render(strings.Repeat(".", width-filled))
}

var (
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	keyStyle        = lipgloss.NewStyle().Foreground(ColorAccentAlt)
	valueStyle      = lipgloss.NewStyle().Foreground(ColorInk).Bold(true)
	dimStyle        = lipgloss.NewStyle().Foreground(ColorDim)
	barFillStyle    = lipgloss.NewStyle().Foreground(ColorAccent)
	barEmptyStyle   = lipgloss.NewStyle().Foreground(ColorDim)
	barBracketStyle = lipgloss.NewStyle().Foreground(ColorDim)
)

func renderBarLine(bar string) string {
	return barBracketStyle.Render("[") + bar + barBracketStyle.Render("]")
}
