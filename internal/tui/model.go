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
		labelStyle.Render(fmt.Sprintf("Files: %d/%d", m.processed, m.total)) + dimStyle.Render(fmt.Sprintf("  errors:%d", m.errors)),
		labelStyle.Render(fmt.Sprintf("Leaks plugged: %d", m.leaks)),
		labelStyle.Render(fmt.Sprintf("Bytes saved: %d", m.bytesSaved)),
		dimStyle.Render(fmt.Sprintf("Elapsed: %s", elapsed)),
		barStyle.Render(bar),
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
	return "[" + strings.Repeat("=", filled) + strings.Repeat(" ", width-filled) + "]"
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	barStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)
