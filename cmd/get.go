package cmd

import (
	"fmt"
	"os"

	"github.com/frahet/envault/internal/identity"
	"github.com/frahet/envault/internal/vault"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get KEY",
	Short: "Decrypt and print a single secret value",
	Args:  cobra.ExactArgs(1),
	RunE:  runGet,
}

func runGet(cmd *cobra.Command, args []string) error {
	fmt.Fprintln(os.Stderr, "warning: secret value printed to stdout — add `envault get *` to HISTIGNORE to prevent shell history capture")

	id, err := identity.Load()
	if err != nil {
		return err
	}

	kv, err := vault.ReadKV(id)
	if err != nil {
		return err
	}

	v, ok := kv[args[0]]
	if !ok {
		return fmt.Errorf("key not found: %s", args[0])
	}

	fmt.Println(v)
	return nil
}
