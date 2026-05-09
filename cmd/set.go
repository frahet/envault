package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"filippo.io/age"
	"github.com/frahet/envault/internal/identity"
	"github.com/frahet/envault/internal/vault"
	"github.com/spf13/cobra"
)

var setGlobalFlag bool

var setCmd = &cobra.Command{
	Use:   "set KEY=VALUE  (or: set KEY  to read VALUE from stdin)",
	Short: "Encrypt and store a secret (overwrites existing key)",
	Long: `Encrypt and store a secret (overwrites existing key).

Two input modes:
  envault set KEY=VALUE                    # value on argv
  envault set KEY < file                   # value from stdin
  envault set KEY <<< "$VALUE"             # value from a shell variable
  printf '%s' "$VALUE" | envault set KEY   # value from a pipe

Stdin mode is preferred for secrets: argv is briefly visible in /proc and to
ps(1); stdin is not.`,
	Args: cobra.ExactArgs(1),
	RunE: runSet,
}

func init() {
	setCmd.Flags().BoolVar(&setGlobalFlag, "global", false, "write to the global vault (~/.envault/) instead of local")
}

func runSet(cmd *cobra.Command, args []string) error {
	key, val, err := parseSetArg(args[0])
	if err != nil {
		return err
	}
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	if err := vault.ValidateValue(val); err != nil {
		return err
	}

	scope := scopeForWrite(setGlobalFlag)

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

	kv[key] = val

	recipients, _, err := vault.LoadRecipients(scope)
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		recipients = []age.Recipient{id.Recipient()}
	}

	return vault.WriteKV(kv, recipients, scope)
}

// parseSetArg returns (key, value). With KEY=VALUE, value comes from argv.
// With bare KEY, value comes from stdin; trailing \r and \n are stripped so
// `<<<` here-strings and most pipes work as expected.
func parseSetArg(arg string) (string, string, error) {
	if strings.Contains(arg, "=") {
		parts := strings.SplitN(arg, "=", 2)
		return parts[0], parts[1], nil
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", "", fmt.Errorf("read value from stdin: %w", err)
	}
	val := strings.TrimRight(string(data), "\r\n")
	if val == "" {
		return "", "", fmt.Errorf("no value provided: pass `KEY=VALUE` on argv, or pipe a value to stdin")
	}
	return arg, val, nil
}
