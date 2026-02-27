#!/usr/bin/env node
// Scans generated/*.types.ts and writes generated/index.ts, exporting each
// file under its own namespace to avoid symbol collisions across OpenAPI specs.
// Run via: make generate-barrel  (called automatically by generate-all / generate)

import { readdirSync, writeFileSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const generatedDir = join(dirname(fileURLToPath(import.meta.url)), 'generated');

const files = readdirSync(generatedDir)
  .filter((f) => f.endsWith('.types.ts'))
  .sort();

if (files.length === 0) {
  console.error('No *.types.ts files found in generated/. Run make generate-all first.');
  process.exit(1);
}

const toNamespace = (filename) =>
  // crds.gen.types.ts -> CrdsGen  |  http-api.types.ts -> HttpApi
  filename
    .replace('.types.ts', '')
    .split(/[^a-zA-Z0-9]+/)
    .filter(Boolean)
    .map((p) => p.charAt(0).toUpperCase() + p.slice(1))
    .join('');

const lines = [
  '// AUTO-GENERATED — do not edit manually.',
  '// Re-run `make generate-all` or `make generate-barrel` to update.',
  '//',
  '// Each file is exported under its own namespace to avoid collisions:',
  ...files.map((f) => `//   ${toNamespace(f)} → ./${f.replace('.ts', '')}`),
  '',
  ...files.map((f) => `export * as ${toNamespace(f)} from './${f.replace('.ts', '')}';`),
  '',
];

writeFileSync(join(generatedDir, 'index.ts'), lines.join('\n'));
console.log(`✔ generated/index.ts updated (${files.length} file(s)): ${files.map(toNamespace).join(', ')}`);
