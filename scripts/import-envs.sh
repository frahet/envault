#!/usr/bin/env bash
# import-envs.sh — scan ~/Documents/projects/ for .env files and import to envault vault.
#
# Usage: ./import-envs.sh [projects-dir]
#
# Run this from the directory that contains (or will contain) your .env.vault.
# The script finds all .env files, shows you what it found, and imports all keys.
# If the same KEY appears in multiple files, the NEWEST file's value wins.

set -euo pipefail

PROJECTS_DIR="${1:-$HOME/Documents/projects}"
ENVAULT=$(command -v envault 2>/dev/null || echo "$HOME/Documents/projects/envault/envault")

# ── sanity checks ──────────────────────────────────────────────────────────────

if [ ! -x "$ENVAULT" ]; then
    echo "error: envault not found. Build it first:"
    echo "  cd ~/Documents/projects/envault && go build -o envault . && sudo cp envault /usr/local/bin/"
    exit 1
fi

if [ ! -f ".env.vault" ]; then
    echo "No .env.vault in $(pwd)."
    echo "Run:  $ENVAULT init"
    exit 1
fi

# ── find .env files ────────────────────────────────────────────────────────────

# Patterns to include
ENV_PATTERNS=( ".env" ".env.local" ".env.development" ".env.production" ".env.staging" ".env.example" )

# Directories to skip
SKIP_DIRS=( "node_modules" ".git" "dist" ".next" ".nuxt" "__pycache__" ".venv" "venv" "vendor" )

BUILD_FIND_CMD='find '"$PROJECTS_DIR"' \( -false'
for d in "${SKIP_DIRS[@]}"; do
    BUILD_FIND_CMD+=" -o -name $d"
done
BUILD_FIND_CMD+=' \) -prune -o \( -false'
for p in "${ENV_PATTERNS[@]}"; do
    BUILD_FIND_CMD+=" -o -name $p"
done
BUILD_FIND_CMD+=' \) -print'

mapfile -t ENV_FILES < <(eval "$BUILD_FIND_CMD" | sort)

if [ ${#ENV_FILES[@]} -eq 0 ]; then
    echo "No .env files found in $PROJECTS_DIR"
    exit 0
fi

# ── summary table ──────────────────────────────────────────────────────────────

echo ""
echo "Found ${#ENV_FILES[@]} .env file(s) in $PROJECTS_DIR:"
echo ""
printf "  %-55s %-18s %s\n" "File" "Modified" "Keys"
printf "  %-55s %-18s %s\n" "$(printf '%0.s-' {1..55})" "$(printf '%0.s-' {1..18})" "-----"

for f in "${ENV_FILES[@]}"; do
    if [[ "$OSTYPE" == "darwin"* ]]; then
        mtime=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M" "$f" 2>/dev/null || echo "unknown")
    else
        mtime=$(stat -c "%y" "$f" 2>/dev/null | cut -c1-16 || echo "unknown")
    fi
    keys=$(grep -cE '^[^#[:space:]][^=]*=' "$f" 2>/dev/null || echo 0)
    short="${f/$HOME/~}"
    printf "  %-55s %-18s %s\n" "$short" "$mtime" "$keys"
done

echo ""
echo "  Newest file's value wins when the same KEY appears in multiple files."
echo ""
read -rp "  Proceed with analysis? [y/N] " confirm
if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "Aborted."
    exit 0
fi

# ── sort by mtime (oldest first so newest overwrites) ─────────────────────────

if [[ "$OSTYPE" == "darwin"* ]]; then
    mapfile -t SORTED_FILES < <(
        for f in "${ENV_FILES[@]}"; do
            stat -f "%m %N" "$f" 2>/dev/null
        done | sort -n | awk '{print $2}'
    )
else
    mapfile -t SORTED_FILES < <(
        for f in "${ENV_FILES[@]}"; do
            stat -c "%Y %n" "$f" 2>/dev/null
        done | sort -n | awk '{print $2}'
    )
fi

# ── parse all files ────────────────────────────────────────────────────────────

declare -A KV
declare -A KEY_SOURCE
declare -A KEY_MTIME
SKIPPED_MULTILINE=()

for f in "${SORTED_FILES[@]}"; do
    if [[ "$OSTYPE" == "darwin"* ]]; then
        file_mtime=$(stat -f "%m" "$f" 2>/dev/null || echo 0)
    else
        file_mtime=$(stat -c "%Y" "$f" 2>/dev/null || echo 0)
    fi

    while IFS= read -r line || [[ -n "$line" ]]; do
        # Skip empty lines and comments
        [[ -z "${line// }" ]] && continue
        [[ "$line" =~ ^[[:space:]]*# ]] && continue

        # Strip 'export ' prefix
        line="${line#export }"

        # Must contain =
        [[ "$line" != *"="* ]] && continue

        key="${line%%=*}"
        val="${line#*=}"

        # Skip invalid keys
        [[ -z "$key" ]] && continue
        [[ "$key" =~ [[:space:]] ]] && continue
        [[ "$key" =~ [^A-Za-z0-9_] ]] && continue

        # Strip surrounding quotes
        if [[ "$val" =~ ^\"(.*)\"$ ]]; then
            val="${BASH_REMATCH[1]}"
        elif [[ "$val" =~ ^\'(.*)\'$ ]]; then
            val="${BASH_REMATCH[1]}"
        fi

        # Detect multiline/PEM values — envault can't store literal newlines
        if [[ "$val" == *"\\n"* || "$val" == "-----BEGIN"* ]]; then
            SKIPPED_MULTILINE+=("$key (${f/$HOME/~})")
            continue
        fi

        KV["$key"]="$val"
        KEY_SOURCE["$key"]="${f/$HOME/~}"
        KEY_MTIME["$key"]="$file_mtime"
    done < "$f"
done

# ── preview what will be imported ─────────────────────────────────────────────

echo ""
if [ ${#KV[@]} -eq 0 ]; then
    echo "No importable keys found (all empty or malformed)."
    exit 0
fi

echo "  Keys to import (${#KV[@]} total):"
echo ""
for key in $(printf '%s\n' "${!KV[@]}" | sort); do
    printf "  %-35s from %s\n" "$key" "${KEY_SOURCE[$key]}"
done

if [ ${#SKIPPED_MULTILINE[@]} -gt 0 ]; then
    echo ""
    echo "  Skipped (multiline/PEM — base64-encode these manually):"
    for item in "${SKIPPED_MULTILINE[@]}"; do
        echo "    $item"
    done
fi

echo ""
read -rp "  Import all ${#KV[@]} key(s) into vault? [y/N] " confirm2
if [[ "$confirm2" != "y" && "$confirm2" != "Y" ]]; then
    echo "Aborted."
    exit 0
fi

# ── import ─────────────────────────────────────────────────────────────────────

echo ""
imported=0
skipped=0

for key in $(printf '%s\n' "${!KV[@]}" | sort); do
    val="${KV[$key]}"
    if "$ENVAULT" set "$key=$val" 2>/dev/null; then
        printf "  ✓ %s\n" "$key"
        ((imported++)) || true
    else
        printf "  ✗ %s  (skipped — run manually: envault set %s=<value>)\n" "$key" "$key"
        ((skipped++)) || true
    fi
done

echo ""
echo "  Imported: $imported    Skipped: $skipped"
echo ""
echo "  Verify:   $ENVAULT list"
echo "  Test:     $ENVAULT run -- <your-dev-command>"
echo ""
echo "  When satisfied:"
echo "    echo '.env' >> .gitignore"
echo "    echo '.env.local' >> .gitignore"
echo "    rm .env .env.local   # delete originals"
echo "    git add .env.vault .env.vault.recipients"
echo "    git commit -m 'chore: migrate secrets to envault'"
