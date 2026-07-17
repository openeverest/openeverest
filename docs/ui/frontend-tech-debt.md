# Frontend Dependency Tech Debt

Last reviewed: 2026-07-16 (right after the MUI 6 → 7 upgrade).

Scope: the `ui/` monorepo (`apps/everest`, `packages/ui-lib`, `packages/design`,
`packages/plugin-sdk`, `packages/eslint-config-react`).

## Current baseline

| Package                                 | Current | Latest             | Gap                |
| --------------------------------------- | ------- | ------------------ | ------------------ |
| `@mui/material` / `@mui/icons-material` | 7.0     | 9.2 (no v8)        | 2 majors           |
| `@mui/x-date-pickers`                   | 7.6     | 9.9                | 2 majors           |
| `material-react-table`                  | 3.2.1   | 3.2.1 (no v4 yet)  | current            |
| `react` / `react-dom`                   | 18.2    | 19.x               | 1 major            |
| `react-router-dom`                      | 6.24    | 7.x                | 1 major            |
| `date-fns`                              | 3.6     | 4.x                | 1 major            |
| `zod`                                   | 3.22    | 4.x                | 1 major            |
| `vite`                                  | 6.4     | 7.x                | 1 major            |
| `eslint`                                | 8.45    | 9.x (**8 is EOL**) | 1 major + EOL      |
| `@typescript-eslint/*`                  | 6.0     | 8.x                | 2 majors           |
| `typescript`                            | 5.2     | 5.9+               | minor drift        |
| `storybook`                             | 8.6     | 9.x                | 1 major (dev-only) |
| `eslint-plugin-react-hooks`             | 4.6     | 5.x                | 1 major            |

## List 1 — Update soon (outdated / EOL / low-risk hygiene)

Not blocked by ecosystem maturity; mostly tooling and version drift.

1. **ESLint 8.45 → 9.x + `@typescript-eslint` 6 → 8** — top priority. ESLint 8 is
   **EOL** (no security patches) and `@typescript-eslint@6` lags modern TS. Requires
   flat-config migration; `api-tests/` and `cli-tests/` already use flat config as a
   template. Also touches `packages/eslint-config-react` peer versions.
2. **TypeScript 5.2 → 5.9+** — minor drift only, low risk. Do before typescript-eslint bump.
3. **`eslint-plugin-react-hooks` 4.6 → 5.x** — pairs with the ESLint 9 bump.
4. **`@mui/x-date-pickers` 7.6 → latest 7.x** — stay on v7 line, pick up fixes.
   Fully compatible with `@mui/material@7`.
5. **Storybook 8.6 → 9.x** — dev-only (not in prod bundle), but v8 is in maintenance.
   Convenient to bundle with the ESLint wave.

## List 2 — Next horizon (~6 months, planned majors)

Stable but require deliberate migration + visual/e2e regression validation. Do these
**sequentially, not batched**.

1. **MUI v7 → v9** (the planned next target; see the MUI version-strategy ADR).
   - `@mui/material` has **no v8** (skipped for cross-package version unification) →
     target 9.x directly.
   - Blocker: **`material-react-table`** peer says `>=6` but was designed for v6/v7 →
     must validate rendering on v9 with the visual suite before committing.
   - `@mui/x-date-pickers` → 9.x moves with it (peer `^7.3 || ^9`, drops v6).
2. **`react-router-dom` 6 → 7** — data-router API breaking changes; independent of MUI.
3. **`vite` 6 → 7** — build tooling; usually cheap, but revalidate all configs/plugins.
4. **`zod` 3 → 4** and **`date-fns` 3 → 4** — validated via form resolvers and the
   date-picker adapter; real breaking changes, but not urgent.

## List 3 — Do not touch yet (recheck in 6–12 months)

Available but premature or without a driver.

1. **React 18 → 19** — MUI 7/9 already support it, but needs a full audit of runtime
   deps (`material-react-table`, `notistack`, `oidc-react`, `@xyflow/react`,
   `react-hook-form`). Move only once they all officially support React 19.
2. **MUI CSS theme variables (`theme.vars` / `applyStyles`) + Pigment CSS (zero-runtime)** —
   strategic MUI direction. Codebase is fully on `sx` + emotion (200+ sites). Premature;
   recheck Pigment maturity in ~1 year.
3. **`material-react-table` v4** — does not exist yet. When released with a stable MUI 9
   peer it unblocks List 2. Recheck in ~6 months.
4. **`@vitest/browser-playwright` / browser mode** — already on 4.x; leave as-is,
   revisit as the API stabilizes.

## Related

- MUI version strategy ADR (memory / internal notes): target destination is MUI v7 now,
  v9 later once `material-react-table` on v9 is validated.
