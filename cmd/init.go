package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"filippo.io/age"
	"github.com/frahet/envault/internal/vault"
	"github.com/spf13/cobra"
)

var initGlobalFlag bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new vault (local in cwd by default, or --global for your personal vault)",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().BoolVar(&initGlobalFlag, "global", false, "create the global vault at ~/.envault/")
}

func runInit(cmd *cobra.Command, args []string) error {
	scope := vault.ScopeLocal
	if initGlobalFlag {
		scope = vault.ScopeGlobal
	}

	idPath, err := identityPath()
	if err != nil {
		return err
	}

	vaultPath, err := vault.VaultPath(scope)
	if err != nil {
		return err
	}

	if _, err := os.Stat(vaultPath); err == nil {
		return fmt.Errorf("%s already exists — %s vault is already initialised", vaultPath, scope)
	}

	id, identityIsNew, err := ensureIdentity(idPath)
	if err != nil {
		return err
	}
	pubkey := id.Recipient().String()

	if err := vault.SaveRecipients([]string{pubkey}, scope); err != nil {
		return err
	}
	if err := vault.WriteKV(map[string]string{}, []age.Recipient{id.Recipient()}, scope); err != nil {
		return err
	}

	if identityIsNew {
		fmt.Printf("Identity:   %s (new)\n", idPath)
	} else {
		fmt.Printf("Identity:   %s (reused)\n", idPath)
	}
	fmt.Printf("Public key: %s\n", pubkey)
	fmt.Printf("Vault:      %s (%s)\n", vaultPath, scope)
	fmt.Println()
	if scope == vault.ScopeLocal {
		fmt.Println("Commit to git:      .env.vault (encrypted secrets), .env.vault.recipients (who can decrypt).")
		fmt.Println("Add to .gitignore:  .env, .env.local, .env.production (never commit plaintext).")
	} else {
		fmt.Println("Personal vault ready. Add keys with `envault set KEY=VALUE` from anywhere.")
		fmt.Println("They are visible everywhere (local vaults override on key collision).")
	}
	return nil
}

// ensureIdentity returns the user's age identity, creating it if the file does not yet exist.
// The identity file is always read from disk (never ENVAULT_IDENTITY) because init writes to disk.
func ensureIdentity(path string) (*age.X25519Identity, bool, error) {
	if _, err := os.Stat(path); err == nil {
		f, err := os.Open(path)
		if err != nil {
			return nil, false, err
		}
		defer f.Close()
		ids, err := age.ParseIdentities(f)
		if err != nil {
			return nil, false, fmt.Errorf("%s: %w", path, err)
		}
		for _, id := range ids {
			if x, ok := id.(*age.X25519Identity); ok {
				return x, false, nil
			}
		}
		return nil, false, fmt.Errorf("%s: no X25519 identity found", path)
	}

	id, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, false, fmt.Errorf("generate identity: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, false, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	if _, err := fmt.Fprintln(f, id); err != nil {
		return nil, false, err
	}
	return id, true, nil
}

func identityPath() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "envault", "identity.age"), nil
}
