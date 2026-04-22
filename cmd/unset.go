package cmd

import (
	"errors"
	"fmt"

	"filippo.io/age"
	"github.com/frahet/envault/internal/identity"
	"github.com/frahet/envault/internal/vault"
	"github.com/spf13/cobra"
)

var unsetGlobalFlag bool

var unsetCmd = &cobra.Command{
	Use:   "unset KEY",
	Short: "Remove a secret from the vault",
	Args:  cobra.ExactArgs(1),
	RunE:  runUnset,
}

func init() {
	unsetCmd.Flags().BoolVar(&unsetGlobalFlag, "global", false, "remove from the global vault (~/.envault/) instead of local")
}

func runUnset(cmd *cobra.Command, args []string) error {
	key := args[0]
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	scope := scopeForWrite(unsetGlobalFlag)

	id, err := identity.Load()
	if err != nil {
		return err
	}

	kv, err := vault.ReadKV(id, scope)
	if err != nil {
		if errors.Is(err, vault.ErrNoVault) {
			return missingVaultErr(scope)
		}
		return err
	}

	if _, exists := kv[key]; !exists {
		fmt.Fprintf(cmd.OutOrStdout(), "key %q not found in %s vault (no-op)\n", key, scope)
		return nil
	}

	delete(kv, key)

	recipients, _, err := vault.LoadRecipients(scope)
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		recipients = []age.Recipient{id.Recipient()}
	}

	if err := vault.WriteKV(kv, recipients, scope); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "removed %q from %s vault\n", key, scope)
	return nil
}
