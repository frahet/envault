package vault

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
)

const VaultFile = ".env.vault"

// ReadKV decrypts the vault and returns all key-value pairs.
func ReadKV(id age.Identity) (map[string]string, error) {
	f, err := os.Open(VaultFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no vault found — run `envault init`")
		}
		return nil, err
	}
	defer f.Close()

	ar := armor.NewReader(f)
	r, err := age.Decrypt(ar, id)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return parseKV(r)
}

// WriteKV encrypts kv and writes the vault atomically.
// The temp file is placed in the same directory as .env.vault to ensure os.Rename is atomic.
func WriteKV(kv map[string]string, recipients []age.Recipient) error {
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients: vault would be unreadable")
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

	tmp := filepath.Join(filepath.Dir(absVaultPath()), ".env.vault.tmp")
	if err := os.WriteFile(tmp, buf.Bytes(), 0600); err != nil {
		return err
	}
	return os.Rename(tmp, VaultFile)
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

// absVaultPath returns the absolute path to .env.vault in the current directory.
func absVaultPath() string {
	abs, err := filepath.Abs(VaultFile)
	if err != nil {
		return VaultFile
	}
	return abs
}
