package cmd

import (
	"errors"
	"fmt"
	"sort"

	"filippo.io/age"
	"github.com/frahet/envault/internal/identity"
	"github.com/frahet/envault/internal/vault"
	"github.com/spf13/cobra"
)

var (
	exportAllFlag   bool
	exportForceFlag bool
)

var exportCmd = &cobra.Command{
	Use:   "export [KEY...]",
	Short: "Copy keys from the global vault into the local vault",
	Long: `Copy keys from the global vault into the local vault.

Useful when bootstrapping a project's committable secrets from your personal
global vault — e.g. Next.js build env that ships to Vercel, a Docker image,
or a service's deploy-time secrets.

If no local vault exists yet, one is created using your current identity as
the sole recipient. Use add-recipient afterward to share with teammates or CI.

Existing local values are preserved by default. Pass --force to overwrite.`,
	Example: `  envault export DATABASE_URL STRIPE_KEY
  envault export --all
  envault export --force DATABASE_URL`,
	Args:              cobra.ArbitraryArgs,
	ValidArgsFunction: completeUnselectedGlobalKeys,
	RunE:              runExport,
}

func init() {
	exportCmd.Flags().BoolVar(&exportAllFlag, "all", false, "export every key from the global vault")
	exportCmd.Flags().BoolVar(&exportForceFlag, "force", false, "overwrite keys that already exist in the local vault")
}

func completeUnselectedGlobalKeys(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	id, err := identity.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	kv, err := vault.ReadKV(id, vault.ScopeGlobal)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	selected := map[string]bool{}
	for _, a := range args {
		selected[a] = true
	}
	keys := make([]string, 0, len(kv))
	for k := range kv {
		if !selected[k] {
			keys = append(keys, k)
		}
	}
	return keys, cobra.ShellCompDirectiveNoFileComp
}

func runExport(cmd *cobra.Command, args []string) error {
	if exportAllFlag && len(args) > 0 {
		return fmt.Errorf("--all conflicts with explicit KEY arguments")
	}
	if !exportAllFlag && len(args) == 0 {
		return fmt.Errorf("specify one or more KEYs or pass --all")
	}

	id, err := identity.Load()
	if err != nil {
		return err
	}

	globalKV, err := vault.ReadKV(id, vault.ScopeGlobal)
	if err != nil {
		if errors.Is(err, vault.ErrNoVault) {
			return fmt.Errorf("no global vault found — run `envault init --global` first")
		}
		return err
	}

	var keysToExport []string
	if exportAllFlag {
		for k := range globalKV {
			keysToExport = append(keysToExport, k)
		}
	} else {
		for _, k := range args {
			if _, ok := globalKV[k]; !ok {
				return fmt.Errorf("key not found in global vault: %s", k)
			}
			keysToExport = append(keysToExport, k)
		}
	}
	sort.Strings(keysToExport)

	localKV, localExisted, err := readLocalForExport(id)
	if err != nil {
		return err
	}

	var clashes []string
	for _, k := range keysToExport {
		if _, exists := localKV[k]; exists {
			clashes = append(clashes, k)
		}
	}
	if len(clashes) > 0 && !exportForceFlag {
		return fmt.Errorf("these keys already exist in the local vault — pass --force to overwrite: %v", clashes)
	}

	for _, k := range keysToExport {
		localKV[k] = globalKV[k]
	}

	recipients, err := localRecipientsOrIdentity(id)
	if err != nil {
		return err
	}

	if err := vault.WriteKV(localKV, recipients, vault.ScopeLocal); err != nil {
		return err
	}
	if !localExisted {
		if err := vault.SaveRecipients([]string{id.Recipient().String()}, vault.ScopeLocal); err != nil {
			return err
		}
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "exported %d key(s) from global into local vault\n", len(keysToExport))
	if len(clashes) > 0 {
		fmt.Fprintf(out, "overwrote %d existing local value(s): %v\n", len(clashes), clashes)
	}
	if !localExisted {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Commit to git:      .env.vault (encrypted secrets), .env.vault.recipients (who can decrypt).")
		fmt.Fprintln(out, "Add to .gitignore:  .env, .env.local, .env.production (never commit plaintext).")
	}
	return nil
}

func readLocalForExport(id *age.X25519Identity) (map[string]string, bool, error) {
	kv, err := vault.ReadKV(id, vault.ScopeLocal)
	if err == nil {
		return kv, true, nil
	}
	if errors.Is(err, vault.ErrNoVault) {
		return map[string]string{}, false, nil
	}
	return nil, false, err
}

func localRecipientsOrIdentity(id *age.X25519Identity) ([]age.Recipient, error) {
	recipients, _, err := vault.LoadRecipients(vault.ScopeLocal)
	if err != nil {
		return nil, err
	}
	if len(recipients) == 0 {
		recipients = []age.Recipient{id.Recipient()}
	}
	return recipients, nil
}
