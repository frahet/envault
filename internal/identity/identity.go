package identity

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
)

const envVar = "ENVAULT_IDENTITY"

// Load returns the age identity from ENVAULT_IDENTITY env var or ~/.config/envault/identity.age.
// ENVAULT_IDENTITY is checked first, enabling CI/CD without an identity file on disk.
func Load() (*age.X25519Identity, error) {
	if raw := os.Getenv(envVar); raw != "" {
		// GitHub Actions injects multiline secrets with literal \n — handle both forms.
		raw = strings.ReplaceAll(raw, `\n`, "\n")
		ids, err := age.ParseIdentities(strings.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", envVar, err)
		}
		for _, id := range ids {
			if x, ok := id.(*age.X25519Identity); ok {
				return x, nil
			}
		}
		return nil, fmt.Errorf("%s: no X25519 identity found", envVar)
	}

	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no identity found at %s — run `envault init`", path)
		}
		return nil, err
	}
	defer f.Close()

	ids, err := age.ParseIdentities(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for _, id := range ids {
		if x, ok := id.(*age.X25519Identity); ok {
			return x, nil
		}
	}
	return nil, fmt.Errorf("%s: no X25519 identity found", path)
}

// DefaultPath returns ~/.config/envault/identity.age.
func DefaultPath() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "envault", "identity.age"), nil
}
