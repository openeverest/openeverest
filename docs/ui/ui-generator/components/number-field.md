# Number Field

## Table of Contents

- [Properties](#properties)
- [Native Validation](#native-validation)
- [Validation Auto-Mapping](#validation-auto-mapping)
- [Unit Badge](#unit-badge)
- [Examples](#examples)
  - [Basic Number Field](#basic-number-field)
  - [Exclusive Bounds](#exclusive-bounds)
  - [Regexp](#regexp)
  - [CEL](#cel)

A numeric input field for integer and decimal values.

## Properties:

- `uiType`: `"number"` (required)
- `path` OR `id`: Data path or unique identifier (required)
- `fieldParams`: Configuration object with the following optional properties:
  - `label`: Display label for the field
  - `placeholder`: Placeholder text shown when field is empty
  - `defaultValue`: Default numeric value
  - `disabled`: Whether the field is disabled (default: `false`)
  - `autoFocus`: Automatically focus this field on render
  - `helperText`: Help text displayed below the field
  - `step`: Increment/decrement step for arrow buttons (e.g., `0.1`, `5`, `10`)
  - `badge`: Unit/suffix rendered at the end of the input (e.g. `Gi`). See [Unit Badge](#unit-badge)
  - `badgeToApi`: When `true`, the `badge` is treated as a **unit** — appended to the value on write and converted back on read. See [Unit Badge](#unit-badge)
- `validation` (optional): Validation rules object with the following properties:
  - `required`: Whether the field is required (default: `false`)
  - `min`: Minimum value (inclusive) - value must be >= specified number
  - `max`: Maximum value (inclusive) - value must be <= specified number
  - `gt`: Greater than (exclusive) - value must be > specified number
  - `lt`: Less than (exclusive) - value must be < specified number
  - `int`: Must be an integer (boolean: `true`)
  - `multipleOf`: Value must be a multiple of specified number
  - `safe`: Must be a safe integer within JavaScript's safe integer range (boolean: `true`)
  - `regex`: Regular expression validation (see [Regex Validation](../validation.md#regex))
  - `celExpressions`: Array of CEL validation expressions for cross-field validation (see [CEL Expression Validation](../validation.md#cel-expression-validation))

## Native Validation:

Validates that input is numeric

## Unit Badge:

`fieldParams.badge` renders a unit/suffix at the end of the input (e.g. `Gi`, `GB`, `%`). When `badgeToApi: true`, the badge is treated as a **unit** and participates in value conversion:

- **On write** the badge is appended to the value: `0.6` with `badge: Gi` becomes `0.6Gi`.
- **On read** the stored value is coerced back into the badge unit. Kubernetes may persist a resource quantity in a _different_ unit than the badge — for example it normalises `0.6Gi` to the milli-byte value `644245094400m` — so a value stored in any supported unit is converted into the badge unit (`644245094400m` with `badge: Gi` becomes `0.6`).

Supported units (Kubernetes resource quantities): `m, k, M, G, T, P, E, Ki, Mi, Gi, Ti, Pi, Ei`.

A value stored in any other (non-standard) unit is **not** converted and is shown as-is. This is expected behavior: if a stored value uses a unit outside the list above (e.g. `kg`), the raw value passes through unchanged.

```yaml
memory:
  uiType: number
  path: spec.components.engine.resources.limits.memory
  fieldParams:
    label: Memory
    badge: Gi
    badgeToApi: true
```

## Validation Auto-Mapping:

The following validation rules are automatically applied to HTML input attributes for browser-level validation:

- `validation.min` → HTML `min` attribute (inclusive lower bound)
- `validation.max` → HTML `max` attribute (inclusive upper bound)
- `validation.gt` → HTML `min` attribute (converted to exclusive lower bound)
- `validation.lt` → HTML `max` attribute (converted to exclusive upper bound)

When converting exclusive bounds (`gt`/`lt`) to HTML attributes:

- For integer validation (`int: true`): offset by 1 (e.g., `gt: 5` becomes `min="6"`)
- With `step` defined: offset by step value (e.g., `gt: 5` with `step: 0.5` becomes `min="5.5"`)
- For arbitrary decimals: offset by 0.000001 (e.g., `gt: 5` becomes `min="5.000001"`)
- Explicit `min`/`max` always take priority over converted `gt`/`lt`

## Examples:

[OpenEverest TextInput “Number type” Story](https://openeverest.io/openeverest/?path=/story/textinput--number-type)

### Basic Number Field

```yaml
replicas:
  uiType: number
  path: spec.replicas
  fieldParams:
    label: Number of Replicas
    defaultValue: 3
    step: 1
    autoFocus: true
  validation:
    required: true
    min: 1
    max: 16
    int: true
```

### Exclusive Bounds

```yaml
validation:
  gt: 0
  lt: 32
  multipleOf: 0.5
```

**Note:** Fields are optional by default. To make a field required, set `required: true` in the `validation` object. Validation rules (min/max/etc.) only apply when a value is entered.

### Regexp

```yaml
portNumber:
  uiType: number
  path: spec.port
  fieldParams:
    label: Port Number
  validation:
    regex:
      pattern: "^[1-9][0-9]{3,4}$"
      message: Port must be between 1000-99999
```

### CEL

> Documentation coming soon.
