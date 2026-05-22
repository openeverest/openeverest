# ToggleField

## Table of Contents

- [Properties](#properties)
- [Behavior](#behavior)
- [Examples](#examples)
  - [Basic toggle](#basic-toggle)
  - [With helper text](#with-helper-text)
  - [Mode overrides](#mode-overrides)

A boolean on/off switch for enabling features (backups, PITR, monitoring, and similar flags).

## Properties

- `uiType`: `"toggle"` (**Required**)
- `path` OR `id`: Data path or unique identifier (**Required**)
- `fieldParams`: Configuration object:
  - `label`: Display label for the field (**Recommended**)
  - `labelCaption`: Secondary text shown under the label (optional)
  - `helperText`: Static hint; mapped to `labelCaption` when `labelCaption` is not set
  - `defaultValue`: Default boolean value (default: `false` when omitted)
  - `disabled`: Whether the switch is disabled (default: `false`)
  - `autoFocus`: Automatically focus this field on render
  - `tooltip`: Tooltip on hover (see field wrappers)
  - `switchFieldProps`: Additional MUI `Switch` props
  - `formControlLabelProps`: Additional MUI `FormControlLabel` props
- `validation` (optional):
  - `celExpressions`: Cross-field CEL validation (see [CEL Expression Validation](../validation.md#cel-expression-validation))
  - `regex`: Not supported for toggle fields
  - `required`: **Not supported** — toggle values are always optional booleans (`true` / `false` / default `false`)

## Behavior

- **Default value:** When `fieldParams.defaultValue` is omitted, the form initializes the field to `false`.
- **Helper text:** `fieldParams.helperText` is shown as `labelCaption` under the main label. Explicit `labelCaption` takes precedence over `helperText`.
- **Validation errors:** Shown in a red `FormHelperText` below the switch via `SwitchInput` `error` and `helperText` props.
- **Postprocess:** `false` is preserved in the API payload (not treated as an empty value).
- **Layout:** Same horizontal spacing as text fields (`minWidth: 450px`).

## Examples

### Basic toggle

```yaml
backupsEnabled:
  uiType: toggle
  path: spec.backup.enabled
  fieldParams:
    label: Enable backups
    defaultValue: false
```

### With helper text

```yaml
monitoringEnabled:
  uiType: toggle
  path: spec.monitoring.enabled
  fieldParams:
    label: Enable monitoring
    helperText: Collect metrics for this database cluster
```

### Mode overrides

```yaml
backupsEnabled:
  uiType: toggle
  path: spec.backup.enabled
  fieldParams:
    label: Enable backups
    modes:
      edit:
        disabled: true
```

To hide the field in a specific form mode:

```yaml
importOnlyFlag:
  uiType: toggle
  path: spec.import.flag
  modes:
    import:
      uiType: hidden
  fieldParams:
    label: Import flag
```
