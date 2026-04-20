package vault

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
)

const RecipientsFile = ".env.vault.recipients"

// RecipientsPath returns the recipients file path for the given scope.
func RecipientsPath(s Scope) (string, error) {
	if s == ScopeLocal {
		return RecipientsFile, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, globalDirName, RecipientsFile), nil
}

// LoadRecipients reads the recipients file at the given scope.
// Returns (nil, nil, nil) if the file does not exist.
func LoadRecipients(s Scope) ([]age.Recipient, []string, error) {
	path, err := RecipientsPath(s)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	defer f.Close()

	var recipients []age.Recipient
	var pubkeys []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		r, err := age.ParseX25519Recipient(line)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid pubkey in %s: %q: %w", path, line, err)
		}
		recipients = append(recipients, r)
		pubkeys = append(pubkeys, line)
	}
	return recipients, pubkeys, sc.Err()
}

// SaveRecipients writes pubkeys to the recipients file at the given scope.
// For the global scope, the parent directory is created (mode 0700) if missing.
func SaveRecipients(pubkeys []string, s Scope) error {
	path, err := RecipientsPath(s)
	if err != nil {
		return err
	}
	if s == ScopeGlobal {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, pk := range pubkeys {
		if _, err := fmt.Fprintln(f, pk); err != nil {
			return err
		}
	}
	return nil
}
