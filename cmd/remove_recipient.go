package cmd

import (
	"fmt"

	"filippo.io/age"
	"github.com/frahet/envault/internal/identity"
	"github.com/frahet/envault/internal/vault"
	"github.com/spf13/cobra"
)

var removeRecipientCmd = &cobra.Command{
	Use:   "remove-recipient <age-pubkey>",
	Short: "Remove a recipient and re-encrypt the vault",
	Long: `Remove a recipient from the vault and re-encrypt to remaining recipients.

Note: this prevents future access but does NOT protect historical git commits.
Historical .env.vault files remain decryptable by the removed recipient.
After removing a recipient, rotate all secrets (re-set them) to fully revoke access.`,
	Args: cobra.ExactArgs(1),
	RunE: runRemoveRecipient,
}

func runRemoveRecipient(cmd *cobra.Command, args []string) error {
	targetPubkey := args[0]

	id, err := identity.Load()
	if err != nil {
		return err
	}

	kv, err := vault.ReadKV(id)
	if err != nil {
		return err
	}

	_, pubkeys, err := vault.LoadRecipients()
	if err != nil {
		return err
	}

	// Build new recipient list without the target.
	var remaining []string
	found := false
	for _, pk := range pubkeys {
		if pk == targetPubkey {
			found = true
			continue
		}
		remaining = append(remaining, pk)
	}

	if !found {
		return fmt.Errorf("recipient not found in vault: %s", targetPubkey)
	}
	if len(remaining) == 0 {
		return fmt.Errorf("cannot remove the only recipient — vault would become unreadable\nTo destroy the vault intentionally: rm %s", vault.VaultFile)
	}

	// Parse remaining pubkeys into age.Recipient values.
	var newRecipients []age.Recipient
	for _, pk := range remaining {
		r, err := age.ParseX25519Recipient(pk)
		if err != nil {
			return fmt.Errorf("invalid pubkey %q: %w", pk, err)
		}
		newRecipients = append(newRecipients, r)
	}

	if err := vault.WriteKV(kv, newRecipients); err != nil {
		return err
	}
	if err := vault.SaveRecipients(remaining); err != nil {
		return err
	}

	fmt.Printf("Removed recipient: %s\n", targetPubkey)
	fmt.Printf("Vault now encrypted to %d recipient(s).\n", len(remaining))
	fmt.Println()
	fmt.Println("To fully revoke access, rotate all secrets — the removed recipient can still")
	fmt.Println("decrypt historical .env.vault files from git history.")
	return nil
}
