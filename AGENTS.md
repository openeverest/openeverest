# OpenEverest — Agent Instructions

Two tiers: **Repo-wide** (everywhere) and **Frontend** (`ui/**` only, React/TS).
Read the whole file; apply Frontend rules only when touching files under `ui/`.

## Repo-wide

- **Self-documenting code.** Rename for clarity instead of commenting. Comments explain _why_, not _what_.
- **No re-exports (hard rule).** Never re-export or alias a symbol from another module — import it from where it's declared. Only a folder's `index.ts` may re-export _its own_ public API.
  ```ts
  // BAD:  export type { X as Y } from '../other';
  // GOOD: import type { X } from '../other';
  ```
- **No dead files.** Don't create empty/placeholder files.
- **Don't edit generated files** (`*.gen.*`, OpenAPI `types/`).

## Frontend (`ui/**`)

### Stack

- React 18 + TS (strict)
- Vite
- Vitest
- MUI (`sx` only)
- react-hook-form + zod
- TanStack Query
- pnpm workspaces (`apps/everest`, `packages/*`).

### Files & naming

- Folders & files `kebab-case`
- hooks `useCamelCase.ts`
- Per-concern files: `.types.ts`, `.constants.ts`, `.messages.ts`, `.utils.ts`, `-schema.ts`, `.context.ts`, `.test.tsx`.
- `index.ts` = barrel for the folder's own public API only.

### Components

- **One component per file** Extract sub-components into their own `sub-component/` folder.
- **Named exports** (default only for page-level/legacy).
- **Bare imports** from `src/` root (`import { X } from 'components/...'`), not `../../`.
- Group imports: libs → project → relative.

### TypeScript

- `strict`; no `any`; no `@ts-ignore` / `@ts-nocheck`.
- **No `as` assertions** — narrow (guards, `typeof`) or annotate instead.
- `interface` for props; `type` for unions/utilities.

### UI conventions

- User-facing strings live in `.messages.ts` (a `Messages` object) — never hardcoded in JSX.
- Styling via MUI `sx` + theme values (`theme.spacing`, `theme.palette`); breakpoints via `useActiveBreakpoint()`.
- Forms: `react-hook-form` + `useFormContext()` + zod `-schema.ts`; inputs from `@percona/ui-lib`.
- Local constants → `.constants.ts`; app-wide → `consts.ts` (`UPPER_SNAKE_CASE`, `PascalCase` enums).

### Data

- API functions in `api/` (one file per resource); query hooks in `hooks/api/<resource>/`.
- Query keys = descriptive arrays (`['db-instances', namespace]`).
- **Coordinate writes with reads:** mutations invalidate (or optimistically update) affected queries in `onSuccess` — never rely on "navigate + stale refetch".

### Testing

- `*.test.tsx` next to the code; `@testing-library/react`; `vi.mock` / `vi.fn`.
- Test behavior, not implementation. Co-locate shared API mocks in `__mocks__/`.
- **Mocks must return stable references** — define the object once in the `vi.mock` closure; a fresh literal per call makes `useMemo`/effect deps loop and hangs tests.

### Context

- `component-name-context/` with `.context.ts` + `-context.types.ts`; expose `useComponentNameContext()`.

### UI Generator (`components/ui-generator/`)

Schema-driven form renderer. Layers: preprocess → schema-build (zod) → render → postprocess.
Runtime field behavior goes through the `fieldOverrides` map on `UiGeneratorContext` (keyed by path), computed in the consumer — never hardcode path checks or add queries inside render internals.

### Don'ts

- No business logic in render components — extract to hooks/utils.
- No `console.log` (ESLint error).
- Don't add deps without checking for an existing equivalent.
- Don't mix concerns (API calls in components, styling in hooks).
