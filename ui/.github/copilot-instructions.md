---
applyTo: "ui/**"
---

# OpenEverest UI — Agent Instructions

## Tech Stack

- **React 18** with TypeScript (strict mode)
- **Vite** for bundling + Vitest for testing
- **MUI v5** — styling via `sx` prop only; no CSS modules, no styled-components
- **react-hook-form** + **zod** for form state & validation
- **TanStack Query** (React Query) for server state
- **pnpm** workspaces (monorepo: `apps/everest`, `packages/*`)

## Project Layout

```
ui/
  apps/everest/src/
    api/                  # API client functions (one file per resource)
    components/           # Reusable components (kebab-case folders)
    consts.ts             # App-wide constants
    hooks/
      api/                # TanStack Query hooks (one folder per resource)
      utils/              # Reusable utility hooks
      rbac/               # Authorization hooks
    pages/                # Route-level page components
    shared-types/         # Shared TypeScript types
    types-declarations/   # Type declarations for external libraries, overrides, d.ts files
    utils/                # Pure utility functions
  packages/
    ui-lib/               # Shared UI library (@percona/ui-lib)
    eslint-config-react/  # Shared ESLint config
    prettier-config/      # Shared Prettier config
```

## Naming Conventions

| What            | Convention        | Example                           |
| --------------- | ----------------- | --------------------------------- |
| Folders         | `kebab-case`      | `time-selection/`                 |
| Component files | `kebab-case.tsx`  | `time-selection.tsx`              |
| Types files     | `.types.ts`       | `time-selection.types.ts`         |
| Constants files | `.constants.ts`   | `time-selection.constants.ts`     |
| Messages files  | `.messages.ts`    | `time-selection.messages.ts`      |
| Utils files     | `.utils.ts`       | `time-selection.utils.ts`         |
| Schema files    | `-schema.ts`      | `schedule-form-schema.ts`         |
| Context files   | `.context.ts`     | `schedule-form-dialog.context.ts` |
| Test files      | `.test.tsx`       | `time-selection.test.tsx`         |
| Hook files      | `useCamelCase.ts` | `useKubernetesClusterInfo.ts`     |
| Index files     | `index.ts`        | barrel re-export only             |

## Component Structure (Required)

Every component folder **must** follow this structure. Files are created only when needed — don't create empty files.

```
component-name/
  index.ts                       # Barrel: export { ComponentName } from './component-name'
  component-name.tsx             # Component implementation
  component-name.types.ts        # Props, internal types
  component-name.constants.ts    # Local constants, enums
  component-name.messages.ts     # All user-facing strings
  component-name.utils.ts        # Pure helper functions
  component-name-schema.ts       # Zod schemas (if form component)
  component-name.test.tsx        # Unit tests
  sub-component/                 # Nested components follow same pattern
  component-name-context/        # Context (if needed)
    component-name.context.ts
    component-name-context.types.ts
```

## Code Style Rules

### Exports

- Use **named exports** — not default exports (`export const Component`, not `export default`).
  Exception: page-level components and legacy code may use default exports.
- Each component folder has an `index.ts` barrel that re-exports the public API.

### Imports

- Use **bare imports** from `src/` root (enabled by `baseUrl: "./src"` in tsconfig):
  `import { FormDialog } from 'components/form-dialog'` — not `'../../components/form-dialog'`.
- Group imports: React/libraries → project imports → relative imports. Separate groups with blank lines.
- **No re-exports**: Do not re-export types/values from other modules to create "convenience" import paths.
  Each module should import directly from the canonical source. Re-exports blur module boundaries
  and make it unclear where a dependency actually comes from. If an import path feels inconvenient,
  move the declaration to a better location instead of masking the problem with a re-export.
  Exception: `index.ts` barrels for component public APIs are allowed.

### TypeScript

- `strict: true` — no `any` unless absolutely necessary.
- `noUnusedLocals`, `noUnusedParameters` are enforced.
- Don't use `@ts-nocheck` or `@ts-ignore` in new code.
- Prefer `interface` for component props; use `type` for unions, intersections, and utility types.

### Strings & Messages

- All user-facing strings go into a `.messages.ts` file as a `Messages` object:
  ```ts
  export const Messages = {
    title: "My Component",
    error: (name: string) => `Failed to load ${name}`,
  };
  ```
- Never hardcode user-facing strings directly in JSX.

### Constants

- Component-local constants go into `.constants.ts`.
- App-wide constants go into `consts.ts`.
- Use `UPPER_SNAKE_CASE` for primitive constants, `PascalCase` for enums.

### Styling

- Use MUI `sx` prop for all styling — no CSS-in-JS, no CSS modules.
- Use MUI theme values, not hardcoded colors/spacing: `(theme) => theme.spacing(2)`.
- Breakpoints via `useActiveBreakpoint()` hook, not raw MUI breakpoints.

### Forms

- Use `react-hook-form` with `useFormContext()` — never `useState` for form fields.
- Validation via `zod` schemas — place in a `-schema.ts` file.
- Use components from `@percona/ui-lib` (`TextInput`, `SelectInput`, `ToggleButtonGroupInput`, etc.).

### API & Data Fetching

- API functions go in `api/` — one file per resource (`instanceApi.ts`, `kubernetesClusterApi.ts`).
- TanStack Query hooks go in `hooks/api/<resource>/` — one folder per API resource.
- Query key convention: descriptive string arrays (`['cluster-info']`, `['db-instances', namespace]`).
- Mutations use `useMutation` or the project's conflict-retry wrapper when handling 409s.

### Testing

- Test files live next to the code they test: `component-name.test.tsx`.
- Use `vi.mock()` for module mocking, `vi.fn()` for function stubs.
- Use `@testing-library/react` (`render`, `screen`, `waitFor`, `fireEvent`).
- Wrap rendered components in `<TestWrapper>` for providers.
- Don't test implementation details — test behavior visible to users.
- Keep common API mocks co-located with their hook folders using `__mocks__/` convention
  (e.g., `hooks/api/backup-storages/__mocks__/`). Reuse these shared mocks across tests.

### Context Pattern

- Create a subfolder `component-name-context/` with:
  - `component-name.context.ts` — `createContext` with a typed default value.
  - `component-name-context.types.ts` — context type definition.
- Export a `useComponentNameContext()` hook for consumers.

## UI Generator Rules

The UI generator (`components/ui-generator/`) is a schema-driven form renderer. Special rules apply:

### Architecture Layers

1. **Preprocessing** (`utils/preprocess/`) — schema normalization, mode overrides. Runs once.
2. **Schema Building** (`utils/schema-builder/`) — converts UI schema to Zod. Runs once.
3. **Rendering** (`ui-component/`, `ui-group/`) — renders components from processed schema.
4. **Postprocessing** (`utils/postprocess/`) — transforms form data back to API shape.

### Adding Runtime Field Behavior

When a field needs runtime behavior (e.g., disabled based on API data, dynamic validation):

1. **Define a runtime field override** via the `fieldOverrides` map on `UiGeneratorContext` —
   keyed by field path, providing `{ disabled?, tooltipTitle?, helperText? }`.
2. **Compute overrides** in the consumer (e.g., `SectionEditModal`) where the relevant data
   is available, and pass them through `<UIGenerator fieldOverrides={...} />`.
3. **Never** add API calls (`useQuery`, `useMutation`) directly inside `ui-component.tsx` or
   other rendering internals. Data should flow in through context/props.
4. **Never** hardcode field path checks inside rendering code. Use the declarative override map.

### Schema Paths

- `path` defines the API field binding: `spec.components.engine.storage.size`.
- Multi-path arrays: first = source, rest = write targets.
- `id`-only fields have no API binding — used for UI-only state.

### Validation & Modes

- Validation is mode-aware: base props + `modes.[edit|new|restore].{...}`.
- CEL expressions are used for cross-field validation.
- `FormMode` (`new`, `edit`, `restore`, `import`) drives which overrides apply.

## Don'ts

- **Don't** put business logic in rendering components — extract it to hooks or utils.
- **Don't** use `console.log` — ESLint `no-console` is set to error.
- **Don't** add dependencies without checking if the workspace already has an equivalent.
- **Don't** create files that aren't needed — no empty placeholder files.
- **Don't** modify generated files (`*.gen.*`, or anything in `types/` — these are OpenAPI-generated).
- **Don't** mix concerns — API calls don't belong in UI components, styling doesn't belong in hooks.
