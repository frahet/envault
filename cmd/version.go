package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set at build time via GoReleaser ldflags.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the envault version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("envault", Version)
	},
}
