package cmd

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
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

		for _, report := range reports {
			if len(report.Categories) == 0 {
				fmt.Fprintf(os.Stdout, "%s: none\n", report.Path)
				continue
			}
			fmt.Fprintf(os.Stdout, "%s: %s\n", report.Path, joinCategories(report.Categories))
		}

		_ = summary
		return nil
	},
}

func joinCategories(categories []string) string {
	out := ""
	for i, c := range categories {
		if i > 0 {
			out += ", "
		}
		out += c
	}
	return out
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
