#!/usr/bin/env bash
# Builds a ui-lib plugin against a pinned @mui/material version so the bundle can
# be loaded under a host running a different MUI (plugin/host independence test).
#
# Usage: build-plugin-variant.sh <plugin-dir> <mui-version> [--ui-lib link|pack] [--no-dedupe]
#   --ui-lib link  consume ui-lib from this checkout (default, like local dev)
#   --ui-lib pack  consume an `npm pack`ed ui-lib (what a third-party plugin gets from npm)
#   --no-dedupe    drop resolve.dedupe, as a plugin author who didn't copy it would
# Output: .variants/<plugin>/<variant>/{main.js,main.js.map,variant.json}
set -euo pipefail

if [[ $# -lt 2 ]]; then
  sed -n '5,9p' "$0"
  exit 1
fi

PLUGIN_DIR="$(cd "$1" && pwd)"
MUI_VERSION="$2"
shift 2
UI_LIB_MODE=link
DEDUPE=1
while [[ $# -gt 0 ]]; do
  case "$1" in
    --ui-lib) UI_LIB_MODE="$2"; shift 2 ;;
    --no-dedupe) DEDUPE=0; shift ;;
    *) echo "unknown argument: $1" >&2; exit 1 ;;
  esac
done

COMPAT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CORE_PACKAGES="$(cd "$COMPAT_DIR/../packages" && pwd)"
NAME="mui-${MUI_VERSION}"
[[ "$UI_LIB_MODE" == pack ]] && NAME="${NAME}-pack"
[[ "$DEDUPE" == 0 ]] && NAME="${NAME}-nodedupe"
OUT="$COMPAT_DIR/.variants/$(basename "$PLUGIN_DIR")/$NAME"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
cp -R "$PLUGIN_DIR/src" "$PLUGIN_DIR/tsconfig.json" "$PLUGIN_DIR/vite.config.ts" "$PLUGIN_DIR/package.json" "$WORK/"
cd "$WORK"

if [[ "$UI_LIB_MODE" == pack ]]; then
  (cd "$CORE_PACKAGES/plugin-ui-lib" && npm pack --silent --pack-destination "$WORK" >/dev/null)
  UI_LIB_SPEC="file:$(ls "$WORK"/openeverest-ui-lib-*.tgz)"
else
  UI_LIB_SPEC="file:$CORE_PACKAGES/plugin-ui-lib"
fi
npm pkg set "dependencies.@openeverest/ui-lib=$UI_LIB_SPEC"
npm pkg set "devDependencies.@openeverest/plugin-sdk=file:$CORE_PACKAGES/plugin-sdk"
npm pkg set "dependencies.@mui/material=$MUI_VERSION"

# MUI 5's @mui/system root is CommonJS without an exports map, so deep imports
# (e.g. @mui/system/useThemeWithoutDefault) would bundle a require("react") stub;
# a MUI 5-era plugin author has to point them at the esm/ build.
MUI5_ESM_ALIAS=0
[[ "$MUI_VERSION" == 5.* ]] && MUI5_ESM_ALIAS=1

cat >vite.variant.config.ts <<EOF
import { defineConfig, mergeConfig } from 'vite';
import base from './vite.config.ts';

export default defineConfig((env) => {
  const config = mergeConfig(typeof base === 'function' ? base(env) : base, {
    resolve: {
      alias: $([[ "$MUI5_ESM_ALIAS" == 1 ]] && echo "[{ find: /^@mui\/system\/(?!esm\/)(.+)$/, replacement: '@mui/system/esm/\$1' }]" || echo "[]"),
    },
  });
  if ($([[ "$DEDUPE" == 0 ]] && echo true || echo false)) config.resolve.dedupe = [];
  return config;
});
EOF

npm install --silent --no-audit --no-fund --legacy-peer-deps

TYPECHECK=pass
npx tsc --noEmit >"$WORK/tsc.log" 2>&1 || TYPECHECK=fail

rm -rf "$OUT"
npx vite build --config vite.variant.config.ts --sourcemap --outDir "$OUT" --emptyOutDir --logLevel warn

# Every @mui/material copy reachable from node_modules; more than one means the bundle may carry two MUIs.
INSTALLED_MUI="$(node -e '
const fs = require("fs"), path = require("path");
const found = new Set(), seen = new Set();
const walk = (dir, depth) => {
  let real;
  try { real = fs.realpathSync(dir); } catch { return; }
  if (seen.has(real) || depth > 3) return;
  seen.add(real);
  const pkg = path.join(real, "node_modules/@mui/material/package.json");
  if (fs.existsSync(pkg)) found.add(fs.realpathSync(path.dirname(pkg)) + "@" + require(pkg).version);
  const nm = path.join(real, "node_modules");
  if (!fs.existsSync(nm)) return;
  for (const name of fs.readdirSync(nm)) {
    if (name.startsWith(".")) continue;
    const entries = name.startsWith("@") ? fs.readdirSync(path.join(nm, name)).map((n) => path.join(name, n)) : [name];
    for (const e of entries) walk(path.join(nm, e), depth + 1);
  }
};
walk(".", 0);
console.log([...found].sort().join(","));
')"

node -e '
const [out, requested, uiLib, dedupe, typecheck, installed, mui5Alias] = process.argv.slice(1);
require("fs").writeFileSync(out + "/variant.json", JSON.stringify({
  requestedMui: requested,
  uiLib,
  dedupe: dedupe === "1",
  mui5EsmAlias: mui5Alias === "1",
  typecheck,
  installedMui: installed ? installed.split(",") : [],
  builtAt: new Date().toISOString(),
}, null, 2) + "\n");
' "$OUT" "$MUI_VERSION" "$UI_LIB_MODE" "$DEDUPE" "$TYPECHECK" "$INSTALLED_MUI" "$MUI5_ESM_ALIAS"

if [[ "$TYPECHECK" == fail ]]; then cp "$WORK/tsc.log" "$OUT/tsc.log"; fi
echo "built $OUT (typecheck: $TYPECHECK, installed MUI: $INSTALLED_MUI)"
