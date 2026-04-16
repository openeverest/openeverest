# UI Generator

## Table of Contents

- [What is UI Generator?](#what-is-ui-generator)
- [How Does It Work?](#how-does-it-work)
- [Top-Level Structure](#top-level-structure)
  - [Topology](#topology)
  - [Sections](#sections)
  - [Components](#components)
  - [Component vs ComponentGroup](#component-vs-componentgroup)
    - [Component (Single Field)](#component-single-field)
    - [ComponentGroup (Nested Fields)](#componentgroup-nested-fields)
- [Mode-Aware Overrides](#mode-aware-overrides)
  - [Component-level modes](#component-level-modes)
  - [FieldParams-level modes](#fieldparams-level-modes)
  - [Validation-level modes](#validation-level-modes)
- Field Types
  - [Number Field](components/number-field.md)
  - [Select Field](components/select-field.md)
  - [Text Field](components/text-field.md)
- [Groups](groups.md)
  - [Line Group](groups.md#line-group)
  - [Accordion Group](groups.md#accordion-group)
- [Validation](validation.md)
  - [Default Validation](validation.md#default-validation)
  - [Schema Custom Validation](validation.md#schema-custom-validation)
  - [Common Validation Rules](validation.md#common-validation-rules)
    - [Required Field Validation](validation.md#required)
    - [Regex Validation](validation.md#regex)
  - [CEL Expression Validation](validation.md#cel-expression-validation)
  - [Mode-Aware Validation](validation.md#mode-aware-validation)
- [Advanced Properties](#advanced-properties)
  - [Path vs ID](#path-vs-id)
  - [Components Order](#components-order)
  - [CEL Condition Rendering](#cel-condition-rendering)
- [Complete Example](#complete-example)

## What is UI Generator?

`ui-generator` is a utility for dynamically generating UI forms based on JSON schema definitions. It allows developers to create complex multi-step forms without writing repetitive UI code.

## How Does It Work?

- You define your form structure using a JSON schema.
- Each field in the schema specifies its type, path, and parameters.
- The generator builds the form and handles default values, validation, and grouping automatically.

//TODO To simplify the work on creating a schema, you can use the ui-generator-builder + section that describes the builder.

## Top-Level Structure

The schema is organized by **topologies** - different form configurations for different use cases. Each topology defines its own set of sections and their order.

```yaml
replica:
  sections: { ... }
  sectionsOrder:
    - basicInfo
    - resources
sharded:
  sections: { ... }
  sectionsOrder:
    - basicInfo
    - resources
```

### Topology

A **topology** is a top-level key representing a specific form configuration. Each topology contains:

- **`sections`**: An object where each key is a section containing form components
- **`sectionsOrder`** (optional): An array defining the order in which sections should be displayed

example for psmdb operator:

```yaml
replica:
  sections:
    basicInfo: { ... }
    resources: { ... }
  sectionsOrder:
    - basicInfo
    - resources
```

### Sections

**Sections** are logical groupings of form fields, typically representing steps in a multi-step form. These sections also describe exactly how the data will be arranged on the db overview page. Each section can have:

- **`label`** (optional): Display name for the section
- **`description`** (optional): Description text for the section
- **`components`**: An object containing Component or ComponentGroup definitions
- **`componentsOrder`** (optional): An array specifying the order of components

Example:

```yaml
basicInfo:
  label: Basic Information
  description: Provide the basic information for your new database.
  components:
    version: { ... }
    nodes: { ... }
  componentsOrder:
    - version
    - nodes
```

//TODO if you don't want to use multiSteps you can put everything what you need into the one form

### Components

**Components** are the building blocks of the form. The `components` object contains key-value pairs where:

- **Key**: A unique identifier for the component (used for internal references)
- **Value**: Either a single field or a group of nested fields or groups.

### Component vs ComponentGroup

#### Component (Single Field)

A **Component** represents a single form field with the following properties:

- **`uiType`**: Type of UI control (`'number'`, `'select'`, `'hidden'`)
- **`path`** OR **`id`**: The data path in the resulting form values (e.g., `"spec.replica.nodes"`)
- **`fieldParams`**: Configuration for the field (label, placeholder, defaultValue, etc.). Supports `modes` for documented per-mode overrides of shared field params
- **`modes`** (optional): Per-mode component-level overrides (e.g. `uiType: hidden`)
- **`validation`** (optional): Validation rules (min, max, etc.). Supports `modes` for per-mode overrides
- **`techPreview`** (optional): Flag to indicate if the field is in technical preview

Example:

```yaml
numberOfnodes:
  uiType: number
  path: spec.replica.nodes
  fieldParams:
    label: Number of nodes
    defaultValue: 3
  validation:
    min: 1
    max: 7
```

#### ComponentGroup (Nested Fields)

A **ComponentGroup** allows you to group multiple components together with custom layout:

- **`uiType`**: Must be `'group'` or `'hidden'`

//TODO If the uiType is hidden, the component will not be displayed on the UI and, as a result, will not participate in generating data for the api.

- **`groupType`** (optional). For a detailed description of the type of groups and their use, see the [Groups](#groups) section.
- **`label`** (optional): Display label for the group.
- **`description`** (optional): Description text for the group

The label and description display format may look different for different groups. A detailed description can be found in the [Groups](#groups) section.

- **`components`**: Nested components (can include other groups)
- **`componentsOrder`** (optional): Order of nested components
- **`groupParams`** (optional): Additional configuration for the group

Example:

```yaml
resources:
  uiType: group
  groupType: line
  label: Resources
  components:
    cpu:
      uiType: number
      path: spec.resources.cpu
      fieldParams:
        label: CPU
        defaultValue: 1
    memory:
      uiType: number
      path: spec.resources.memory
      fieldParams:
        label: Memory (GB)
        defaultValue: 2
  componentsOrder:
    - cpu
    - memory
```

## Mode-Aware Overrides

### FieldParams-level modes

Use `fieldParams.modes` when the override belongs to the field presentation layer.

For now, the documented and supported `fieldParams.modes` overrides are:

- `disabled`
- `readOnly`
- `label`
- `helperText`
- `defaultValue`
- `autoFocus`

```yaml
dbName:
  uiType: text
  path: metadata.name
  fieldParams:
    label: Database name
    helperText: Must be unique
    modes:
      edit:
        disabled: true
        helperText: Name cannot be changed after creation
```

### Validation-level modes

Use `validation.modes` when the override belongs to validation logic.
Validation-specific mode-aware behavior is documented in [validation.md](validation.md#mode-aware-validation).

## Advanced Properties

### Path vs ID

Each component must have either a `path` or an `id` property (but not both):

- **`path`**: Dot-notation string representing where the value should be stored in the form data
  - Example: `"spec.replica.nodes"` → `{ spec: { replica: { nodes: value } } }`
- **`id`**: Custom identifier used when you don’t want to include field data in the final API request, but need it for validation or conditional rendering.

### Components Order

Both sections and groups can specify the order of their child elements using the `Order` suffix:

- **`sectionsOrder`**: Array of section keys defining section order
- **`componentsOrder`**: Array of component keys defining component order within a section or group

If not specified, the order is determined by the object key insertion order. If only a few sections are ordered, they will be ordered and displayed first. The remaining sections/components will be displayed next by the object key insertion order.

**Example:**

```yaml
sections:
  basicInfo: { ... }
  resources: { ... }
  advanced: { ... }
sectionsOrder:
  - basicInfo
  - resources
  - advanced
```

The next is also valid:

```yaml
sectionsOrder:
  - resources
  - advanced
```

### CEL Condition Rendering

//TODO

## Complete Example

//TODO update after release of psbdb-provider

```yaml
uiSchema:
  replicaSet:
    sections:
      databaseVersion:
        label: "Database Version"
        description: "Provide the information about the database version you want to use."
        components:
          version:
            uiType: select
            path: "spec.engine.version"
            fieldParams:
              label: "Database Version"
              optionsPath: "spec.componentTypes.mongod.versions"
              optionsPathConfig:
                labelPath: "version"
                valuePath: "version"
            validation:
              required: true
      resources:
        label: "Resources"
        description: "Configure the resources your new database will have access to."
        components:
          nodes:
            uiType: group
            components:
              numberOfnodes:
                path: "spec.components.engine.replicas"
                uiType: number
                fieldParams:
                  label: "Number of nodes"
                  defaultValue: 3
                validation:
                  required: true
                  min: 1
                  int: true
                  celExpressions:
                    - celExpr: "spec.components.engine.replicas % 2 == 1"
                      message: "The number of nodes must be odd"

              resources:
                uiType: group
                groupType: line
                components:
                  cpu:
                    path: "spec.components.engine.resources.limits.cpu"
                    uiType: number
                    fieldParams:
                      label: "CPU"
                      defaultValue: 1
                      step: 0.1
                    validation:
                      min: 0.6
                      required: true

                  memory:
                    path: "spec.components.engine.resources.limits.memory"
                    uiType: number
                    fieldParams:
                      label: "Memory"
                      defaultValue: 4
                      step: 0.001
                      badge: "Gi"
                      badgeToApi: true
                    validation:
                      min: 0.512
                      required: true

                  disk:
                    path: "spec.components.engine.storage.size"
                    uiType: number
                    fieldParams:
                      label: "Disk"
                      defaultValue: 25
                      badge: "Gi"
                      badgeToApi: true
                    validation:
                      min: 1
                      int: true
                      required: true
      advanced:
        label: "Advanced configuration"
        description: "Configure advanced settings for your database"
        components:
          storageClass:
            uiType: select
            path: spec.components.engine.storage.storageClass
            fieldParams:
              label: "Storage class"
              defaultValue: "local-path"
              options:
                - label: "local-path"
                  value: "local-path"
          configuration:
            uiType: text
            path: spec.components.engine.configuration
            fieldParams:
              label: "Advanced configuration"
              multiline: true
              minRows: 3
              maxRows: 8
        componentsOrder:
          - storageClass
          - configuration
    sectionsOrder:
      - databaseVersion
      - resources
      - advanced
  sharded:
    sections:
      databaseVersion:
        label: "Database Version"
        description: "Provide the information about the database version you want to use."
        components:
          version:
            uiType: select
            path: "spec.engine.version"
            fieldParams:
              label: "Database Version"
              optionsPath: "spec.componentTypes.mongod.versions"
              optionsPathConfig:
                labelPath: "version"
                valuePath: "version"
            validation:
              required: true
      resources:
        label: "Resources"
        description: "Some description about resources"
        components:
          shards:
            uiType: number
            path: "spec.sharding.shards"
            fieldParams:
              required: true
              label: "Nº of shards"
              defaultValue: 1
          numberOfnodes:
            path: "spec.components.engine.replicas"
            uiType: number
            fieldParams:
              label: "Number of nodes"
              defaultValue: 3
            validation:
              required: true
              min: 1
              int: true
              celExpressions:
                - celExpr: "spec.components.engine.replicas % 2 == 1"
                  message: "The number of nodes must be odd"
          nodesResources:
            uiType: group
            groupType: line
            components:
              cpu:
                path: "spec.components.engine.resources.limits.cpu"
                uiType: number
                fieldParams:
                  label: "CPU"
                  defaultValue: 1
                  step: 0.1
                validation:
                  min: 0.6
                  required: true
              memory:
                path: "spec.components.engine.resources.limits.memory"
                uiType: number
                fieldParams:
                  label: "Memory"
                  defaultValue: 4
                  step: 0.001
                  badge: "Gi"
                  badgeToApi: true
                validation:
                  min: 0.512
                  required: true
              disk:
                path: "spec.components.engine.storage.size"
                uiType: number
                fieldParams:
                  label: "Disk"
                  defaultValue: 25
                  badge: "Gi"
                  badgeToApi: true
                  validation:
                    min: 1
                    int: true
                    required: true
          numberOfRouters:
            path: "spec.components.proxy.replicas"
            uiType: number
            fieldParams:
              label: "Number of routers"
          routersResources:
            uiType: group
            groupType: line
            components:
              cpu:
                path: "spec.components.proxy.resources.limits.cpu"
                uiType: number
                fieldParams:
                  label: "CPU"
                validation:
                  min: 1
                  max: 10
              memory:
                path: "spec.components.proxy.resources.limits.memory"
                uiType: number
                fieldParams:
                  label: "Memory"
                validation:
                  min: 1
                  max: 10
          numberOfConfigServers:
            uiType: number
            path: "spec.components.configServer.replicas"
            fieldParams:
              label: "Nº of configuration servers"
              defaultValue: 3
            validation:
              celExpressions:
                - celExpr: "!(spec.components.engine.replicas > 1 && spec.components.configServer.replicas == 1)"
                  message: "The number of configuration servers cannot be 1 if the number of database nodes is greater than 1"
        componentsOrder:
          - shards
          - numberOfnodes
          - nodesResources
          - numberOfRouters
          - routersResources
          - numberOfConfigServers
      advanced:
        label: "Advanced configuration"
        description: "Configure advanced settings for your database"
        components:
          storageClass:
            uiType: select
            path: spec.components.engine.storage.storageClass
            fieldParams:
              label: "Storage class"
              defaultValue: "local-path"
              options:
                - label: "local-path"
                  value: "local-path"
          configuration:
            uiType: text
            path: spec.components.engine.configuration
            fieldParams:
              label: "Advanced configuration"
              defaultValue: "operationProfiling:\n  mode: slowOp\n"
              multiline: true
              minRows: 3
              maxRows: 8
        componentsOrder:
          - storageClass
          - configuration
    sectionsOrder:
      - databaseVersion
      - resources
      - advanced
```
