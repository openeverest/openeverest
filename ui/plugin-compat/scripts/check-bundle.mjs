#!/usr/bin/env node
// Static checks on built plugin bundles: only host-provided bare imports, no
// CommonJS require stubs, and exactly one bundled copy of MUI/Emotion.
// Usage: check-bundle.mjs <variant-dir>... (each containing main.js + main.js.map)
import fs from 'node:fs';
import path from 'node:path';
import zlib from 'node:zlib';

const HOST_PROVIDED = new Set(['react', 'react-dom', 'react/jsx-runtime']);
const SINGLETON_PACKAGES = ['@mui/material', '@mui/system', '@emotion/react', '@emotion/cache'];

const stripPrivate = (p) => p.replace(/^\/private\//, '/');

function bareImports(code) {
  const specs = new Set();
  // Anchored to line starts: MUI error messages embed `import ... from '...'` examples in strings.
  const patterns = [
    /^import\s[^;]*?\bfrom\s*["']([^"']+)["']/gm,
    /^import\s*["']([^"']+)["']/gm,
    /^export\s[^;]*?\bfrom\s*["']([^"']+)["']/gm,
    /\bimport\(\s*["']([^"']+)["']\s*\)/g,
  ];
  for (const re of patterns) {
    for (const m of code.matchAll(re)) {
      if (!m[1].startsWith('.') && !m[1].startsWith('/') && !m[1].includes('://')) specs.add(m[1]);
    }
  }
  return [...specs].sort();
}

function packageRoots(map, outDir) {
  const roots = Object.fromEntries(SINGLETON_PACKAGES.map((p) => [p, new Set()]));
  for (const source of map.sources) {
    const abs = stripPrivate(path.resolve(outDir, source));
    for (const pkg of SINGLETON_PACKAGES) {
      const marker = `/node_modules/${pkg}/`;
      const idx = abs.lastIndexOf(marker);
      if (idx !== -1) roots[pkg].add(abs.slice(0, idx + marker.length - 1));
    }
  }
  return roots;
}

function versionOf(root, installed) {
  const hit = installed.find((entry) => stripPrivate(entry).startsWith(`${root}@`));
  return hit ? hit.slice(hit.lastIndexOf('@') + 1) : (root.match(/@mui\+material@([^_/]+)/)?.[1] ?? '?');
}

function check(dir) {
  const code = fs.readFileSync(path.join(dir, 'main.js'), 'utf8');
  const map = JSON.parse(fs.readFileSync(path.join(dir, 'main.js.map'), 'utf8'));
  const variantFile = path.join(dir, 'variant.json');
  const variant = fs.existsSync(variantFile) ? JSON.parse(fs.readFileSync(variantFile, 'utf8')) : {};
  const installed = variant.installedMui ?? [];

  const imports = bareImports(code);
  const roots = packageRoots(map, dir);
  const muiCopies = [...roots['@mui/material']].map((r) => versionOf(r, installed));
  const failures = [];

  const disallowed = imports.filter((s) => !HOST_PROVIDED.has(s));
  if (disallowed.length) failures.push(`bare imports not provided by host: ${disallowed.join(', ')}`);
  if (/__require\(\s*["']react/.test(code)) failures.push('CommonJS require("react") stub bundled');
  for (const [pkg, set] of Object.entries(roots)) {
    if (set.size > 1) failures.push(`${set.size} copies of ${pkg} bundled`);
  }
  if (variant.requestedMui && muiCopies.length && !muiCopies.includes(variant.requestedMui)) {
    failures.push(`requested MUI ${variant.requestedMui} but bundled ${muiCopies.join(', ')}`);
  }

  return {
    variant: path.relative(process.cwd(), dir),
    requestedMui: variant.requestedMui ?? '?',
    bundledMui: muiCopies,
    imports,
    sizeKb: Math.round(code.length / 1024),
    gzipKb: Math.round(zlib.gzipSync(code).length / 1024),
    failures,
  };
}

const dirs = process.argv.slice(2);
if (!dirs.length) {
  console.error('usage: check-bundle.mjs <variant-dir>...');
  process.exit(2);
}
const results = dirs.map(check);
for (const r of results) {
  const status = r.failures.length ? 'FAIL' : 'ok  ';
  console.log(
    `${status} ${r.variant}  mui=${r.bundledMui.join('+') || 'none'}  ${r.sizeKb}kB/${r.gzipKb}kB gz  imports=[${r.imports.join(', ')}]`,
  );
  for (const f of r.failures) console.log(`       - ${f}`);
}
if (process.env.CHECK_BUNDLE_JSON) fs.writeFileSync(process.env.CHECK_BUNDLE_JSON, JSON.stringify(results, null, 2));
process.exit(results.some((r) => r.failures.length) ? 1 : 0);
