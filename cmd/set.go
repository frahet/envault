package cmd

import (
	"fmt"
	"strings"

	"filippo.io/age"
	"github.com/frahet/envault/internal/identity"
	"github.com/frahet/envault/internal/vault"
	"github.com/spf13/cobra"
)

var setCmd = &cobra.Command{
	Use:   "set KEY=VALUE",
	Short: "Encrypt and store a secret (overwrites existing key)",
	Args:  cobra.ExactArgs(1),
	RunE:  runSet,
}

func runSet(cmd *cobra.Command, args []string) error {
	parts := strings.SplitN(args[0], "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("usage: envault set KEY=VALUE")
	}
	key, val := parts[0], parts[1]
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	if err := vault.ValidateValue(val); err != nil {
		return err
	}

	id, err := identity.Load()
	if err != nil {
		return err
	}

	kv, err := vault.ReadKV(id)
	if err != nil {
		return err
	}

	kv[key] = val

	recipients, _, err := vault.LoadRecipients()
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		recipients = []age.Recipient{id.Recipient()}
	}

	return vault.WriteKV(kv, recipients)
}
