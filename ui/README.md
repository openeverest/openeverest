# OpenEverest Frontend

OpenEverest UI is a PNPM monorepo living in `ui/`.

## UI architecture

Main workspaces:

- `apps/everest`: the OpenEverest frontend application (Vite + React).
- `packages/ui-lib`: reusable UI components.
- `packages/design`: design tokens/themes and shared styling setup.
- `packages/utils`: shared utility helpers.
- `packages/types`: shared TypeScript types.
- `packages/eslint-config-react`, `packages/prettier-config`, `packages/tsconfig`: shared tooling configs.

How code flows:

- `apps/everest` consumes shared packages from `packages/*` via workspace deps.
- Turborepo orchestrates tasks across workspaces (`test`, `lint`, `format`, `build`).
- App-level tests live in `apps/everest/src/**` (unit and browser-mode tests), while end-to-end tests live in `apps/everest/.e2e`.

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
