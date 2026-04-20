package vault

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"filippo.io/age"
)

const RecipientsFile = ".env.vault.recipients"

// LoadRecipients reads .env.vault.recipients and returns parsed age recipients and their raw pubkey strings.
func LoadRecipients() ([]age.Recipient, []string, error) {
	f, err := os.Open(RecipientsFile)
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
			return nil, nil, fmt.Errorf("invalid pubkey in %s: %q: %w", RecipientsFile, line, err)
		}
		recipients = append(recipients, r)
		pubkeys = append(pubkeys, line)
	}
	return recipients, pubkeys, sc.Err()
}

// SaveRecipients writes pubkeys to .env.vault.recipients (one per line).
func SaveRecipients(pubkeys []string) error {
	f, err := os.Create(RecipientsFile)
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
