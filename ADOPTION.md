# Adopting envault

Step-by-step runbook for migrating a project from plaintext `.env` files to encrypted vaults.

## Why

Plaintext `.env` files on disk and in git history are a standing liability — anything that reads your filesystem (rogue dependency, leaked backup, screen share, AI agent) reads your secrets. envault keeps secrets encrypted at rest with age (X25519 + ChaCha20-Poly1305) and decrypts only into the env of the process that needs them.

## Per-project migration checklist

Run from inside the project's working tree.

- [ ] **1. Init the local vault**
      ```sh
      envault init
      ```
      Creates `./.env.vault` (encrypted) and `./.env.vault.recipients` (committable list of pubkeys allowed to decrypt). Reuses your existing identity at `~/Library/Application Support/envault/identity.age` (macOS) or `~/.config/envault/identity.age` (Linux).

- [ ] **2. Move every secret from `.env` into the vault**
      For each line `KEY=VALUE` in the current plaintext file:
      ```sh
      envault set KEY < /dev/stdin <<< "VALUE"
      # or, if the value is already in your global vault:
      envault export KEY
      ```
      `envault export` is the fast path when the secret already exists in your personal global vault — no re-pasting, no clipboard exposure.

- [ ] **3. gitignore the plaintext files**
      Add to `.gitignore`:
      ```
      .env
      .env.local
      .env.production
      ```

- [ ] **4. Commit the encrypted artifacts**
      ```sh
      git add .env.vault .env.vault.recipients .gitignore
      git commit -m "secrets: migrate to envault"
      ```

- [ ] **5. Wrap entrypoints with `envault run`**
      Anywhere the project shells out to a binary that reads env vars, prefix it. See [CI/CD patterns](#cicd-patterns) below.

- [ ] **6. Delete the plaintext files**
      ```sh
      rm .env .env.local .env.production 2>/dev/null
      ```
      Do **not** `git rm` them — if they were ever committed, the git history already leaks. Use [git-filter-repo](https://github.com/newren/git-filter-repo) or BFG and force-push if rotation is needed.

- [ ] **7. Rotate any secret that ever touched `git log`**
      Removing a file from a future commit doesn't revoke leaked values. Treat them as compromised and rotate at the issuing service (Stripe, Anthropic, AWS, etc.).

- [ ] **8. Deploy and verify**
      Pull a preview/canary, confirm services come up green, then promote.

## CI/CD patterns

In environments where the identity file can't sit on disk, set the `ENVAULT_IDENTITY` env var to the file's contents. envault reads it in preference to the file. Multiline secrets in GitHub Actions are handled transparently.

### Vercel (Next.js, build-time env)

In the project's Vercel dashboard → Settings → Environment Variables, set `ENVAULT_IDENTITY` to the contents of your identity file. Then wrap build/start commands:

```json
{
  "scripts": {
    "dev": "envault run -- next dev",
    "build": "envault run -- next build",
    "start": "envault run -- next start"
  }
}
```

### GitHub Actions

```yaml
- name: Build
  env:
    ENVAULT_IDENTITY: ${{ secrets.ENVAULT_IDENTITY }}
  run: envault run -- pnpm build
```

Multiline private keys in Actions get newline-escaped automatically — envault handles both forms.

### systemd

```ini
[Service]
Environment=ENVAULT_IDENTITY=...
ExecStart=/usr/local/bin/envault run -- /opt/myapp/server
```

Prefer `EnvironmentFile=/etc/myapp.env` with `0600` perms over inline `Environment=` for the identity itself.

### Docker / docker-compose

```yaml
services:
  app:
    environment:
      ENVAULT_IDENTITY: ${ENVAULT_IDENTITY}
    command: envault run -- /app/server
```

Pass `ENVAULT_IDENTITY` from the host or a secrets backend; never bake it into the image.

## Project rollout order

Migrate high-blast-radius projects first — anything with payment, key-custody, or production write access.

| Project | Sensitive keys |
|---------|----------------|
| `tradebot` | `PRIVATE_KEY`, `BINANCE_API_KEY`, `BINANCE_API_SECRET` |
| `novoorion.ai` | Supabase service-role keys, MetaMask config |
| `crypto-tax` | `BINANCE_API_KEY`, `BINANCE_API_SECRET`, `ANTHROPIC_API_KEY` |
| `novo-token` | `PRIVATE_KEY`, `BASESCAN_API_KEY` |
| `huginn` | Telegram token, Claude API key, Muninn credentials |
| `gitpulse` | GitHub PAT |

File a migration issue in each target repo referencing this guide — keep envault itself out of the per-project loop.

## Common pitfalls

- **Don't commit the identity file.** It's the root key for decryption. Permissions are `0600` for a reason.
- **Don't share the same identity across team members.** Each teammate runs `envault init` once, sends you their pubkey, and you `envault add-recipient` to grant access.
- **`add-recipient` is forward-only.** Anyone who has ever decrypted the vault still has every old value. Rotate after `remove-recipient`.
- **`envault get` writes to shell history.** Use it for one-off debugging; prefer `envault run` for automation.
- **Values with literal newlines (PEM keys, JSON blobs) must be base64-encoded** before `envault set`. envault hard-errors otherwise.
- **`.env.vault.recipients` is the source of truth for who can decrypt.** Editing it by hand without `add-recipient`/`remove-recipient` won't re-encrypt the vault — the new pubkey list becomes a lie.
