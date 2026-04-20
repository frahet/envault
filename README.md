# envault

Age-encrypted `.env` replacement — secrets that live safely in git, never as plaintext on disk.

Pure Go CLI. No server. Uses [filippo.io/age](https://github.com/FiloSottile/age) for authenticated encryption.

## Install

### macOS / Linuxbrew

```sh
brew install frahet/tap/envault
```

### Linux (amd64) — direct tarball

```sh
mkdir -p ~/.local/bin
curl -sSL https://github.com/frahet/envault/releases/latest/download/envault_0.1.1_linux_amd64.tar.gz -o /tmp/envault.tar.gz
tar xzf /tmp/envault.tar.gz -C ~/.local/bin envault
chmod +x ~/.local/bin/envault
envault version
```

If `envault` is not found, add `~/.local/bin` to your `PATH` (e.g. `echo 'export PATH=$HOME/.local/bin:$PATH' >> ~/.zshrc`). System-wide install: `sudo install -m 0755 /tmp/envault /usr/local/bin/envault`.

### From source

```sh
go install github.com/frahet/envault@latest
```

## Quickstart

First time on a new machine:

```sh
envault init --global
envault set --global ANTHROPIC_API_KEY=sk-ant-...
envault set --global GITHUB_TOKEN=ghp_...
envault list
```

Run any command with secrets injected as env vars:

```sh
envault run -- pnpm dev
envault run -- python bot.py
```

That's it. Your global vault lives at `~/.envault/.env.vault` and is available from every directory on this machine.

## Local vs global scope

envault has two vaults:

| Scope | Path | Purpose |
|-------|------|---------|
| **Global** | `~/.envault/.env.vault` | Personal keys shared across every project (Anthropic, GitHub, etc.). Per-user, per-machine. |
| **Local** | `./.env.vault` | Project-specific secrets committed to the repo (e.g. Next.js build-time env decrypted on Vercel). |

**Reads** (`list`, `get`, `run`) merge both scopes automatically — **local wins** on key collision. `list` annotates each key `[local]` or `[global]`.

**Writes** (`set`, `add-recipient`, `remove-recipient`) default to **local** if a local vault exists in the current directory, otherwise **global**. Pass `--global` to force global.

To opt a project into a committable local vault:

```sh
cd ~/projects/my-app
envault init                      # creates ./.env.vault for this project
envault set DATABASE_URL=postgres://...   # writes local
git add .env.vault .env.vault.recipients
git commit -m "add encrypted secrets"
```

## Commands

| Command | Description |
|---------|-------------|
| `envault init [--global]` | Create a new vault (local by default, `--global` for personal). Reuses existing identity. |
| `envault set KEY=VALUE [--global]` | Encrypt and store a secret. |
| `envault get KEY` | Decrypt and print one value (merged read). |
| `envault list` | List all keys (values redacted, scope annotated). |
| `envault run -- <cmd>` | Run `<cmd>` with all vault keys injected as env vars. |
| `envault add-recipient <age1...> [--global]` | Re-encrypt vault for an additional pubkey. |
| `envault remove-recipient <age1...> [--global]` | Remove a recipient. Rotate secrets after! |
| `envault whoami` | Print identity source + public key. |
| `envault pubkey` | Print public key only (scriptable). |
| `envault version` | Print version. |

## Sharing secrets with a teammate or another machine

Have them send you their age public key (`envault pubkey` on their side). Then:

```sh
envault add-recipient age1abc...
git add .env.vault .env.vault.recipients
git commit -m "share vault with alice"
git push
```

They pull, run `envault list`, and can now decrypt.

To sync your *global* vault across your own machines, make `~/.envault/` itself a private git repo (e.g. `frahet/dotsecrets`) and clone it on each machine.

## Identity

Your age private key lives at:

- macOS: `~/Library/Application Support/envault/identity.age`
- Linux: `~/.config/envault/identity.age`

**Never commit this file.** It's protected by file permissions (`0600`) and is the root of your ability to decrypt.

### CI / containers / systemd

Where a file on disk is impractical, set the `ENVAULT_IDENTITY` environment variable to the contents of the identity file. envault reads it in preference to the file. GitHub Actions multiline secrets are handled transparently.

```yaml
env:
  ENVAULT_IDENTITY: ${{ secrets.ENVAULT_IDENTITY }}
run: envault run -- pnpm build
```

## Security notes

- The identity file is **unencrypted at rest**. Accepted tradeoff for v0 — v1 will add keychain integration.
- `envault get` prints to stdout, which writes to shell history. Use it as an escape hatch; prefer `envault run` for automated flows.
- `remove-recipient` only prevents *future* access. The removed recipient can still decrypt historical `.env.vault` from git history. **Rotate all secrets after removal** to fully revoke.
- Values containing literal newlines are rejected — base64-encode them first (`printf '%s' "$VALUE" | base64`).

## License

MIT. Issues and roadmap: [github.com/frahet/envault/issues](https://github.com/frahet/envault/issues).
