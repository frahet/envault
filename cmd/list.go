package cmd

import (
	"fmt"
	"sort"

	"github.com/frahet/envault/internal/identity"
	"github.com/frahet/envault/internal/vault"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secret key names (values redacted, annotated with source scope)",
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	id, err := identity.Load()
	if err != nil {
		return err
	}

	kv, source, err := vault.ReadMerged(id)
	if err != nil {
		return err
	}

	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Printf("%s=<redacted> [%s]\n", k, source[k])
	}
	return nil
}
