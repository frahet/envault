# CLAUDE.md — envault

Pure Go CLI for .env encryption using `filippo.io/age`. Never writes plaintext to disk.

## Commands
```bash
go build -o envault .   # build binary
go test ./...            # run tests
go vet ./...             # vet
```

## Architecture
```
main.go                         -- entry point
cmd/                            -- cobra commands
  root.go                       -- root + command registration
  init.go                       -- generate identity, create vault
  set.go                        -- encrypt and store KEY=VALUE
  get.go                        -- decrypt and print one key
  list.go                       -- list key names (values redacted)
  run.go                        -- decrypt into memory, syscall.Exec child
  add_recipient.go              -- add age pubkey, rewrap vault
  remove_recipient.go           -- remove age pubkey, rewrap vault
  version.go                    -- print version (set via GoReleaser ldflags)
internal/identity/identity.go   -- load age identity from file or ENVAULT_IDENTITY env var
internal/vault/vault.go         -- encrypt/decrypt vault, atomic write
internal/vault/recipients.go    -- read/write .env.vault.recipients
```

## Key implementation constraints

- `cmd/run.go`: `DisableFlagParsing = true` on the run command — Cobra must not consume `--`
- `internal/vault/vault.go`: temp file written to same dir as `.env.vault` (not `$TMPDIR`) for atomic rename
- `internal/vault/vault.go`: KEY=VALUE parsed with `SplitN(line, "=", 2)` — values may contain `=`
- `cmd/run.go`: strip `ENVAULT_IDENTITY` from child env before `syscall.Exec`
- `cmd/run.go`: all cleanup must complete before `syscall.Exec` — `defer` won't run after Exec
- `internal/vault/vault.go`: hard-error on values with literal newlines (ValidateValue)

## Vault format
- `.env.vault` — armored age ciphertext (age -a flag via `armor.NewWriter`)
- `.env.vault.recipients` — one age pubkey per line, committed to repo
- Plaintext inside encryption: line-delimited `KEY=VALUE`, UTF-8, one per line
- Values with literal newlines must be base64-encoded by caller

## Identity
- `~/.config/envault/identity.age` — age private key, unencrypted (OS permissions guard)
- `ENVAULT_IDENTITY` env var — overrides file; for CI/CD (GitHub Actions secrets)
  - GitHub Actions injects multiline as literal `\n` — identity loader handles both forms
  - Strip this env var from child process in `envault run`

## Security decisions (explicit)
- Identity file is unencrypted at rest — accepted for v0; v1 adds keychain integration
- `envault get` prints to stdout — writes shell history; escape hatch only
- `remove-recipient` only stops future access — rotate secrets after removal to fully revoke

## Distribution (TODO — before first tag)
- GoReleaser `.goreleaser.yaml` — macOS arm64/amd64, Linux amd64
- Homebrew tap: `frahet/homebrew-tap`
- Version injected via ldflags: `-X github.com/frahet/envault/cmd.Version={{.Version}}`

## Skill routing
- Bugs, errors → /investigate
- Ready to ship → /ship
- Code review → /review
- Architecture questions → /plan-eng-review
