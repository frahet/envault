package cmd

import (
	"fmt"

	"github.com/frahet/envault/internal/identity"
	"github.com/spf13/cobra"
)

var pubkeyCmd = &cobra.Command{
	Use:   "pubkey",
	Short: "Print the public key of the current identity (scriptable)",
	RunE:  runPubkey,
}

func runPubkey(cmd *cobra.Command, args []string) error {
	id, err := identity.Load()
	if err != nil {
		return err
	}
	fmt.Println(id.Recipient().String())
	return nil
}
