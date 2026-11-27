# ListField

> **Tech Preview.** The list field may change in a coming release without a deprecation period.
> Try it, but don't rely on its schema contract yet.

## Table of Contents

- [Properties](#properties)
- [The Item](#the-item)
  - [Not Supported Yet](#not-supported-yet)
- [Validation](#validation)
  - [A CEL Rule on the List](#a-cel-rule-on-the-list)
- [Locking Rows](#locking-rows)
  - [Append-Only](#append-only)
- [Examples](#examples)
  - [Source Ranges](#source-ranges)
  - [TLS Additional Names](#tls-additional-names)
- [Behavior](#behavior)

A list of strings, such as allowed source ranges, extra host names or command-line arguments,
edited as rows the user adds and removes. Each row is one entry; the field writes
`[entry, …]` to its `path`, in the order of the rows.

A list is the sibling of the [map field](map-field.md): the same rows, with one input in a row
instead of a key and a value.

## Properties:

- `uiType`: `"list"` (**Required**)
- `path`: Where the list is written, e.g.
  `spec.components.proxy.service.loadBalancerService.sourceRanges`. One path only (**Required**)
- `itemTemplate`: One entry (**Required**): a text field, see [The Item](#the-item)
- `fieldParams`: Configuration object with the following properties:
  - `label`: Display label, shown above the rows
  - `addButtonLabel`: Text of the button that adds a row (default: `Add`)
  - `defaultValue`: Initial entries, written in the API shape, e.g. `[10.0.0.0/8]`. Only a list of
    strings: the UI ignores anything else and `provider-sdk generate` rejects it (rule
    `list-invalid-default`)
  - `helperText`: Help text displayed below the rows
  - `disabled`: Whether the field is disabled: every input, the Add button and every Remove
    button (default: `false`)
  - `disableAdd`: Whether the Add button is disabled (default: `false`), see
    [Locking Rows](#locking-rows)
  - `disableRemove`: Whether the Remove button is disabled on the rows the form opened with
    (default: `false`), see [Locking Rows](#locking-rows)
  - `modes`: Per-mode overrides, see [FieldParams-level modes](../Readme.md#fieldparams-level-modes).
    A list also takes `disableAdd` and `disableRemove` there
- `validation` (optional): Rules for the list, see [Validation](#validation)

## The Item

`itemTemplate` describes one entry, and the field repeats it for every row. An entry is a
[text field](text-field.md), written directly:

```yaml
itemTemplate:
  uiType: text
  fieldParams:
    placeholder: 10.0.0.0/8
```

- **No `path` and no `id`.** The field is the entry: what the user types in a row is what the
  list holds at that place.
- **Params and rules are a text field's.** `label`, `placeholder`, `helperText`, `type: password`,
  `disabled` and `modes` under `fieldParams`; `min`, `max`, `regex` and the transforms under
  `validation`, with `modes` too. Transforms such as `trim` or `toLowerCase` change what is
  written. They run after the other rules, as on any text field: a `regex` checks the entry as
  typed. With no `placeholder`, the input shows `Value`.
- **`required` changes nothing on an item.** A row left empty is dropped, not reported. To ask
  for entries, set `required` or `min` on the list.
- **`disabled` applies to the rows the form opened with.** A row the user adds is editable, see
  [Locking Rows](#locking-rows).
- **The list decides what is written.** `uiType: list` writes the rows as one array of their
  texts. The item only describes the input.

### Not Supported Yet

An item is checked when a provider is built: `provider-sdk generate` rejects everything below. All
of it is planned, none is ruled out.

**The list isn't rendered** (rule `list-unsupported`). It is treated like an unknown `uiType`: not
shown, not sent, and an edit keeps its saved value.

- An item that isn't `uiType: text`. The input is text, so the list writes strings. A typed
  item is being considered: a number, a quantity, a toggle, or a select.
- A group as the item, e.g. a name and a role: a list of objects.
- An item with a `path` or an `id`.
- A widget, or no `itemTemplate` at all.

**The option is ignored** (rule `list-ignored-option`). The list is rendered without it.

- `defaultValue` on the item, in `fieldParams` or in a mode: a new row starts empty. The list's
  own `defaultValue` sets the initial entries.
- `celExpressions` on the item: a row can't evaluate CEL.
- `dataSource` on the item: the input stays a plain text input.
- `modes` on the item, e.g. `modes: { edit: { uiType: hidden } }`: a row has the same input in
  every form mode. `fieldParams.modes` and `validation.modes` do work.
- `badge` / `badgeToApi` on the item.

## Validation

Rules for the list as a whole:

- `required`: At least one entry
- `min`: At least this many entries
- `max`: At most this many entries
- `unique`: No value more than once (default: `false`)
- `celExpressions`: CEL rules about the list, see [A CEL Rule on the List](#a-cel-rule-on-the-list)

On the list, `min` and `max` count entries, the way they count characters on a
[text field](text-field.md). The list's `min` and `max` are whole numbers of 0 or more, and `min`
can't be above `max` in any form mode, or no list would pass: `provider-sdk generate` rejects that
(rule `list-invalid-limits`). Only filled rows count. `regex` isn't supported on the list: the UI
ignores it and `provider-sdk generate` rejects it (rule `list-ignored-option`). Put the pattern on
the item, where it checks every entry.

On the item, `min` and `max` are still a text field's: `max: 43` there limits the length of an
entry, not the number of entries.

A list may hold a value twice, as a list of arguments does. With `unique: true` a repeated value
is reported on the later row, e.g. `10.0.0.0/8 is added more than once`. Values are compared as
they are written, after the item's transforms: with `toLowerCase`, `Example.com` repeats
`example.com`.

A count error shows under the rows in place of the helper text and says what to do, e.g.
`Remove 1 entry (at most 20 allowed)`. It doesn't hide the rows' own errors, which stay under
their inputs. The list is checked again as the user types, whenever a row changes.

### A CEL Rule on the List

A rule reads the list at its `path` as the list it writes, an array of strings:

```yaml
sourceRanges:
  uiType: list
  path: spec.components.proxy.service.loadBalancerService.sourceRanges
  fieldParams:
    label: Allowed source ranges
  validation:
    celExpressions:
      - celExpr: >-
          !has(spec.components.proxy.service.loadBalancerService.sourceRanges) ||
          !('0.0.0.0/0' in spec.components.proxy.service.loadBalancerService.sourceRanges)
        message: 0.0.0.0/0 opens the service to everyone
  itemTemplate:
    uiType: text
```

- **No entries, no list.** A list with no entries writes nothing, so the rule finds it unset and
  asks for it with `has()` first. Without that the user sees `CEL expression error: …`.
- **Every entry.** `.all(v, …)` and `.exists(v, …)` run a condition for each entry `v`.
- **Other fields.** The rule can read any field.
- **When it runs.** Again whenever a row changes.
- **Where the error shows.** Under the rows, in place of the helper text, like a count error. It
  names no row, so say in `message` which entries are wrong. One error shows there at a time: a
  count error first, then the rule's. The rows' own errors stay under their inputs.
- **A rule on another field** reads the list the same way, and its error shows under that field.

More expressions are in [Reading a List Field](../validation.md#reading-a-list-field).

Not supported yet: a rule on the item, see [Not Supported Yet](#not-supported-yet). Such a rule
needs the row it stands in, which CEL isn't given yet. Until then, check the entries from the list
with `.all(v, …)`.

## Locking Rows

A list locks its rows the way a map does, see
[Locking Rows](map-field.md#locking-rows) in the map field: the form tells the rows it opened
with from the rows the user added since.

| Set                         | Effect                                                                                      |
| --------------------------- | ------------------------------------------------------------------------------------------- |
| `disabled` on the item      | The input is disabled in the rows the form opened with                                      |
| `disableAdd` on the list    | The Add button is disabled                                                                  |
| `disableRemove` on the list | The rows the form opened with can't be removed, or emptied. An added row can be removed     |
| `disabled` on the list      | Everything is disabled: every input, the Add button and every Remove button, added rows too |

All four go under `fieldParams`, and under `fieldParams.modes.<mode>` to apply in one form mode
only. A disabled button stays visible.

- **An added row is checked by the item's rules**, also when the item is disabled. In a row the
  form opened with, a disabled input skips its own rules, like any
  [disabled field](../validation.md#disabled-fields).
- **With `disableRemove`, a saved entry can't be emptied either.** Clearing its input is reported
  with `Enter a value` instead of dropping the row.
- **`unique` still checks every row.** A value an added row repeats is reported on the added row,
  also when the other row is disabled.
- **Only the form is locked.** The API doesn't know these params, so a change made outside the
  form, e.g. with kubectl, isn't stopped.

### Append-Only

The item disabled, with `disableRemove`: a saved entry can't be changed or removed. The user can
add entries, and change or remove one until it is saved.

```yaml
sourceRanges:
  uiType: list
  path: spec.components.proxy.service.loadBalancerService.sourceRanges
  fieldParams:
    label: Allowed source ranges
    modes:
      edit:
        disableRemove: true
  itemTemplate:
    uiType: text
    fieldParams:
      modes:
        edit:
          disabled: true
```

## Examples:

### Source Ranges

IPv4 ranges allowed to reach a load balancer, each in CIDR notation and none of them twice:

```yaml
sourceRanges:
  uiType: list
  path: spec.components.proxy.service.loadBalancerService.sourceRanges
  fieldParams:
    label: Allowed source ranges
    addButtonLabel: Add range
    helperText: Leave empty to allow every address
    defaultValue:
      - 10.0.0.0/8
  validation:
    max: 20
    unique: true
  itemTemplate:
    uiType: text
    fieldParams:
      placeholder: 10.0.0.0/8
    validation:
      regex:
        pattern: '^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\.){3}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])/(3[0-2]|[12]?[0-9])$'
        message: Use CIDR notation, e.g. 10.0.0.0/8
```

### TLS Additional Names

Extra DNS names for a certificate. A name is written in lower case whatever the user types, so
`DB.example.com` and `db.example.com` count as the same name:

```yaml
additionalNames:
  uiType: list
  path: spec.components.engine.parameters.tls.additionalNames
  fieldParams:
    label: Additional DNS names
    addButtonLabel: Add name
  validation:
    unique: true
  itemTemplate:
    uiType: text
    fieldParams:
      placeholder: db.example.com
    validation:
      toLowerCase: true
      max: 253
      regex:
        pattern: '^(\*\.)?[A-Za-z0-9]([-A-Za-z0-9]{0,61}[A-Za-z0-9])?(\.[A-Za-z0-9]([-A-Za-z0-9]{0,61}[A-Za-z0-9])?)*$'
        message: A DNS name such as db.example.com, or a wildcard such as *.example.com
```

## Behavior:

- **Empty rows**: a row with nothing typed in it is dropped on save, not reported.
- **Blank entries**: a row of spaces only is an entry, written as typed. With `trim` on the item
  it is reported with `Enter a value`.
- **No entries**: nothing is written on create. On edit, the saved list is removed.
- **Edit**: the saved list is replaced as a whole, so a removed entry is removed from the instance.
- **Order**: entries are written in the order of the rows, and an edit shows them in the saved
  order. Rows can't be moved: to change the order, remove an entry and add it again.
- **Duplicates**: allowed, unless `validation.unique` is set.
- **Preview and overview**: an entry-count chip, e.g. _3 entries_. It opens the values in a
  dialog, where each can be copied. An item with `type: password` is shown masked, in the dialog
  too, and can't be copied.
- **New row**: starts empty and editable, also when the item is `disabled`, and its input gets
  the focus. See [Locking Rows](#locking-rows).
- **Long values**: one too long for its input ends in an ellipsis, and hovering the input shows
  it in full. Not for an item with `type: password`, and not for an item with its own `tooltip`,
  which is shown instead.
- **Screen readers**: a row's input is named by the item's label, or `Value`, and the row number,
  e.g. `Value 2`.
- **Not rendered**: a list whose `itemTemplate` isn't [the item above](#the-item), a list with
  several paths (an edit reads only the first), or one whose path another field also writes,
  writes inside or sits in (the list writes its whole value). It is ignored like an unknown
  `uiType`: not shown, not sent, and an edit keeps the saved value. `provider-sdk generate` rejects
  such a schema (rules `list-unsupported`, `list-multi-path` and `list-overlap`).

Where the field works is listed in [Known Limitations](../known-limitations.md#maps-and-lists).
