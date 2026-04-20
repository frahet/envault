package cmd

import (
	"errors"
	"fmt"

	"filippo.io/age"
	"github.com/frahet/envault/internal/identity"
	"github.com/frahet/envault/internal/vault"
	"github.com/spf13/cobra"
)

var addRecipientGlobalFlag bool

var addRecipientCmd = &cobra.Command{
	Use:   "add-recipient <age-pubkey>",
	Short: "Re-encrypt vault to include a new recipient",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddRecipient,
}

func init() {
	addRecipientCmd.Flags().BoolVar(&addRecipientGlobalFlag, "global", false, "operate on the global vault (~/.envault/) instead of local")
}

func runAddRecipient(cmd *cobra.Command, args []string) error {
	newPubkey := args[0]
	newRecipient, err := age.ParseX25519Recipient(newPubkey)
	if err != nil {
		return fmt.Errorf("invalid age public key: %w", err)
	}

	scope := scopeForWrite(addRecipientGlobalFlag)

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

	recipients, pubkeys, err := vault.LoadRecipients(scope)
	if err != nil {
		return err
	}

	for _, pk := range pubkeys {
		if pk == newPubkey {
			return fmt.Errorf("recipient already in %s vault: %s", scope, newPubkey)
		}
	}

	recipients = append(recipients, newRecipient)
	pubkeys = append(pubkeys, newPubkey)

	if err := vault.WriteKV(kv, recipients, scope); err != nil {
		return err
	}
	if err := vault.SaveRecipients(pubkeys, scope); err != nil {
		return err
	}

	fmt.Printf("Added recipient to %s vault: %s\n", scope, newPubkey)
	fmt.Printf("Vault now encrypted to %d recipient(s).\n", len(pubkeys))
	return nil
}
