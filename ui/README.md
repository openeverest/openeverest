# OpenEverest Frontend

OpenEverest UI is a PNPM monorepo living in `ui/`.

## Architecture docs

- [docs/ui/architecture/README.md](../docs/ui/architecture/README.md)

## Prerequisites

- Install PNPM: https://pnpm.io/installation

## Core commands (from `ui/`)

Rule of thumb:

- Use `make` for top-level workflows in the whole `ui/` monorepo.
- Use `pnpm` when working inside a specific workspace/package.

Install dependencies:

```bash
make init
```

Run Everest in dev mode:

```bash
make dev
```

Build all UI packages:

```bash
make build
```

Run all monorepo tests:

```bash
make test
```

Run all lint tasks:

```bash
make lint
```

Run all format tasks:

```bash
make format
```

Contributor preflight before PR (format + lint + Everest unit/browser tests + Everest build):

```bash
make preflight
```

## Everest app test commands (from `ui/apps/everest`)

Run all Everest tests (unit + browser):

```bash
pnpm test
```

Run only unit tests:

```bash
pnpm test:unit
```

Watch only unit tests:

```bash
pnpm test:unit:watch
```

Run only browser-mode tests (headless):

```bash
pnpm test:browser
```

Watch only browser-mode tests:

```bash
pnpm test:browser:watch
```

Watch all tests (unit + browser projects):

```bash
pnpm test:watch
```

Run a specific workspace command from `ui/`:

```bash
pnpm --filter <workspace> <command>
```

More on filtering: https://pnpm.io/filtering

## E2E tests

E2E setup and local execution details are documented in `apps/everest/.e2e/Readme.md`.

## Static Dependency Analysis

Detect cycles (database-form + ui-generator scope):

```bash
pnpm analyze:deps
```

Generate graph in DOT format:

```bash
pnpm analyze:deps:graph
```

Render DOT graph to SVG (requires [Graphviz](https://graphviz.org/)):

```bash
pnpm analyze:deps:svg
```

Generate machine-readable JSON report:

```bash
pnpm analyze:deps:json
```

Generated artifacts (`deps-graph.dot`, `deps-graph.svg`, `deps-report.json`) are git-ignored.
