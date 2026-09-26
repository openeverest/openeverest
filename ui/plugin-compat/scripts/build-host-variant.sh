#!/usr/bin/env bash
# Builds the host UI with every @mui/* runtime package forced to <mui-version>
# and stores the result in .hosts/host-mui-<mui-version>. ui/package.json and
# pnpm-lock.yaml are edited only for the build and restored afterwards.
#
# Usage: build-host-variant.sh <mui-version>      (e.g. 6.5.0, 9.4.0)
#        build-host-variant.sh current            (snapshot the lockfile's MUI as-is)
set -euo pipefail

if [[ $# -ne 1 ]]; then
  sed -n '6,7p' "$0"
  exit 1
fi

COMPAT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
UI_DIR="$(cd "$COMPAT_DIR/.." && pwd)"
APP_DIR="$UI_DIR/apps/everest"
VERSION="$1"
LOG="$(mktemp -t host-build).log"

cd "$UI_DIR"
restore() { :; }
if [[ "$VERSION" != current ]]; then
  BACKUP="$(mktemp -d)"
  cp package.json pnpm-lock.yaml "$BACKUP/"
  restore() {
    cp "$BACKUP/package.json" "$BACKUP/pnpm-lock.yaml" "$UI_DIR/"
    (cd "$UI_DIR" && pnpm install --frozen-lockfile --silent >/dev/null 2>&1) || echo "WARN: run 'pnpm install' in ui/ to restore node_modules"
  }
  trap restore EXIT

  MAJOR="${VERSION%%.*}"
  node -e '
const fs = require("fs");
const [version, major] = process.argv.slice(1);
const pkg = JSON.parse(fs.readFileSync("package.json", "utf8"));
Object.assign(pkg.pnpm.overrides, {
  "@mui/material": version,
  "@mui/system": `^${major}`,
  "@mui/utils": `^${major}`,
  "@mui/styled-engine": `^${major}`,
  "@mui/icons-material": `^${major}`,
});
fs.writeFileSync("package.json", JSON.stringify(pkg, null, 2) + "\n");
' "$VERSION" "$MAJOR"
  echo "installing host deps with @mui/material=$VERSION (log: $LOG)"
  pnpm install --no-frozen-lockfile >"$LOG" 2>&1
fi

INSTALLED="$(cd "$APP_DIR" && node -p "require(require.resolve('@mui/material/package.json', { paths: ['.'] })).version" 2>/dev/null \
  || node -p "require('$APP_DIR/node_modules/@mui/material/package.json').version")"
LABEL="host-mui-${INSTALLED}"
OUT="$COMPAT_DIR/.hosts/$LABEL"

TYPECHECK=pass
echo "building host ($LABEL)"
if ! NODE_ENV=production pnpm turbo run build --filter '@percona/everest...' --force >>"$LOG" 2>&1; then
  # Type errors in the host are expected across MUI majors; what matters here is the runtime bundle.
  TYPECHECK=fail
  NODE_ENV=production pnpm turbo run build --filter '@percona/everest^...' --force >>"$LOG" 2>&1
  # Icons renamed in @mui/icons-material 9; mapped at build time so the host source stays untouched.
  cat >"$APP_DIR/icons-compat.mjs" <<'EOF'
export * from '@mui/icons-material/index';
export { default as DeleteOutline } from '@mui/icons-material/DeleteOutlined';
export { default as ErrorOutline } from '@mui/icons-material/ErrorOutlined';
export { default as PersonOutline } from '@mui/icons-material/PersonOutlined';
EOF
  cat >"$APP_DIR/vite.compat.config.ts" <<'EOF'
import { defineConfig, mergeConfig } from 'vite';
import base from './vite.config.ts';

export default defineConfig(
  mergeConfig(base, {
    resolve: {
      alias: [
        { find: /^@mui\/icons-material$/, replacement: new URL('./icons-compat.mjs', import.meta.url).pathname },
        { find: /^@mui\/icons-material\/ErrorOutline$/, replacement: '@mui/icons-material/ErrorOutlined' },
        { find: /^@mui\/icons-material\/PersonOutline$/, replacement: '@mui/icons-material/PersonOutlined' },
      ],
    },
  })
);
EOF
  trap 'rm -f "$APP_DIR/vite.compat.config.ts" "$APP_DIR/icons-compat.mjs"; restore' EXIT
  (cd "$APP_DIR" && NODE_ENV=production npx vite build --config vite.compat.config.ts --emptyOutDir >>"$LOG" 2>&1)
fi

rm -rf "$OUT"
mkdir -p "$(dirname "$OUT")"
cp -R "$APP_DIR/dist" "$OUT"
echo "{ \"mui\": \"$INSTALLED\", \"typecheck\": \"$TYPECHECK\" }" >"$OUT/host.json"
echo "built $OUT (typecheck: $TYPECHECK, log: $LOG)"
