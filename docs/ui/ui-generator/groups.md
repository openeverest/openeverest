# Groups

Groups allow you to organize multiple fields together with different layout options.

## Table of Contents

- [Line Group](#line-group)
- [Accordion Group](#accordion-group)
- [Bordered Group](#bordered-group)

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

Displays components inside a static bordered card. Use it to visually separate a block of related fields from the rest of the section. The card has no chevron or switch: it is always expanded.

- **Heading is optional.** With `label` and/or `description`, the card shows a heading: `label` as the title, `description` as a caption below it.
- **Without `label` and `description`**, it is just a bordered box around the fields, with no heading row.
- Grouping is visual only: nested fields keep their own `path`, so a bordered group does not change the API payload.

### Card with a heading

```yaml
storage:
  uiType: group
  groupType: bordered
  label: Storage
  description: Defines the type and performance of storage for your database.
  components:
    storageClass:
      uiType: text
      path: spec.components.engine.storage.class
      fieldParams:
        label: Storage class
  componentsOrder:
    - storageClass
```

### Box without a heading

```yaml
resourcesBox:
  uiType: group
  groupType: bordered
  components:
    cpu:
      uiType: number
      path: spec.components.engine.resources.limits.cpu
      fieldParams:
        label: CPU
    memory:
      uiType: number
      path: spec.components.engine.resources.limits.memory
      fieldParams:
        label: Memory
  componentsOrder:
    - cpu
    - memory
```

### Nesting other groups

A bordered group can contain other groups, for example a `line` group to put fields in one row:

```yaml
advanced:
  uiType: group
  groupType: bordered
  label: Advanced
  components:
    inner:
      uiType: group
      groupType: line
      components:
        replicas: { ... }
        image: { ... }
      componentsOrder:
        - replicas
        - image
  componentsOrder:
    - inner
```

//TODO visual example
