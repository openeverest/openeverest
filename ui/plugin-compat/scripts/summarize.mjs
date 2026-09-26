#!/usr/bin/env node
// Prints a plugin-variant x check matrix per host from Playwright JSON reports.
// Usage: summarize.mjs results/*.json
import fs from 'node:fs';
import path from 'node:path';

const SYMBOL = { expected: 'pass', unexpected: 'FAIL', flaky: 'flaky', skipped: 'skip' };

function collect(suite, rows, prefix = []) {
  for (const child of suite.suites ?? []) collect(child, rows, child.title.endsWith('.ts') ? prefix : [...prefix, child.title]);
  for (const spec of suite.specs ?? []) {
    const status = spec.tests.map((t) => t.status).find((s) => s !== 'expected') ?? 'expected';
    const knownBug = spec.tests.some((t) => t.annotations?.some((a) => a.type === 'fail'));
    rows.push({ group: prefix.join(' / '), check: spec.title, status: knownBug && status === 'expected' ? 'xfail' : (SYMBOL[status] ?? status) });
  }
}

for (const file of process.argv.slice(2)) {
  const report = JSON.parse(fs.readFileSync(file, 'utf8'));
  const rows = [];
  for (const suite of report.suites) collect(suite, rows);

  const groups = [...new Set(rows.map((r) => r.group))];
  const checks = [...new Set(rows.map((r) => r.check))];
  const cell = (g, c) => rows.find((r) => r.group === g && r.check === c)?.status ?? '';
  const width = Math.max(...groups.map((g) => g.length));

  console.log(`\n## ${path.basename(file, '.json')}`);
  checks.forEach((c, i) => console.log(`  [${i + 1}] ${c}`));
  console.log(`${''.padEnd(width)}  ${checks.map((_, i) => `[${i + 1}]`.padEnd(6)).join('')}`);
  for (const g of groups) console.log(`${g.padEnd(width)}  ${checks.map((c) => cell(g, c).padEnd(6)).join('')}`);
}
