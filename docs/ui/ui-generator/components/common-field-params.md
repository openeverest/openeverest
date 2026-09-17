# Common Field Parameters

All UIGenerator field types (`text`, `number`, `select`, `hidden`) share a base set of `fieldParams`
properties documented here. Field-specific parameters are documented separately.

## Table of Contents

- [label](#label)
- [defaultValue](#defaultvalue)
- [disabled](#disabled)
- [autoFocus](#autofocus)
- [helperText](#helpertext)
- [info](#info)
- [tooltip](#tooltip)
- [badge / badgeToApi](#badge--badgetoapitrue)
- [modes](#modes)

---

## `label`

Display label rendered above (or floating inside) the field.

```yaml
fieldParams:
  label: Storage Class
```

---

## `defaultValue`

Initial value pre-filled when the form opens in `New` mode (or any mode that does not have an
`extractInstanceValues` result for this field).

```yaml
fieldParams:
  defaultValue: "local-path"
```

---

## `disabled`

Prevents the user from interacting with the field. The value is still submitted with the form.

```yaml
fieldParams:
  disabled: true
```

Use `fieldParams.modes` to disable only in specific form modes (e.g., prevent editing a field
after creation):

```yaml
fieldParams:
  label: Database name
  modes:
    edit:
      disabled: true
```

---

## `autoFocus`

Moves browser focus to this field when the form step renders. Use at most once per step.

```yaml
fieldParams:
  autoFocus: true
```

---

## `helperText`

Short guidance text rendered below the field (similar to a MUI `FormHelperText`). Replaced by
the validation error message when the field is in an error state.

```yaml
fieldParams:
  helperText: "Must be unique within the namespace"
```

---

## `info`

Informational text shown as a small **`ℹ`** (`InfoOutlined`) icon button next to the field.
Hovering or focusing the icon reveals the text in a tooltip.

Use `info` to surface additional context — explanations of why a setting exists, links to
documentation concepts, or guidance that is too long for `helperText` — without cluttering the
visible field area.

```yaml
fieldParams:
  label: Storage Class
  info: >
    The storage class determines how persistent volumes are provisioned.
    Choose "local-path" for single-node clusters or a cloud-provider class
    for production deployments.
```

### When to use `info` vs `helperText`

| | `helperText` | `info` |
|---|---|---|
| **Visibility** | Always visible below the field | Hidden behind an icon, shown on hover |
| **Length** | Keep it short (one line) | Can be a full sentence or two |
| **Error interaction** | Replaced by the validation error | Not affected by validation errors |
| **Best for** | Format hints, units, constraints | Background context, rationale, docs links |

### Accessibility

The icon button has `aria-label="Field information"` so screen-reader users can discover and
activate it with keyboard navigation.

### Example — plugin developer context

```yaml
storageClass:
  uiType: select
  path: spec.components.engine.storage.storageClass
  fieldParams:
    label: Storage Class
    info: >
      "local-path" is pre-installed in most Kubernetes distributions.
      For production, use a cloud-provider storage class that supports
      ReadWriteOnce (RWO) access mode.
    options:
      - label: local-path
        value: local-path
```

---

## `tooltip`

> **Note:** The `tooltip` feature wraps the entire field in an MUI `<Tooltip>` that activates
> on hover. For most use cases, prefer `info` (an icon button) which avoids the layout
> compensation currently required by `tooltip`. Active tracking issue: [#2007].

Text shown when the user hovers anywhere over the field. Useful for explaining why a field is
disabled.

```yaml
fieldParams:
  disabled: true
  tooltip: "Storage class cannot be changed after the cluster is created."
```

---

## `badge` / `badgeToApi: true`

Appends a unit suffix (e.g. `Gi`, `Mi`) to the right of the input.

- `badge: "Gi"` — displays the suffix in the input adornment.
- `badgeToApi: true` — appends the badge string to the submitted value before sending it to the
  API (e.g. the user types `25` and the API receives `"25Gi"`).

```yaml
fieldParams:
  label: Disk
  badge: "Gi"
  badgeToApi: true
```

---

## `modes`

Per-mode overrides for a subset of `fieldParams` properties. Supported overrides:

| Property | Type | Description |
|---|---|---|
| `disabled` | `boolean` | Override disabled state for this mode |
| `readOnly` | `boolean` | Override readOnly state for this mode |
| `label` | `string` | Override the field label |
| `helperText` | `string` | Override the helper text |
| `defaultValue` | `unknown` | Override the default value |
| `autoFocus` | `boolean` | Override autoFocus |

```yaml
dbName:
  uiType: text
  path: metadata.name
  fieldParams:
    label: Database name
    helperText: Must be unique within the namespace
    modes:
      edit:
        disabled: true
        helperText: Name cannot be changed after creation
```

See [Readme.md — Mode-Aware Overrides](../Readme.md#mode-aware-overrides) for full details.
