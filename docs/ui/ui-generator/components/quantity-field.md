# Quantity Field

## Table of Contents

- [Properties](#properties)
- [How Values Flow](#how-values-flow)
- [Native Validation](#native-validation)
- [Validation Auto-Mapping](#validation-auto-mapping)
- [Examples](#examples)
  - [CPU (cores)](#cpu-cores)
  - [Memory / disk](#memory--disk)

A numeric input for Kubernetes resource quantities (CPU, memory, disk). The form
holds a **number in `unit`**; the API always receives a **quantity string**.

Prefer `quantity` over `number` + `badge` / `badgeToApi` for resource fields.
`badge` / `badgeToApi` are not supported on this type.

## Properties:

- `uiType`: `"quantity"` (required)
- `path` OR `id`: Data path or unique identifier (required)
- `fieldParams`: Configuration object with the following optional properties:
  - `label`: Display label for the field
  - `placeholder`: Placeholder text shown when field is empty
  - `defaultValue`: Default numeric value (in `unit`, or cores when `unit` is omitted)
  - `disabled`: Whether the field is disabled (default: `false`)
  - `autoFocus`: Automatically focus this field on render
  - `helperText`: Help text displayed below the field
  - `unit`: Kubernetes quantity suffix for form values and API writes
    (`m`, `k`, `M`, `G`, `T`, `P`, `E`, `Ki`, `Mi`, `Gi`, `Ti`, `Pi`, `Ei`).
    Omit for CPU in cores (writes `"0.5"`, not `"500m"`).
  - `unitLabel`: End-adornment / overview label; falls back to `unit` (e.g. `"cores"`)
  - `step`: Increment for arrow buttons (e.g. `0.1`, `1`). Omit to accept any
    fractional value (`step="any"` is applied for you — do not set `"any"` in the schema).
- `validation` (optional): Same numeric rules as [number](number-field.md). All
  bounds (`min` / `max` / `gt` / `lt`) and CEL expressions see the **number in
  `unit`**, not the raw API string:
  - `required`, `min`, `max`, `gt`, `lt`, `int`, `multipleOf`, `safe`, `regex`
  - `celExpressions` — compare against other fields or `original` in edit mode
    (e.g. `size >= original.size`). Both sides are numbers in the same unit.

## How Values Flow:

| Stage        | Behaviour                                                                 |
| ------------ | ------------------------------------------------------------------------- |
| **Read**     | API quantity → number in `unit` (or cores / bytes when `unit` is omitted) |
| **Form / CEL** | Number in `unit`. Edit-mode `original` is built from the same read     |
| **Write**    | Number → quantity string (`10` + `Gi` → `"10Gi"`; `0.5` → `"0.5"`)        |
| **Unchanged** | If the form number still equals the stored quantity, the stored string is kept (e.g. `"10240Mi"` stays, not rewritten as `"10Gi"`) |
| **Display**  | Overview uses `unitLabel` (or `unit`): `0.5 cores`, `10 Gi`               |

Bare numbers from the API are always the base unit (cores or bytes). Kubernetes
itself emits them when it canonicalises (e.g. `"1000m"` → `"1"`). With
`unit: m`, that `"1"` is read as `1000`.

## Native Validation:

Validates that input is numeric (same coerced-number Zod schema as `number`).

## Validation Auto-Mapping:

Same HTML `min` / `max` mapping as [number](number-field.md#validation-auto-mapping).
Omitting `step` applies HTML `step="any"`.

## Examples:

### CPU (cores)

```yaml
cpu:
  uiType: quantity
  path: spec.components.engine.resources.cpu
  fieldParams:
    label: CPU
    unitLabel: cores
  validation:
    required: true
    min: 0.1
```

Form value `0.5` is written as `"0.5"`. A stored `"500m"` is shown as `0.5`.

### Memory / disk

```yaml
disk:
  uiType: quantity
  path: spec.components.engine.storage.size
  fieldParams:
    label: Disk
    unit: Gi
  validation:
    required: true
    min: 1
    modes:
      edit:
        celExpressions:
          - celExpr: spec.components.engine.storage.size >= original.spec.components.engine.storage.size
            message: Disk size cannot be decreased
```

`min: 1` means 1 Gi. A stored `"10240Mi"` is shown as `10`. The edit CEL rule
compares numbers (`10 >= 10`), not `"10" >= "10240Mi"`.
