package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/frahet/envault/internal/identity"
	"github.com/frahet/envault/internal/vault"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:                "run -- <command> [args...]",
	Short:              "Decrypt vault into memory and exec command with secrets as env vars",
	DisableFlagParsing: true, // required: Cobra must not consume the -- separator
	RunE:               runRun,
}

func runRun(cmd *cobra.Command, args []string) error {
	// Find the -- separator and extract the child command.
	sep := -1
	for i, a := range args {
		if a == "--" {
			sep = i
			break
		}
	}
	if sep == -1 || sep == len(args)-1 {
		return fmt.Errorf("usage: envault run -- <command> [args...]")
	}
	child := args[sep+1:]
	if len(child) == 0 {
		return fmt.Errorf("no command specified after --")
	}

	id, err := identity.Load()
	if err != nil {
		return err
	}

	kv, err := vault.ReadKV(id)
	if err != nil {
		return err
	}

	// Build env: start with current environment, strip ENVAULT_IDENTITY so the
	// private key is not inherited by the child process.
	env := make([]string, 0, len(os.Environ())+len(kv))
	for _, e := range os.Environ() {
		if len(e) >= 17 && e[:17] == "ENVAULT_IDENTITY=" {
			continue
		}
		env = append(env, e)
	}
	for k, v := range kv {
		env = append(env, k+"="+v)
	}

	// Resolve the binary before syscall.Exec (defer won't run after Exec).
	bin, err := exec.LookPath(child[0])
	if err != nil {
		return fmt.Errorf("%s: %w", child[0], err)
	}

	return syscall.Exec(bin, child, env)
}
