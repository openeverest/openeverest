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
  - `helperText`: Secondary text shown under the label (optional). Validation errors render separately below the switch.
  - `defaultValue`: Default boolean value (default: `false` when omitted)
  - `disabled`: Whether the switch is disabled (default: `false`)
  - `autoFocus`: Automatically focus this field on render
- `validation` (optional):
  - `celExpressions`: Cross-field CEL validation (see [CEL Expression Validation](../validation.md#cel-expression-validation))
  - `regex`: Not supported for toggle fields
  - `required`: **Not supported** — toggle values are always optional booleans (`true` / `false` / default `false`)

## Behavior

- **Default value:** When `fieldParams.defaultValue` is omitted, the form initializes the field to `false`.
- **Helper text:** `fieldParams.helperText` renders as a caption under the main label — `helperText` is the same universal param used by every other field, so provider authors don't need a toggle-specific key. Validation errors (e.g. from CEL expressions) appear separately in a red `FormHelperText` below the switch, so a toggle can show its description and an error at the same time.
- **Validation errors:** Field-level validation errors (e.g. from CEL expressions) are shown in a red `FormHelperText` below the switch.
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
