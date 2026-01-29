package cmd

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"bleach/internal/processor"
	"bleach/internal/tui"
)

var scanCmd = &cobra.Command{
	Use:   "scan <path>",
	Short: "Report privacy metadata without modifying files",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		updates := make(chan processor.ProgressUpdate, 64)
		model := tui.NewModel(updates)
		program := tea.NewProgram(model)

		uiDone := make(chan struct{})
		go func() {
			_, _ = program.Run()
			close(uiDone)
		}()

		summary, reports, err := processor.Run(context.Background(), path, processor.Options{Mode: processor.ModeScan}, updates)
		close(updates)
		<-uiDone
		if err != nil {
			return err
		}

		for i, report := range reports {
			if i > 0 {
				fmt.Fprintln(os.Stdout)
			}
			fmt.Fprintf(os.Stdout, "%s\n", scanFileStyle.Render(report.Path))
			if len(report.Details) == 0 {
				fmt.Fprintf(os.Stdout, "  %s %s\n",
					scanBulletStyle.Render("-"),
					scanDimStyle.Render("none"),
				)
				continue
			}
			for _, detail := range report.Details {
				if len(detail.Values) == 0 {
					continue
				}
				fmt.Fprintf(os.Stdout, "  %s\n", scanCategoryStyle.Render(detail.Category+":"))
				for _, value := range detail.Values {
					fmt.Fprintf(os.Stdout, "    %s %s\n", scanBulletStyle.Render("-"), scanValueStyle.Render(value))
				}
			}
		}

		_ = summary
		return nil
	},
}

var (
	scanFileStyle     = lipgloss.NewStyle().Bold(true).Foreground(tui.ColorAccent)
	scanCategoryStyle = lipgloss.NewStyle().Foreground(tui.ColorAccentAlt)
	scanValueStyle    = lipgloss.NewStyle().Foreground(tui.ColorInk)
	scanDimStyle      = lipgloss.NewStyle().Foreground(tui.ColorDim)
	scanBulletStyle   = lipgloss.NewStyle().Foreground(tui.ColorDim)
)

func init() {
	rootCmd.AddCommand(scanCmd)
}
