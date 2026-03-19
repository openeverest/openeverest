import fs from 'node:fs';
import path from 'node:path';
import { spawnSync } from 'node:child_process';

const cwd = process.cwd();
const dotPath = path.resolve(cwd, 'deps-graph.dot');
const svgPath = path.resolve(cwd, 'deps-graph.svg');

const run = (cmd, args, opts = {}) =>
  spawnSync(cmd, args, {
    cwd,
    stdio: 'pipe',
    encoding: 'utf8',
    ...opts,
  });

if (!fs.existsSync(dotPath)) {
  const graphResult = run('pnpm', ['analyze:deps:graph']);
  if (graphResult.status !== 0) {
    process.stderr.write(graphResult.stderr || graphResult.stdout);
    process.exit(graphResult.status ?? 1);
  }
}

// Use system Graphviz `dot` to render SVG
const dotResult = run('dot', ['-Tsvg', dotPath, '-o', svgPath]);
if (dotResult.status !== 0) {
  process.stderr.write(
    'Failed to render SVG. Is Graphviz installed? (apt install graphviz / brew install graphviz)\n'
  );
  if (dotResult.stderr) process.stderr.write(dotResult.stderr);
  process.exit(1);
}

if (!fs.existsSync(svgPath)) {
  process.stderr.write('SVG file was not created.\n');
  process.exit(1);
}

const platform = process.platform;
let openResult;

if (platform === 'darwin') {
  openResult = run('open', [svgPath]);
} else if (platform === 'win32') {
  openResult = run('cmd', ['/c', 'start', '""', svgPath], { shell: true });
} else {
  openResult = run('xdg-open', [svgPath]);
}

if (openResult.status !== 0) {
  process.stdout.write(`SVG rendered: ${svgPath}\n`);
  process.stdout.write(
    'Could not auto-open browser. Open the file manually.\n'
  );
  process.exit(0);
}

process.stdout.write(`SVG rendered and opened: ${svgPath}\n`);
