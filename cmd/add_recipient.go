package cmd

import (
	"fmt"

	"filippo.io/age"
	"github.com/frahet/envault/internal/identity"
	"github.com/frahet/envault/internal/vault"
	"github.com/spf13/cobra"
)

var addRecipientCmd = &cobra.Command{
	Use:   "add-recipient <age-pubkey>",
	Short: "Re-encrypt vault to include a new recipient",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddRecipient,
}

func runAddRecipient(cmd *cobra.Command, args []string) error {
	newPubkey := args[0]
	newRecipient, err := age.ParseX25519Recipient(newPubkey)
	if err != nil {
		return fmt.Errorf("invalid age public key: %w", err)
	}

	id, err := identity.Load()
	if err != nil {
		return err
	}

	kv, err := vault.ReadKV(id)
	if err != nil {
		return err
	}

	recipients, pubkeys, err := vault.LoadRecipients()
	if err != nil {
		return err
	}

	// Check for duplicate.
	for _, pk := range pubkeys {
		if pk == newPubkey {
			return fmt.Errorf("recipient already in vault: %s", newPubkey)
		}
	}

	recipients = append(recipients, newRecipient)
	pubkeys = append(pubkeys, newPubkey)

	if err := vault.WriteKV(kv, recipients); err != nil {
		return err
	}
	if err := vault.SaveRecipients(pubkeys); err != nil {
		return err
	}

	fmt.Printf("Added recipient: %s\n", newPubkey)
	fmt.Printf("Vault now encrypted to %d recipient(s).\n", len(pubkeys))
	return nil
}
