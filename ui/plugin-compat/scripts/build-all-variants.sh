#!/usr/bin/env bash
# Builds every plugin bundle used by the suite into .variants/ (plugin-hub is
# expected next to the core checkout, override with PLUGIN_HUB_DIR).
set -euo pipefail

COMPAT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PLUGIN_HUB_DIR="${PLUGIN_HUB_DIR:-$COMPAT_DIR/../../../plugin-hub}"
MUI_VERSIONS=(${MUI_VERSIONS:-5.18.0 6.5.0 7.3.11 9.4.0})
# Published-ui-lib scenarios: MUI pins that differ from ui-lib's own dependency.
PACK_VERSIONS=(${PACK_VERSIONS:-6.5.0 9.4.0})

build() { "$COMPAT_DIR/scripts/build-plugin-variant.sh" "$@" 2>&1 | grep -E '^built|rror' | cut -c1-160; }

rm -rf "$COMPAT_DIR/.variants"
for plugin in "$COMPAT_DIR/canary" "$PLUGIN_HUB_DIR"; do
  for v in "${MUI_VERSIONS[@]}"; do build "$plugin" "$v"; done
  for v in "${PACK_VERSIONS[@]}"; do build "$plugin" "$v" --ui-lib pack --no-dedupe; done
done
