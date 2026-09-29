# Groups

Groups allow you to organize multiple fields together with different layout options.

## Table of Contents

- [Line Group](#line-group)
- [Accordion Group](#accordion-group)
- [Bordered Group](#bordered-group)
- [Toggleable Group](#toggleable-group)

Related: [Number Field](components/number-field.md), [Select Field](components/select-field.md), [Text Field](components/text-field.md), [Validation](validation.md)

## Line Group

//TODO will be renamed, documentation should be checked before merging
Displays components in a horizontal line (flex layout).

```yaml
resourceGroup:
  uiType: group
  groupType: line
  label: Resources
  components:
    cpu: { ... }
    memory: { ... }
    disk: { ... }
  componentsOrder:
    - cpu
    - memory
    - disk
```

//TODO visual example

## Accordion Group

Displays components in a collapsible accordion panel.

```yaml
advancedSettings:
  uiType: group
  groupType: accordion
  label: Advanced Settings
  description: Optional advanced configuration
  components:
    setting1: { ... }
    setting2: { ... }
```

//TODO visual example

## Bordered Group

A static bordered card around related fields. Visual only: fields keep their own `path`.

- `label` and `description` are optional. Omit both for a plain box without a heading.
- Can contain other groups, e.g. a `line` group.

```yaml
storage:
  uiType: group
  groupType: bordered
  label: Storage
  description: Defines the type and performance of storage for your instance.
  components:
    storageClass:
      uiType: text
      path: spec.components.engine.storage.class
      fieldParams:
        label: Storage class
  componentsOrder:
    - storageClass
```

![Bordered group](images/bordered-group.png)

## Toggleable Group

A bordered card with an **Enable** switch. The switch exists only in the form and is not sent to the API.

```yaml
monitoring:
  uiType: group
  groupType: toggleable
  label: Monitoring
  components:
    endpoint:
      uiType: text
      path: spec.monitoring.endpoint
      fieldParams:
        label: Endpoint
      validation:
        required: true
  componentsOrder:
    - endpoint
```

**Behavior**

- **Off:** fields are hidden, not validated, and removed from the request. On an existing instance, turning it off deletes the saved values.
- **Initial state:** off for a new instance; on if the instance already has a value in any of the group's fields.

**Rules** — if one is broken, the group renders as a plain bordered group (with a warning in development builds):

- The group has at least one field with a `path`.
- No field outside the group writes the same `path`.
- No toggleable inside another toggleable.
- Section and group keys use only letters, digits, `_` and `-`.

**Keep in mind**

- A CEL rule outside the group sees the group's fields as absent while it is off. Guard them with `has()` on the exact field path: `!has(spec.monitoring.endpoint) || ...`. Development builds warn about an unguarded reference.
- Don't put fields that are read-only in edit mode inside the group: turning it off deletes them too.
- Supported in instance (topology) schemas only, not in backup-class schemas.

![Toggleable group, off](images/toggleable-group-off.png)

![Toggleable group, on](images/toggleable-group-on.png)
