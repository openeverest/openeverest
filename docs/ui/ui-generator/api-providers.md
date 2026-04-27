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

That's it. Options load automatically, first option is selected by default, and loading/error states are handled.

## Schema Reference

```yaml
fieldKey:
  uiType: select
  dataSource:
    provider: <providerKey> # required — see Available Providers
    config: # optional
      refetchInterval: 5000
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

| Key                 | Returns                                     | Scoped to         |
| ------------------- | ------------------------------------------- | ----------------- |
| `monitoringConfigs` | MonitoringConfig resource names             | cluster/namespace |
| `storageClasses`    | StorageClass names available on the cluster | cluster           |

## Field Behavior

| State          | Field    | Text                       |
| -------------- | -------- | -------------------------- |
| Loading        | disabled | Loading...                 |
| Fetch failed   | disabled | Failed to load options     |
| No options     | enabled  | No options available       |
| Options loaded | enabled  | first option auto-selected |

## Adding a New Provider in https://github.com/openeverest/openeverest

1. Create a `useXxxOptions` hook in `hooks/api/<resource>/` that accepts `ProviderParams` and returns `ProviderOptions`.
2. Register it in `api-providers/providers.ts`:

   ```ts
   import { useXxxOptions } from "hooks/api/<resource>/useXxxOptions";

   providerRegistry.register("xxx", {
     description: "Short description.",
     useOptions: useXxxOptions,
   });
   ```

3. Use `provider: xxx` in the schema schema.

### ProviderParams / ProviderOptions types

```ts
interface ProviderParams {
  namespace: string;
  cluster: string;
  config?: { refetchInterval?: number };
}

interface ProviderOptions {
  options: Array<{ label: string; value: string }>;
  isLoading: boolean;
  error: unknown;
  isEmpty: boolean;
  rawData?: unknown;
}
```
