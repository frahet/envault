# Migrate this project from .env to envault

Paste this entire document into a new Claude session in the target project directory.

---

## What envault is

`envault` is a Go CLI that encrypts `.env` secrets at rest using `filippo.io/age`.
Secrets live only in `.env.vault` (armored age ciphertext, safe to commit).
`envault run -- <cmd>` decrypts into memory and execs the child process — plaintext never touches disk.

Binary: `~/Documents/projects/envault/envault`  
Docs: `~/Documents/projects/envault/CLAUDE.md`

## Your task

Migrate this project from plain `.env` files to envault. Follow these steps exactly.

### Step 1 — check if a vault already exists

```bash
ls .env.vault 2>/dev/null && echo "vault exists" || echo "no vault yet"
```

If no vault: `~/Documents/projects/envault/envault init`

### Step 2 — import all .env files in this project

```bash
cd <project-root>
~/Documents/projects/envault/scripts/import-envs.sh .
```

The script finds every `.env`, `.env.local`, `.env.development`, `.env.production` file
in the current directory, shows you a preview, and imports all keys into the vault.

If the script is not found: run `chmod +x ~/Documents/projects/envault/scripts/import-envs.sh` first.

### Step 3 — verify all secrets imported correctly

```bash
~/Documents/projects/envault/envault list
```

Check that every key from your `.env` files is present.

### Step 4 — test the dev workflow

Replace whatever command starts your dev server with `envault run -- <that command>`.

Examples:
- `pnpm dev` → `~/Documents/projects/envault/envault run -- pnpm dev`
- `python manage.py runserver` → `~/Documents/projects/envault/envault run -- python manage.py runserver`
- `go run ./cmd/...` → `~/Documents/projects/envault/envault run -- go run ./cmd/...`

Verify the app starts and can read environment variables.

### Step 5 — update package.json / Makefile / scripts

If `package.json` has a `dev` script, update it:

```json
"scripts": {
  "dev": "envault run -- next dev",
  "dev:direct": "next dev"
}
```

If there is a `Makefile`:
```makefile
dev:
    envault run -- <original command>
```

### Step 6 — protect .gitignore

Make sure `.env` files are in `.gitignore`:

```bash
grep -q "^\.env$" .gitignore || echo ".env" >> .gitignore
grep -q "^\.env\.local$" .gitignore || echo ".env.local" >> .gitignore
grep -q "^\.env\.production$" .gitignore || echo ".env.production" >> .gitignore
```

### Step 7 — commit the vault, delete the plaintext

```bash
git add .env.vault .env.vault.recipients
git rm --cached .env .env.local .env.production 2>/dev/null || true
rm -f .env .env.local .env.production
git commit -m "chore: migrate secrets to envault"
```

### Step 8 — update CLAUDE.md (if it exists)

Add a note to the project's CLAUDE.md:

```markdown
## Secrets
Secrets are managed with [envault](https://github.com/frahet/envault).
- `.env.vault` — encrypted secrets (committed)
- `.env.vault.recipients` — who can decrypt (committed)
- Run dev: `envault run -- <dev command>`
- Add a secret: `envault set KEY=value`
- CI: set `ENVAULT_IDENTITY` secret in GitHub Actions (content of `~/.config/envault/identity.age`)
```

### Multiline values (PEM keys, JSON blobs)

The import script skips values with literal newlines. Handle them manually:

```bash
# PEM key
printf '%s' "$(cat private_key.pem)" | base64 | envault set PRIVATE_KEY=$(base64 private_key.pem)
# Or: envault set PRIVATE_KEY=$(base64 private_key.pem | tr -d '\n')
```

### CI/CD (GitHub Actions)

Add `ENVAULT_IDENTITY` to GitHub Actions secrets — paste the contents of
`~/.config/envault/identity.age`. Then in your workflow:

```yaml
- name: Run with secrets
  run: envault run -- pnpm build
  env:
    ENVAULT_IDENTITY: ${{ secrets.ENVAULT_IDENTITY }}
```
