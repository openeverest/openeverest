# API Providers

Select fields whose options come from an Everest API at runtime.

## Table of Contents

- [Quick Start](#quick-start)
- [Schema Reference](#schema-reference)
- [Validation, Modes & fieldParams](#validation-modes--fieldparams)
- [Available Providers](#available-providers)
- [Field Behavior](#field-behavior)
- [Adding a New Provider](#adding-a-new-provider-in-httpsgithubcomopeneverestopeneverest)

## Quick Start

```yaml
storageClass:
  uiType: select
  dataSource:
    provider: storageClasses
  path: spec.engine.storage.class
  fieldParams:
    label: Storage class
```

That's it. Options load automatically, the first option is selected by default (for providers with auto-select, see [Field Behavior](#field-behavior)), and loading/error states are handled.

## Schema Reference

```yaml
fieldKey:
  uiType: select
  dataSource:
    provider: <providerKey> # required — see Available Providers
  path: <dot.separated.path> # required if the value should be submitted
  fieldParams:
    label: string
    helperText: string # shown below; defaults to "No options available" when empty
    disabled: boolean # always disabled, regardless of loading state
    defaultValue: string # overrides auto-selection of first option
    displayEmpty: boolean # adds empty "None" option for optional fields
  validation: # same as regular select — required, regex, celExpressions
    required: true
  modes: # per-mode overrides — same as any field
    edit:
      helperText: Changing this will restart the agent
```

## Validation, Modes & fieldParams

A `dataSource` field is still a regular `select` — the only difference is that options come from an API instead of being declared inline. Everything else works the same:

- **Validation** — `required`, `regex`, and `celExpressions` all apply as documented in [Validation](validation.md). CEL cross-field rules (e.g. "storageClass is required when replicas > 1") work without changes — see [CEL Expressions](validation.md#cel-expressions).
- **Modes** — per-mode overrides for `fieldParams`, `validation`, and `uiType` follow the standard rules described in [Mode-Aware Overrides](Readme.md#mode-aware-overrides).
- **fieldParams** — supports the same keys as [Select Field](components/select-field.md): `label`, `helperText`, `disabled`, `defaultValue`, `displayEmpty`, etc. The only key you should **not** set manually is `options` — it is always provided by the API.

## Available Providers

| Key                 | Returns                                     | Scoped to         | Empty State Fallback           | Auto-select |
| ------------------- | ------------------------------------------- | ----------------- | ------------------------------ | ----------- |
| `monitoringConfigs` | MonitoringConfig resource names             | cluster/namespace | ActionableAlert + create modal | yes         |
| `storageClasses`    | StorageClass names available on the cluster | cluster           | None (not user-creatable)      | yes         |
| `backupStorages`    | BackupStorage (storage location) names      | cluster/namespace | ActionableAlert + create modal | no          |

## Field Behavior

| State                               | Field    | Value                                                   | Fallback                                                     |
| ----------------------------------- | -------- | ------------------------------------------------------- | ------------------------------------------------------------ |
| Loading                             | disabled | unchanged; helper text "Loading..."                     | not shown                                                    |
| Fetch failed                        | disabled | unchanged; helper text "Failed to load options"         | not shown                                                    |
| No options                          | hidden   | —                                                       | shown instead of select if provider has `emptyStateFallback` |
| Options loaded, `autoSelect: true`  | enabled  | empty or unknown value replaced with the first option   | not shown                                                    |
| Options loaded, `autoSelect: false` | enabled  | kept as is — empty stays empty, unknown values are kept | not shown                                                    |

Providers with `autoSelect: true` fill an empty field with the first option and replace a value that is no longer in the list. This happens on form mount (via `DataSourcePrefetcher`, even for steps the user hasn't opened) and whenever the field remounts. Providers for optional references, where an empty value is a meaningful choice (e.g. a storage location that falls back to the provider's bundled store), register `autoSelect: false`. Use `displayEmpty: true` on such fields so the user can pick "None".

When a provider registers an `emptyStateFallback`, `DataSourceField` renders the fallback component **instead of** the select field whenever options are empty. The select is not shown at all — the fallback completely replaces it. This is used for resources that can be user-created (e.g. monitoring endpoints). Providers for non-creatable resources (e.g. StorageClasses) register `emptyStateFallback: null`, in which case the select is rendered normally with a "No options available" helper text.

## Adding a New Provider in https://github.com/openeverest/openeverest

1. Create a `useXxxOptions` hook in `hooks/api/<resource>/` that accepts `ProviderParams` and returns `ProviderOptions`.
2. If the resource can be user-created, create a fallback component in `api-providers/fallbacks/` (see `monitoring-empty-fallback.tsx` for the pattern).
3. Register it in `api-providers/providers.ts`:

   ```ts
   import { useXxxOptions } from "hooks/api/<resource>/useXxxOptions";
   import { XxxEmptyFallback } from "./fallbacks";

   providerRegistry.register("xxx", {
     description: "Short description.",
     useOptions: useXxxOptions,
     // Use { component: XxxEmptyFallback } for user-creatable resources,
     // or null for resources that cannot be created by the user.
     emptyStateFallback: { component: XxxEmptyFallback },
     // true: preselect the first option and replace unknown values.
     // false: for optional references where an empty value is a valid choice.
     autoSelect: true,
   });
   ```

4. Use `provider: xxx` in the schema.

### ProviderParams / ProviderOptions types

```ts
interface ProviderParams {
  namespace: string;
  cluster: string;
}

interface ProviderOptions {
  options: Array<{ label: string; value: string }>;
  isLoading: boolean;
  error: unknown;
  isEmpty: boolean;
  rawData?: unknown;
}
```

## Future Ideas

<!-- TODO: These features are not yet implemented and are listed as potential improvements for openeverest. -->

- **`dataSource.config`** — per-field configuration for providers (e.g. custom `refetchInterval`, filtering, `enabled` via CEL expressions depending on other form fields). Currently not implemented because all built-in providers use sensible internal defaults and there is no real use-case for two different queries to the same resource within a single form. Can be added back to `DataSource`, `ProviderParams`, and the provider hooks when a concrete need arises.
