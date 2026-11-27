# MapField

> **Tech Preview.** The map field may change in a coming release without a deprecation period.
> Try it, but don't rely on its schema contract yet.

## Table of Contents

- [Properties](#properties)
- [The Row](#the-row)
  - [Not Supported Yet](#not-supported-yet)
- [Validation](#validation)
  - [A CEL Rule on the Map](#a-cel-rule-on-the-map)
- [Locking Rows](#locking-rows)
  - [Nothing Can Be Changed](#nothing-can-be-changed)
  - [Fixed Keys](#fixed-keys)
  - [No Removing](#no-removing)
  - [No Adding](#no-adding)
  - [A Fixed Set](#a-fixed-set)
  - [Append-Only](#append-only)
- [Examples](#examples)
  - [Key/Value Map](#keyvalue-map)
  - [Validated Keys](#validated-keys)
- [Behavior](#behavior)

A map of strings, such as labels, annotations or environment variables, edited as rows the user
adds and removes. Each row is a key and a value; the field writes `{ key: value, … }` to its
`path`.

For a list of strings, edited as the same rows with one input each, see the
[list field](list-field.md).

## Properties:

- `uiType`: `"map"` (**Required**)
- `path`: Where the map is written, e.g. `spec.components.proxy.service.annotations`. One path
  only (**Required**)
- `itemTemplate`: One row (**Required**): a group of a text field for the key and a text field
  for the value, see [The Row](#the-row)
- `fieldParams`: Configuration object with the following properties:
  - `label`: Display label, shown above the rows
  - `addButtonLabel`: Text of the button that adds a row (default: `Add`)
  - `defaultValue`: Initial entries, written in the API shape, e.g. `{ team: db }`. Only a map of
    strings: the UI ignores anything else and `provider-sdk generate` rejects it (rule
    `map-invalid-default`)
  - `helperText`: Help text displayed below the rows
  - `disabled`: Whether the field is disabled: every input, the Add button and every Remove
    button (default: `false`)
  - `disableAdd`: Whether the Add button is disabled (default: `false`), see
    [Locking Rows](#locking-rows)
  - `disableRemove`: Whether the Remove button is disabled on the rows the form opened with
    (default: `false`), see [Locking Rows](#locking-rows)
  - `modes`: Per-mode overrides, see [FieldParams-level modes](../Readme.md#fieldparams-level-modes).
    A map also takes `disableAdd` and `disableRemove` there
- `validation` (optional): Rules for the map, see [Validation](#validation)

## The Row

`itemTemplate` describes one row, and the field repeats it for every entry. A row is a
[group](../groups.md) of two [text fields](text-field.md), written like any other group:

```yaml
itemTemplate:
  uiType: group
  groupType: line
  components:
    key:
      uiType: text
      path: key
    value:
      uiType: text
      path: value
```

- **`path` says which field is which.** Inside a row a `path` is relative to the row: the field
  with `path: key` holds the entry's key, the one with `path: value` its value. A row has exactly
  these two. The component names (`key` and `value` above) are yours, as anywhere in a schema.
- **Layout.** `groupType: line` puts the two inputs side by side. With no `groupType` they are
  stacked, and a line separates the rows. The button that removes the row sits to the right of it
  in both.
- **Params and rules are a text field's.** `label`, `placeholder`, `helperText`, `type: password`,
  `disabled` and `modes` under `fieldParams`; `required`, `min`, `max`, `regex` and the transforms
  under `validation`, with `modes` too. Transforms such as `trim` or `toLowerCase` change what is
  written, and duplicate keys are found after them. With no `placeholder`, the inputs show `Key`
  and `Value`.
- **`disabled` applies to the rows the form opened with.** A row the user adds is editable, see
  [Locking Rows](#locking-rows).
- **The map decides what is written.** `uiType: map` writes the rows as one object,
  `{ key: value, … }`. The row only describes the inputs.

### Not Supported Yet

A row is checked when a provider is built: `provider-sdk generate` rejects everything below. All
of it is planned, none is ruled out.

**The map isn't rendered** (rule `map-unsupported`). It is treated like an unknown `uiType`: not
shown, not sent, and an edit keeps its saved value.

- A key or a value that isn't `uiType: text`. Both inputs are text, so the map writes strings. A
  typed value is being considered: a number, a quantity, a toggle, or a select, with its options
  from a `dataSource` too.
- A third field, e.g. a note beside the value: an entry holds one value.
- A group inside the row.
- A `groupType` other than `line`.
- A widget, or anything else that isn't the group above.

**The option is ignored** (rule `map-ignored-option`). The map is rendered without it.

- `defaultValue` on a row's field, in `fieldParams` or in a mode: a new row starts empty. The
  map's own `defaultValue` sets the initial entries.
- `celExpressions` on a row's field: a row can't evaluate CEL.
- `dataSource` on a row's field: the input stays a plain text input.
- `modes` on a row's field, e.g. `modes: { edit: { uiType: hidden } }`: a row has the same two
  fields in every form mode. `fieldParams.modes` and `validation.modes` do work.
- `badge` / `badgeToApi` on a row's field.

## Validation

Rules for the map as a whole:

- `required`: At least one entry
- `min`: At least this many entries
- `max`: At most this many entries
- `celExpressions`: CEL rules about the map, see [A CEL Rule on the Map](#a-cel-rule-on-the-map)

On the map, `min` and `max` count entries, the way they count characters on a
[text field](text-field.md). The map's `min` and `max` are whole numbers of 0 or more, and `min`
can't be above `max` in any form mode, or no map would pass: `provider-sdk generate` rejects that
(rule `map-invalid-limits`). Only filled rows count. `regex` isn't supported on the map: the UI
ignores it and `provider-sdk generate` rejects it (rule `map-ignored-option`).

On a row's field, `min` and `max` are still a text field's: `max: 63` there limits the length of a
key or a value, not the number of entries.

A count error shows under the rows in place of the helper text and says what to do, e.g.
`Remove 1 entry (at most 2 allowed)`. It doesn't hide the rows' own errors, which stay under
their inputs. The map is checked again as the user types, whenever a key changes or a row fills
or empties.

### A CEL Rule on the Map

A rule reads the map at its `path` as the map it writes, `{ key: value }`:

```yaml
annotations:
  uiType: map
  path: spec.components.proxy.service.annotations
  fieldParams:
    label: Annotations
  validation:
    celExpressions:
      - celExpr: >-
          !has(spec.components.proxy.service.annotations) ||
          spec.components.proxy.service.annotations.all(k, !k.startsWith('kubernetes.io/'))
        message: Keys under kubernetes.io/ are reserved
```

- **No entries, no map.** A map with no entries writes nothing, so the rule finds it unset and
  asks for it with `has()` first. Without that the user sees `CEL expression error: …`.
- **Every entry.** `.all(k, …)` and `.exists(k, …)` run a condition for each key `k`; the value
  is `<path>[k]`.
- **Other fields.** The rule can read any field.
- **When it runs.** Again whenever a key or a value changes.
- **Where the error shows.** Under the rows, in place of the helper text, like a count error. It
  names no row, so say in `message` which entries are wrong. One error shows there at a time: a
  count error first, then the rule's. The rows' own errors stay under their inputs.
- **A rule on another field** reads the map the same way, and its error shows under that field.

More expressions are in [Reading a Map Field](../validation.md#reading-a-map-field).

Not supported yet: a rule on a row's key or value, see [Not Supported Yet](#not-supported-yet).
Such a rule needs the row it stands in (this key, this value), which CEL isn't given yet. Until
then, check the entries from the map with `.all(k, …)`.

These rules always apply, without being set:

- **Keys are unique.** A map holds a key once, so a key set again is reported on the later row,
  instead of one value silently replacing the other.
- **A filled row has a key.** A row with only a value is reported.

## Locking Rows

A schema can fix the entries a form starts with and still let the user add new ones. For that the
form tells two kinds of rows apart:

- **Rows the form opened with**: the entries the form held when it loaded. When an instance is
  edited or restored, these are its saved entries. In the create wizard, they are the entries of
  the map's `defaultValue`, or of the preset the user picked.
- **Added rows**: rows the user added with the Add button since then.

**A row field's `disabled` applies to the rows the form opened with. A row added since is
editable.** The same goes for `readOnly`. An added row's inputs are checked by the field's rules.
In a row the form opened with, a disabled input skips its own rules, like any
[disabled field](../validation.md#disabled-fields).

| Set                         | Effect                                                                                      |
| --------------------------- | ------------------------------------------------------------------------------------------- |
| `disabled` on a row's field | That input is disabled in the rows the form opened with                                     |
| `disableAdd` on the map     | The Add button is disabled                                                                  |
| `disableRemove` on the map  | The rows the form opened with can't be removed, or emptied. An added row can be removed     |
| `disabled` on the map       | Everything is disabled: every input, the Add button and every Remove button, added rows too |

All four go under `fieldParams`, and under `fieldParams.modes.<mode>` to apply in one form mode
only. A disabled button stays visible.

- **Every row the form opened with is locked the same way.** One entry can't be locked alone.
- **A saved entry is no longer an added one.** After a save, the next form that opens the instance
  starts with the entry, so the locks apply to it.
- **The rules that always apply still check every row.** A key an added row repeats is reported on
  the added row, also when the other row's key is disabled.
- **What is written doesn't change.** The map is written whole from the keys and values of its
  rows, locked or not.
- **Only the form is locked.** The API doesn't know these params, so a change made outside the
  form, e.g. with kubectl, isn't stopped.

The snippets below lock the edit form. Without `modes`, the same params lock every form, the
create wizard too, where they apply to the entries of `defaultValue`.

### Nothing Can Be Changed

`disabled` on the map: no input can be typed in, and no row added or removed.

```yaml
labels:
  uiType: map
  path: spec.labels
  fieldParams:
    label: Labels
    modes:
      edit:
        disabled: true
  itemTemplate:
    uiType: group
    groupType: line
    components:
      key: { uiType: text, path: key }
      value: { uiType: text, path: value }
```

### Fixed Keys

`disabled` on the key field: a saved key can't be changed, its value can. The user can add
entries, and remove any.

```yaml
labels:
  uiType: map
  path: spec.labels
  fieldParams:
    label: Labels
  itemTemplate:
    uiType: group
    groupType: line
    components:
      key:
        uiType: text
        path: key
        fieldParams:
          modes:
            edit:
              disabled: true
      value:
        uiType: text
        path: value
```

### No Removing

`disableRemove` on the map: a saved entry can't be removed. Its Remove button is disabled, and
clearing both its inputs is reported with `Enter a key` instead of dropping the row. Its key and
value can still be changed, so disable the key field too to keep the entry under its key.

```yaml
labels:
  uiType: map
  path: spec.labels
  fieldParams:
    label: Labels
    modes:
      edit:
        disableRemove: true
  itemTemplate:
    uiType: group
    groupType: line
    components:
      key: { uiType: text, path: key }
      value: { uiType: text, path: value }
```

### No Adding

`disableAdd` on the map: no entry can be added. The saved ones can be changed and removed.

```yaml
labels:
  uiType: map
  path: spec.labels
  fieldParams:
    label: Labels
    modes:
      edit:
        disableAdd: true
  itemTemplate:
    uiType: group
    groupType: line
    components:
      key: { uiType: text, path: key }
      value: { uiType: text, path: value }
```

### A Fixed Set

The key disabled, with `disableAdd` and `disableRemove`: the entries stay the ones saved, and only
their values change.

```yaml
labels:
  uiType: map
  path: spec.labels
  fieldParams:
    label: Labels
    modes:
      edit:
        disableAdd: true
        disableRemove: true
  itemTemplate:
    uiType: group
    groupType: line
    components:
      key:
        uiType: text
        path: key
        fieldParams:
          modes:
            edit:
              disabled: true
      value:
        uiType: text
        path: value
```

### Append-Only

The key and the value disabled, with `disableRemove`: a saved entry can't be changed or removed.
The user can add entries, and change or remove one until it is saved.

```yaml
labels:
  uiType: map
  path: spec.labels
  fieldParams:
    label: Labels
    modes:
      edit:
        disableRemove: true
  itemTemplate:
    uiType: group
    groupType: line
    components:
      key:
        uiType: text
        path: key
        fieldParams:
          modes:
            edit:
              disabled: true
      value:
        uiType: text
        path: value
        fieldParams:
          modes:
            edit:
              disabled: true
```

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
    uiType: group
    groupType: line
    components:
      key:
        uiType: text
        path: key
        fieldParams:
          placeholder: service.beta.kubernetes.io/…
      value:
        uiType: text
        path: value
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
    max: 10
  itemTemplate:
    uiType: group
    groupType: line
    components:
      key:
        uiType: text
        path: key
        validation:
          regex:
            pattern: '^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*/)?[A-Za-z0-9]([-A-Za-z0-9_.]{0,61}[A-Za-z0-9])?$'
            message: An optional DNS prefix and '/', then up to 63 letters, digits, '-', '_' or '.'
      value:
        uiType: text
        path: value
        validation:
          max: 63
```

## Behavior:

- **Empty rows**: a row with neither key nor value is dropped on save, not reported.
- **Empty values**: an entry whose value is empty is sent with it, e.g. a label `infra: ""`: a map
  still holds the key. Set `validation.required` on the row's value field if an entry needs a value.
- **No entries**: nothing is written on create. On edit, the saved map is removed.
- **Edit**: the saved map is replaced as a whole, so a removed entry is removed from the instance.
- **Order**: entries keep the order of the rows. The API doesn't keep a map's order, so on edit
  they come back in the order the API returns.
- **Preview and overview**: an entry-count chip, e.g. *3 entries*. It opens a key/value table in a
  dialog, where each row can be copied as `key=value`. A row field with `type: password` is shown
  masked, in the table too, and such a row can't be copied.
- **New row**: starts empty and editable, also when a row field is `disabled`, and its first
  input gets the focus. See [Locking Rows](#locking-rows).
- **Long keys and values**: one too long for its input ends in an ellipsis, and hovering the input
  shows it in full. Not for a field with `type: password`, and not for a field with its own
  `tooltip`, which is shown instead.
- **Screen readers**: a row's inputs are named by their label, or `Key` / `Value`, and the row
  number, e.g. `Key 2`.
- **Not rendered**: a map whose `itemTemplate` isn't [the row above](#the-row), a map with several
  paths (an edit reads only the first), or one whose path another field also writes, writes inside
  or sits in (the map writes its whole value). It is ignored like an unknown `uiType`: not shown,
  not sent, and an edit keeps the saved value. `provider-sdk generate` rejects such a schema (rules
  `map-unsupported`, `map-multi-path` and `map-overlap`).

Where the field works is listed in [Known Limitations](../known-limitations.md#maps-and-lists).
