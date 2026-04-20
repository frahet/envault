package cmd

import (
	"fmt"
	"os"

	"github.com/frahet/envault/internal/identity"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Print the current identity source and public key",
	RunE:  runWhoami,
}

func runWhoami(cmd *cobra.Command, args []string) error {
	id, err := identity.Load()
	if err != nil {
		return err
	}

	if os.Getenv("ENVAULT_IDENTITY") != "" {
		fmt.Println("Identity:   ENVAULT_IDENTITY env var")
	} else {
		path, err := identity.DefaultPath()
		if err != nil {
			return err
		}
		fmt.Printf("Identity:   %s\n", path)
	}
	fmt.Printf("Public key: %s\n", id.Recipient().String())
	return nil
}
