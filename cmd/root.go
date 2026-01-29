package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bleach",
	Short: "bleach 🧼 - strip identifying metadata from images",
	Long:  "bleach 🧼 is a concurrency-safe CLI for stripping EXIF, XMP, and IPTC metadata from images.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
}
