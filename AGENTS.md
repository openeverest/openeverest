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

### Reuse & design system

- **Look before you build.** Before writing any UI, search `packages/ui-lib` and `apps/everest/src/components` for something to reuse, adapt or extend. Building new is the last resort — the moment you feel you're re-creating an existing card/button/surface, stop and reuse it.
- **Promote generic primitives to `ui-lib` immediately.** Reusable, domain-free UI (buttons, cards, surfaces, pickers) belongs in `@percona/ui-lib` from the start — don't park it in `apps/everest` with a "we'll extract it later" note (that's how the app becomes a dump). Extract only when the API is generic and no domain type leaks; keep domain-specific wrappers in the app.
- **New shared component ⇒ add a Storybook story** next to it and a test covering real behavior.

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
- **Theme-first styling.** Design tokens and any recurring look (card/heading/surface border, background, radius, selection ring) live in the MUI theme as component `variants` (see `MuiCard` `grey`/`selectable`), inherited everywhere — not re-written inline per call site. `sx` is only for one-off layout/composition (`display`, `gap`, padding, grid), and reads theme values (`theme.spacing`, `theme.palette`). Breakpoints via `useActiveBreakpoint()`.
- **Don't duplicate styles, don't re-declare defaults.** If the same style block appears in more than one component, hoist it (theme variant, ui-lib primitive, or a shared `.constants.ts`) — no copy-paste. Never re-state a value the theme already provides (e.g. `borderRadius: theme.shape.borderRadius` — `Card`/`Paper` already apply it).
- Forms: `react-hook-form` + `useFormContext()` + zod `-schema.ts`; inputs from `@percona/ui-lib`.
- Local constants → `.constants.ts`; app-wide → `consts.ts` (`UPPER_SNAKE_CASE`, `PascalCase` enums).

### Data

- API functions in `api/` (one file per resource); query hooks in `hooks/api/<resource>/`.
- Query keys = descriptive arrays (`['db-instances', namespace]`).
- **Coordinate writes with reads:** mutations invalidate (or optimistically update) affected queries in `onSuccess` — never rely on "navigate + stale refetch".

### Testing

- `*.test.tsx` next to the code; `@testing-library/react`; `vi.mock` / `vi.fn`.
- Test behavior, not implementation. Co-locate shared API mocks in `__mocks__/`.
- **No trivial or redundant tests.** Don't add tests that merely assert a component renders a passed-in prop/label with no logic, or that duplicate coverage already provided elsewhere — they add CI time without catching real regressions. Cover meaningful behavior, edge cases, and branching logic.
- **Mocks must return stable references** — define the object once in the `vi.mock` closure; a fresh literal per call makes `useMemo`/effect deps loop and hangs tests.

### Context

- `component-name-context/` with `.context.ts` + `-context.types.ts`; expose `useComponentNameContext()`.

### UI Generator (`components/ui-generator/`)

Schema-driven form renderer. Layers: preprocess → schema-build (zod) → render → postprocess.
Runtime field behavior goes through the `fieldOverrides` map on `UiGeneratorContext` (keyed by path), computed in the consumer — never hardcode path checks or add queries inside render internals.
**Keep docs in sync (required):** any change to the UI Generator architecture or its schema props (adding/renaming/removing a prop, `uiType`, `groupType`, validation rule, mode behavior, etc.) must update the architecture docs in `docs/ui/architecture/ui-generator/` and the user-facing docs in `docs/ui/ui-generator/` in the same PR, and bump the `Last updated` date in the touched docs.

### Don'ts

- No business logic in render components — extract to hooks/utils.
- No `console.log` (ESLint error).
- Don't add deps without checking for an existing equivalent.
- Don't mix concerns (API calls in components, styling in hooks).
