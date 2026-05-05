# Section Group

`groupType: section` renders a bordered card with a title and optional description, grouping related fields visually. It is the schema-driven equivalent of the `FormCard` pattern used in the hardcoded form steps.

## Parameters

| Key               | Type        | Description                                                         |
| ----------------- | ----------- | ------------------------------------------------------------------- |
| `uiType`          | `"group"`   | Required.                                                           |
| `groupType`       | `"section"` | Required. Selects this wrapper.                                     |
| `label`           | `string`    | Card title. Rendered with `sectionHeading` typography.              |
| `description`     | `string`    | Helper text below the title. Rendered with `caption` typography.    |
| `components`      | `object`    | Child components or nested groups.                                  |
| `componentsOrder` | `string[]`  | Render order of `components` keys.                                  |
| `modes`           | `object`    | Group-level mode overrides — `hidden` or `disabled` per `FormMode`. |

## Usage

```yaml
storageSection:
  uiType: group
  groupType: section
  label: Storage
  description: Defines the type and performance of storage for your database.
  components:
    storageClass:
      uiType: select
      path: spec.components.engine.storage.storageClass
      fieldParams:
        label: Storage Class
        options:
          - { label: local-path, value: local-path }
          - { label: gp3, value: gp3 }
  componentsOrder:
    - storageClass
```

## Mode Overrides

Group-level `modes` follow the same pattern as `component.modes` and `fieldParams.modes`.

**`hidden: true`** — removes the section from the form entirely for that mode. Child validation is skipped.

```yaml
credentialsSection:
  uiType: group
  groupType: section
  label: Credentials
  modes:
    edit:
      hidden: true
  components:
    password:
      uiType: text
      path: spec.credentials.password
      fieldParams:
        label: Password
```

**`disabled: true`** — renders the card grayed out with all interactions blocked.

```yaml
resourcesSection:
  uiType: group
  groupType: section
  label: Resources
  modes:
    restore:
      disabled: true
  components:
    cpu:
      uiType: number
      path: spec.components.engine.resources.limits.cpu
      fieldParams:
        label: CPU
```

## Nesting

A section can contain other group types. Use `groupType: line` inside a section to lay fields out horizontally.

```yaml
resourcesSection:
  uiType: group
  groupType: section
  label: Resources
  description: Configure CPU and memory for your database nodes.
  components:
    limits:
      uiType: group
      groupType: line
      components:
        cpu:
          uiType: number
          path: spec.components.engine.resources.limits.cpu
          fieldParams:
            label: CPU
            defaultValue: 1
        memory:
          uiType: number
          path: spec.components.engine.resources.limits.memory
          fieldParams:
            label: Memory
            defaultValue: 4
            badge: Gi
            badgeToApi: true
```
