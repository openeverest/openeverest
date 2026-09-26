#!/usr/bin/env bash
# Runs the suite for every host in .hosts/ against the frozen plugin bundles in
# .variants/ and prints the matrix. Bundles are hash-checked first so a run can
# never silently test rebuilt plugins.
set -uo pipefail

COMPAT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$COMPAT_DIR"

if [[ ! -f .variants/MANIFEST.sha256 ]]; then
  (cd .variants && find . -name main.js | sort | xargs shasum -a 256 >MANIFEST.sha256)
  echo "froze $(wc -l <.variants/MANIFEST.sha256 | tr -d ' ') plugin bundles"
fi
(cd .variants && shasum -a 256 --quiet -c MANIFEST.sha256) || { echo "plugin bundles changed since they were frozen" >&2; exit 1; }

echo "== static bundle checks"
node scripts/check-bundle.mjs .variants/*/* | cut -c1-160

rm -rf results
for host in .hosts/*/; do
  label="$(basename "$host")"
  echo "== $label"
  HOST_LABEL="$label" HOST_DIST="$COMPAT_DIR/$host" PLAYWRIGHT_JSON_OUTPUT_FILE="results/$label.json" \
    npx playwright test --reporter=dot,json | tail -3
done

node scripts/summarize.mjs results/*.json
