package cmd

import (
	"fmt"
	"os"

	"github.com/frahet/envault/internal/identity"
	"github.com/frahet/envault/internal/vault"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:               "get KEY",
	Short:             "Decrypt and print a single secret value (merged local+global, local wins)",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeMergedKeys,
	RunE:              runGet,
}

func completeMergedKeys(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	id, err := identity.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	kv, _, err := vault.ReadMerged(id)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	return keys, cobra.ShellCompDirectiveNoFileComp
}

func runGet(cmd *cobra.Command, args []string) error {
	fmt.Fprintln(os.Stderr, "warning: secret value printed to stdout — add `envault get *` to HISTIGNORE to prevent shell history capture")

	id, err := identity.Load()
	if err != nil {
		return err
	}

	kv, _, err := vault.ReadMerged(id)
	if err != nil {
		return err
	}

	v, ok := kv[args[0]]
	if !ok {
		return fmt.Errorf("key not found: %s", args[0])
	}

	fmt.Println(v)
	return nil
}
