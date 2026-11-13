# MapField

> **Tech Preview.** The map field may change in a coming release without a deprecation period.
> Try it, but don't rely on its schema contract yet.

## Table of Contents

- [Properties](#properties)
- [The keyValue Widget](#the-keyvalue-widget)
- [Validation](#validation)
- [Examples](#examples)
  - [Key/Value Map](#keyvalue-map)
  - [Validated Keys](#validated-keys)
- [Behavior](#behavior)

A map of strings, such as labels, annotations or environment variables, edited as rows the user
adds and removes. Each row is a key and a value; the field writes `{ key: value, … }` to its
`path`.

## Properties:

- `uiType`: `"map"` (**Required**)
- `path`: Where the map is written, e.g. `spec.components.proxy.service.annotations`. One path
  only (**Required**)
- `itemTemplate`: What a row is (**Required**). Today one widget, see
  [The keyValue Widget](#the-keyvalue-widget):
  - `uiType`: `"widget"`
  - `widgetType`: `"keyValue"`
  - `fieldParams` (optional): `key` and `value`, each with the params of a
    [text field](text-field.md), e.g. `placeholder`, `label`, `helperText` or `type: password`,
    except `defaultValue` (a new row starts empty; the map's own `defaultValue` sets the initial
    entries) and `badge` / `badgeToApi`
  - `validation` (optional): `key` and `value`, each with the rules of a
    [text field](text-field.md), except `celExpressions`
- `fieldParams`: Configuration object with the following properties:
  - `label`: Display label, shown above the rows
  - `addButtonLabel`: Text of the button that adds a row (default: `Add`)
  - `defaultValue`: Initial entries, written in the API shape, e.g. `{ team: db }`
  - `helperText`: Help text displayed below the rows
  - `disabled`: Whether the field is disabled (default: `false`)
  - `modes`: Per-mode overrides, see [FieldParams-level modes](../Readme.md#fieldparams-level-modes)
- `validation` (optional): Rules for the map, see [Validation](#validation)

## The keyValue Widget

`keyValue` is a template widget: it stands for a piece of schema, and preprocess expands it before
anything else runs. It becomes a `line` group of two [text fields](text-field.md), a `key` and a
`value` side by side, with what you set under `fieldParams.key` / `validation.key` applied to the
key, and the same for the value:

```yaml
itemTemplate:               # what you write
  uiType: widget
  widgetType: keyValue
  fieldParams:
    value: { type: password }

# what it stands for
uiType: group
groupType: line
components:
  key:   { uiType: text, path: key,   fieldParams: { placeholder: Key } }
  value: { uiType: text, path: value, fieldParams: { placeholder: Value, type: password } }
```

So a row's inputs behave like any text field: their params and rules apply, and transforms such as
`trim` or `toLowerCase` change what is written, with duplicate keys found after them. Writing the
row as your own group isn't supported yet.

## Validation

Rules for the map as a whole:

- `required`: At least one entry
- `minItems`: At least this many entries
- `maxItems`: At most this many entries

Only filled rows count. `regex` and `celExpressions` aren't supported on the map yet.

These rules always apply, without being set:

- **Keys are unique.** A map holds a key once, so a key set again is reported on the later row,
  instead of one value silently replacing the other.
- **A filled row has a key.** A row with only a value is reported.

## Examples:

### Key/Value Map

```yaml
annotations:
  uiType: map
  path: spec.components.proxy.service.annotations
  fieldParams:
    label: Load balancer annotations
    addButtonLabel: Add annotation
    defaultValue:
      service.beta.kubernetes.io/aws-load-balancer-type: nlb
  itemTemplate:
    uiType: widget
    widgetType: keyValue
    fieldParams:
      key:
        placeholder: service.beta.kubernetes.io/…
```

### Validated Keys

A Kubernetes label key is an optional DNS prefix and `/`, then a name of up to 63 characters. A
`max` on the whole key would reject valid prefixed keys, so the pattern checks the parts:

```yaml
podMonitorLabels:
  uiType: map
  path: parameters.podMonitorLabels
  fieldParams:
    label: PodMonitor labels
  validation:
    required: true
    maxItems: 10
  itemTemplate:
    uiType: widget
    widgetType: keyValue
    validation:
      key:
        regex:
          pattern: '^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*/)?[A-Za-z0-9]([-A-Za-z0-9_.]{0,61}[A-Za-z0-9])?$'
          message: An optional DNS prefix and '/', then up to 63 letters, digits, '-', '_' or '.'
      value:
        max: 63
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
- **New row**: starts empty, and its key input gets the focus.
- **Screen readers**: a row's inputs are named by their label, or `Key` / `Value`, and the row
  number, e.g. `Key 2`.
- **Not rendered**: a map whose `itemTemplate` isn't the `keyValue` widget, a map with several
  paths (an edit reads only the first), or one whose path another field also writes, writes inside
  or sits in (the map writes its whole value). It is ignored like an unknown `uiType`: not shown,
  not sent, and an edit keeps the saved value. `provider-sdk generate` rejects such a schema (rules
  `map-unsupported`, `map-multi-path` and `map-overlap`).

Where the field works is listed in [Known Limitations](../known-limitations.md#maps-and-lists).
