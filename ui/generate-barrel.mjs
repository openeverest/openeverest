#!/usr/bin/env node
// Scans api/ui/*.types.ts and writes api/ui/index.ts, exporting each file
// under its own namespace to avoid symbol collisions across OpenAPI specs.
// Called automatically by generate-openapi-types / generate-openapi-type targets.

import { readdirSync, writeFileSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const apiUiDir = join(dirname(fileURLToPath(import.meta.url)), '..', 'api', 'ui');

const files = readdirSync(apiUiDir)
  .filter((f) => f.endsWith('.types.ts'))
  .sort();

if (files.length === 0) {
  console.error('No *.types.ts files found in api/ui/. Run make generate-openapi-types first.');
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
  '// Re-run `make generate-openapi-types` in ui/ to update.',
  '//',
  '// Each file is exported under its own namespace to avoid collisions:',
  ...files.map((f) => `//   ${toNamespace(f)} → ./${f.replace('.ts', '')}`),
  '',
  ...files.map((f) => `export * as ${toNamespace(f)} from './${f.replace('.ts', '')}';`),
  '',
];

writeFileSync(join(apiUiDir, 'index.ts'), lines.join('\n'));
console.log(`✔ api/ui/index.ts updated (${files.length} file(s)): ${files.map(toNamespace).join(', ')}`);
