package cmd

import (
	"fmt"
	"os"

	"github.com/frahet/envault/internal/vault"
)

// scopeForWrite picks the default scope for a write command.
// --global always wins. Otherwise: local if a local vault exists in cwd, else global.
// This means writes in a project with an opted-in local vault go there; elsewhere they go global.
func scopeForWrite(useGlobal bool) vault.Scope {
	if useGlobal {
		return vault.ScopeGlobal
	}
	if _, err := os.Stat(vault.VaultFile); err == nil {
		return vault.ScopeLocal
	}
	return vault.ScopeGlobal
}

// missingVaultErr produces a user-facing error when a write targets a scope with no vault yet.
func missingVaultErr(s vault.Scope) error {
	if s == vault.ScopeGlobal {
		return fmt.Errorf("no global vault yet — run `envault init --global` first")
	}
	return fmt.Errorf("no local vault here — run `envault init` in this directory, or pass --global to use your personal vault")
}
