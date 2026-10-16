# Quantity Field

## Table of Contents

- [Properties](#properties)
- [Values: form, CEL and API](#values-form-cel-and-api)
- [Units](#units)
- [Examples](#examples)
  - [CPU](#cpu)
  - [Memory](#memory)
  - [CEL](#cel)

A numeric input for a Kubernetes [resource quantity](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/), such as CPU, memory or disk size. Use it instead of a `number` field whenever the API path holds a quantity (`resources.limits.*`, `resources.requests.*`, `storage.size`): the value is always sent as a quantity string, never as a float, which Kubernetes rejects on these paths.

## Properties

- `uiType`: `"quantity"` (required)
- `path` OR `id`: Data path or unique identifier (required)
- `fieldParams`: Configuration object with the following optional properties:
  - `unit`: Kubernetes unit of the value in the form (default `""`, the base unit: cores, bytes). See [Units](#units)
  - `unitLabel`: Text shown after the input instead of `unit`, e.g. `cores`
  - `label`, `placeholder`, `defaultValue`, `disabled`, `autoFocus`, `helperText`, `step`: as in the [number field](number-field.md#properties)
  - `badge` and `badgeToApi` are not supported: `unit` replaces them
- `validation` (optional): the same rules as the [number field](number-field.md#properties) (`required`, `min`, `max`, `gt`, `lt`, `int`, `multipleOf`, `safe`, `regex`, `celExpressions`), applied to the number in `unit`

## Values: form, CEL and API

| Where       | Value                                  | Example (`unit: Gi`)                     |
| ----------- | -------------------------------------- | ---------------------------------------- |
| API (read)  | any quantity string or integer         | `"10Gi"`, `"10240Mi"`, `"644245094400m"` |
| Form        | a number in `unit`                     | `10`, `10`, `0.6`                        |
| CEL         | the same number, also under `original` | `10`                                     |
| API (write) | a quantity string in `unit`            | `"10Gi"`                                 |

- A stored value in any supported unit is converted into `unit`.
- A value the user did not change is sent back exactly as stored (e.g. `"10240Mi"` stays `"10240Mi"`).
- A stored value that is not a quantity is shown and sent as is.

## Units

`unit` is a Kubernetes quantity suffix: `""` (base unit), `m`, `k`, `M`, `G`, `T`, `P`, `E`, `Ki`, `Mi`, `Gi`, `Ti`, `Pi`, `Ei`.

| Resource | Typical `unit` | Form value for `"500m"` / `"1Gi"` |
| -------- | -------------- | --------------------------------- |
| CPU      | `""` (cores)   | `0.5`                             |
| CPU      | `m`            | `500`                             |
| Memory   | `Gi`           | `1`                               |
| Memory   | `Mi`           | `1024`                            |

## Examples

### CPU

```yaml
cpu:
  uiType: quantity
  path: spec.components.engine.resources.limits.cpu
  fieldParams:
    label: CPU
    unitLabel: cores
    step: 0.1
  validation:
    required: true
    min: 0.1
```

`0.5` is sent as `"0.5"`.

### Memory

```yaml
memory:
  uiType: quantity
  path: spec.components.engine.resources.limits.memory
  fieldParams:
    label: Memory
    unit: Gi
  validation:
    min: 0.5
```

`4` is sent as `"4Gi"`.

### CEL

CEL sees the number in `unit`, so rules compare plain numbers:

```yaml
disk:
  uiType: quantity
  path: spec.components.engine.storage.size
  fieldParams:
    label: Disk
    unit: Gi
  validation:
    modes:
      edit:
        celExpressions:
          - celExpr: "spec.components.engine.storage.size >= original.spec.components.engine.storage.size"
            message: Disk size cannot be decreased
```

The Kubernetes CEL `quantity()` library is not available in the UI.
