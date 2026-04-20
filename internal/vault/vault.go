package vault

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
)

// VaultFile is the filename used inside whichever directory holds the vault (local or global).
const VaultFile = ".env.vault"

const globalDirName = ".envault"

// Scope selects which vault a command operates on.
type Scope int

const (
	ScopeLocal Scope = iota
	ScopeGlobal
)

func (s Scope) String() string {
	if s == ScopeGlobal {
		return "global"
	}
	return "local"
}

// ErrNoVault is returned when no vault file exists for the requested scope (or for either scope in merged reads).
var ErrNoVault = errors.New("no vault found — run `envault init` (or `envault init --global`)")

// VaultPath returns the vault file path for the given scope.
func VaultPath(s Scope) (string, error) {
	if s == ScopeLocal {
		return VaultFile, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, globalDirName, VaultFile), nil
}

// ReadKV decrypts the vault at the given scope.
// Returns ErrNoVault if the vault file does not exist.
func ReadKV(id age.Identity, s Scope) (map[string]string, error) {
	path, err := VaultPath(s)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoVault
		}
		return nil, err
	}
	defer f.Close()

	ar := armor.NewReader(f)
	r, err := age.Decrypt(ar, id)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (%s vault): %w", s, err)
	}
	return parseKV(r)
}

// ReadMerged reads global and local vaults (whichever exist) and merges them.
// Local entries win on key collision. Returns the merged kv map and a per-key
// source map so callers can annotate output.
// Returns ErrNoVault if neither scope has a vault.
func ReadMerged(id age.Identity) (map[string]string, map[string]Scope, error) {
	kv := map[string]string{}
	source := map[string]Scope{}
	foundAny := false

	for _, s := range []Scope{ScopeGlobal, ScopeLocal} {
		path, err := VaultPath(s)
		if err != nil {
			return nil, nil, err
		}
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, err
		}
		foundAny = true
		part, err := ReadKV(id, s)
		if err != nil {
			return nil, nil, err
		}
		for k, v := range part {
			kv[k] = v
			source[k] = s
		}
	}

	if !foundAny {
		return nil, nil, ErrNoVault
	}
	return kv, source, nil
}

// WriteKV encrypts kv and writes the vault atomically to the given scope.
// For the global scope, the parent directory is created (mode 0700) if missing.
func WriteKV(kv map[string]string, recipients []age.Recipient, s Scope) error {
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients: vault would be unreadable")
	}

	path, err := VaultPath(s)
	if err != nil {
		return err
	}

	if s == ScopeGlobal {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
	}

	var buf bytes.Buffer
	aw := armor.NewWriter(&buf)
	w, err := age.Encrypt(aw, recipients...)
	if err != nil {
		return err
	}
	for k, v := range kv {
		if _, err := fmt.Fprintf(w, "%s=%s\n", k, v); err != nil {
			return err
		}
	}
	if err := w.Close(); err != nil {
		return err
	}
	if err := aw.Close(); err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ValidateValue rejects values that contain literal newlines.
// Callers must base64-encode multi-line values (PEM keys, JSON blobs) before passing to envault set.
func ValidateValue(v string) error {
	if strings.ContainsAny(v, "\n\r") {
		return fmt.Errorf("value contains a newline — base64-encode it first:\n  printf '%%s' \"$VALUE\" | base64 | xargs -I{} envault set KEY={}")
	}
	return nil
}

func parseKV(r io.Reader) (map[string]string, error) {
	kv := make(map[string]string)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("malformed vault line: %q (expected KEY=VALUE)", line)
		}
		kv[parts[0]] = parts[1]
	}
	return kv, sc.Err()
}
