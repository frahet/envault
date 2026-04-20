package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"filippo.io/age"
	"github.com/frahet/envault/internal/vault"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate an age identity and create an empty vault",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	idPath, err := identityPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(idPath); err == nil {
		return fmt.Errorf("identity already exists at %s — delete it first if you want to reinitialise", idPath)
	}

	if _, err := os.Stat(vault.VaultFile); err == nil {
		return fmt.Errorf("%s already exists — delete it first if you want to reinitialise", vault.VaultFile)
	}

	id, err := age.GenerateX25519Identity()
	if err != nil {
		return fmt.Errorf("generate identity: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(idPath), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(idPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintln(f, id)

	pubkey := id.Recipient().String()

	// Bootstrap recipients file and empty vault encrypted to self.
	if err := vault.SaveRecipients([]string{pubkey}); err != nil {
		return err
	}
	if err := vault.WriteKV(map[string]string{}, []age.Recipient{id.Recipient()}); err != nil {
		return err
	}

	fmt.Printf("Identity:  %s\n", idPath)
	fmt.Printf("Public key: %s\n", pubkey)
	fmt.Printf("Vault:     %s\n", vault.VaultFile)
	fmt.Println()
	fmt.Println("Add to .gitignore:  .env.vault is your encrypted secrets. Commit it.")
	fmt.Println("Commit to git:      .env.vault.recipients lists who can decrypt.")
	return nil
}

func identityPath() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "envault", "identity.age"), nil
}
