# ArrayField

## Table of Contents

- [Properties](#properties)
- [Validation](#validation)
- [Examples](#examples)
  - [Key/Value Map](#keyvalue-map)
  - [Validated Keys and Values](#validated-keys-and-values)
- [Behavior](#behavior)

A list of rows the user adds and removes. Each row is an item rendered from `itemTemplate`; the
field writes all rows to its `path` in the API shape set by `outputType`.

Supported today: `outputType: object` with `itemTemplate.uiType: keyValue`, a string map such as
labels or annotations. Other combinations, like a list of strings or of objects, are planned; until
then such a field isn't rendered (see [Behavior](#behavior)).

## Properties:

- `uiType`: `"array"` (**Required**)
- `path`: Where the map is written, e.g. `spec.components.proxy.service.annotations` (**Required**)
- `outputType`: `"object"` — rows are written as a map, `{ key: value }` (**Required**)
- `itemTemplate` (**Required**):
  - `uiType`: `"keyValue"` — a row of a key and a value input
  - `fieldParams` (optional):
    - `key`: Parameters of the key input, as for a [text field](text-field.md), e.g. `placeholder`
    - `value`: Parameters of the value input, as for a [text field](text-field.md)
  - `validation` (optional):
    - `key`: Rules for every key, as for a [text field](text-field.md), except `celExpressions`
    - `value`: Rules for every value, as for a [text field](text-field.md), except `celExpressions`
- `fieldParams`: Configuration object with the following properties:
  - `label`: Display label, shown above the rows
  - `addButtonLabel`: Text of the button that adds a row (default: `Add`)
  - `defaultValue`: Initial entries, written in the API shape, e.g. `{ team: db }`
  - `helperText`: Help text displayed below the rows
  - `disabled`: Whether the field is disabled (default: `false`)
  - `modes`: Per-mode overrides, see [FieldParams-level modes](../Readme.md#fieldparams-level-modes)
- `validation` (optional): Rules for the list, see [Validation](#validation)

## Validation

Rules for the list as a whole:

- `required`: At least one entry
- `minItems`: At least this many entries
- `maxItems`: At most this many entries

Only filled rows count. `regex` and `celExpressions` aren't supported on the list yet.

These rules always apply, without being set:

- **Keys are unique.** A map holds a key once, so a key set again is reported on the later row,
  instead of one value silently replacing the other.
- **A filled row has a key.** A row with only a value is reported.

## Examples:

### Key/Value Map

```yaml
annotations:
  uiType: array
  outputType: object
  path: spec.components.proxy.service.annotations
  fieldParams:
    label: Load balancer annotations
    addButtonLabel: Add annotation
    defaultValue:
      service.beta.kubernetes.io/aws-load-balancer-type: nlb
  itemTemplate:
    uiType: keyValue
    fieldParams:
      key:
        placeholder: service.beta.kubernetes.io/…
      value:
        placeholder: Value
```

### Validated Keys and Values

```yaml
podMonitorLabels:
  uiType: array
  outputType: object
  path: parameters.podMonitorLabels
  fieldParams:
    label: PodMonitor labels
  validation:
    required: true
    maxItems: 10
  itemTemplate:
    uiType: keyValue
    validation:
      key:
        max: 63
        regex:
          pattern: "^[A-Za-z0-9]([-A-Za-z0-9_./]*[A-Za-z0-9])?$"
          message: Use letters, numbers, '-', '_', '.' and '/'
      value:
        required: true
```

## Behavior:

- **Empty rows**: a row with neither key nor value is dropped on save, not reported.
- **Empty values**: an entry whose value is empty is sent with it, e.g. a label `infra: ""`: a map
  still holds the key. Set `itemTemplate.validation.value.required` if an entry needs a value.
- **No entries**: nothing is written on create. On edit, the saved map is removed.
- **Edit**: the saved map is replaced as a whole, so a removed entry is removed from the instance.
- **Order**: entries keep the order of the rows. The API doesn't keep a map's order, so on edit
  they come back in the order the API returns.
- **Preview and overview**: entries are shown as `key = value`, separated by commas.
- **New row**: the key input gets the focus.
- **Not rendered**: a field in a shape other than `outputType: object` with `itemTemplate.uiType:
  keyValue`, or one whose path another field also writes, writes inside or sits in (the array
  writes its whole value). It is ignored like an unknown `uiType`: not shown, not sent, and an edit
  keeps the saved value. `provider-sdk generate` rejects such a schema (rules `array-unsupported`
  and `array-overlap`).

Where the field works is listed in [Known Limitations](../known-limitations.md#arrays).
