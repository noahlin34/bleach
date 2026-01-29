package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"bleach/internal/processor"
	"bleach/internal/tui"
)

var (
	cleanInPlace     bool
	cleanOutputDir   string
	cleanPreserveICC bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean [flags] <path>",
	Short: "Strip EXIF/XMP/IPTC metadata from images",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		if cleanInPlace && cleanOutputDir != "" {
			return fmt.Errorf("--inplace cannot be used with --output")
		}

		outputDir := cleanOutputDir
		if !cleanInPlace && outputDir == "" {
			outputDir = "bleached"
		}

		if !cleanInPlace {
			if err := os.MkdirAll(outputDir, 0o755); err != nil {
				return err
			}
		}

		updates := make(chan processor.ProgressUpdate, 64)
		model := tui.NewModel(updates)
		program := tea.NewProgram(model)

		uiDone := make(chan struct{})
		go func() {
			_, _ = program.Run()
			close(uiDone)
		}()

		summary, _, err := processor.Run(context.Background(), path, processor.Options{
			Mode:        processor.ModeClean,
			InPlace:     cleanInPlace,
			OutputDir:   outputDir,
			PreserveICC: cleanPreserveICC,
		}, updates)

		close(updates)
		<-uiDone
		if err != nil {
			return err
		}

		rows := []tui.SummaryRow{
			{Label: "Total files processed", Value: fmt.Sprintf("%d", summary.Processed)},
			{Label: "Privacy leaks plugged", Value: fmt.Sprintf("%d", summary.Leaks)},
			{Label: "Space saved (bytes)", Value: fmt.Sprintf("%d", summary.BytesSaved)},
		}
		fmt.Fprintln(os.Stdout, tui.RenderSummary(rows))
		if cleanInPlace {
			fmt.Fprintln(os.Stdout, "In-place clean complete.")
		} else {
			outPath := outputDir
			if abs, absErr := filepath.Abs(outputDir); absErr == nil {
				outPath = abs
			}
			fmt.Fprintf(os.Stdout, "Cleaned files written to: %s\n", outPath)
			fmt.Fprintln(os.Stdout, "Note: originals are unchanged unless --inplace is used.")
		}

		return nil
	},
}

func init() {
	cleanCmd.Flags().BoolVarP(&cleanInPlace, "inplace", "i", false, "modify files in place")
	cleanCmd.Flags().StringVarP(&cleanOutputDir, "output", "o", "", "destination folder for sanitized copies")
	cleanCmd.Flags().BoolVar(&cleanPreserveICC, "preserve-icc", false, "preserve ICC color profiles")

	rootCmd.AddCommand(cleanCmd)
}
