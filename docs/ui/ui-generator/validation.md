# Validation

## Table of Contents

- [Default Validation](#default-validation)
- [Schema Custom Validation](#schema-custom-validation)
- [Common Validation Rules](#common-validation-rules)
  - [Required](#required)
  - [Regex](#regex)
- [Mode-Aware Validation](#mode-aware-validation)
- [Groups](groups.md)

## Default Validation

Each field type has built-in validation based on its type

Example:

- **Number fields**: Validate that input is numeric
- **Select fields**: Validate that selected value is in the options list

More information can be found in **Native Validation** section for each field.

## Schema Custom Validation

Custom validation rules can be defined in the `validation` property. These have higher priority than default validation and will override defaults if the same properties are specified.

## Common Validation Rules

The following validation rules are supported for **all field types**.

### Required

Control whether a field must have a value using the `required` parameter in the `validation` object:

```yaml
validation:
  required: true # (default is false)
```

### Regex

Apply regular expression validation to any field using the `regex` property in the `validation` object:

| Property  | Type              | Description                                           |
| --------- | ----------------- | ----------------------------------------------------- |
| `pattern` | string            | Regular expression pattern (without delimiters)       |
| `message` | string (optional) | Custom error message to display on validation failure |

**Number field with regex:**

```yaml
portNumber:
  uiType: number
  path: spec.port
  fieldParams:
    label: Port Number
  validation:
    regex:
      pattern: "^[1-9][0-9]{3,4}$"
      message: Port must be between 1000-99999
```

More regexp examples can be found in the documentation for a specific field in the **examples => regexp section**

## CEL Expression Validation

CEL (Common Expression Language) validation allows you to define cross-field validation rules using CEL expressions. These expressions can reference multiple fields and return `true` when validation passes or `false` when it fails.

**Important:** CEL expressions should return `true` for valid data and `false` for invalid data.

**Properties:**

| Property  | Type   | Description                                    |
| --------- | ------ | ---------------------------------------------- |
| `celExpr` | string | CEL expression that returns boolean            |
| `message` | string | Error message to display when validation fails |

### Example:

```yaml
numberOfConfigServers:
  uiType: number
  path: spec.sharding.configServer.replicas
  fieldParams:
    label: Number of configuration servers
    defaultValue: 3
  validation:
    celExpressions:
      - celExpr: "!(spec.replica.nodes > 1 && spec.sharding.configServer.replicas == 1)"
        message: The number of configuration servers cannot be 1 if the number of database nodes is greater than 1
```

In this example, the validation fails (returns false) when there are more than 1 database nodes AND the number of config servers is 1. The `!` operator negates the condition so it returns `false` when the invalid condition is true.

**Select field with CEL validation:**

```yaml
tier:
  uiType: select
  path: spec.tier
  fieldParams:
    label: Service Tier
    options:
      - label: Free
        value: free
      - label: Pro
        value: pro
      - label: Enterprise
        value: enterprise
  validation:
    celExpressions:
      - celExpr: "self == 'pro' || self == 'enterprise' || spec.users < 10"
        message: Free tier is limited to 10 users
```

In this example, the `self` keyword refers to the current field's value. The validation passes when the tier is 'pro' or 'enterprise', or when the number of users is less than 10.

**Note:** All validation rules only apply when a value is entered. Empty fields will pass validation by default since fields are optional unless explicitly marked as `required: true`.

More CEL examples can be found in the documentation for a specific field in the **examples => CEL section**

## Mode-Aware Validation

By default, validation rules apply in **all modes** (create, edit, restore, import). When you need different rules per mode — for example, preventing disk descaling only during edit — use the **mode-aware validation** format.

### Structure

```yaml
validation:
  required: true
  min: 1
  celExpressions:
    - celExpr: "spec.replicas % 2 == 1"
      message: Must be odd
  modes:
    edit:
      celExpressions:
        - celExpr: "spec.disk >= original.spec.disk"
          message: Disk cannot be decreased
    new:
      inheritShared: false
      required: true
```

### Properties

| Property        | Type    | Description                                                            |
| --------------- | ------- | ---------------------------------------------------------------------- |
| `modes`         | object  | Per-mode (`new`, `edit`, `restore`, `import`) overrides                |
| `inheritShared` | boolean | Set to `false` in a mode branch to ignore base rules (default: `true`) |

### Merge Semantics

1. **Scalar rules** (`required`, `min`, `max`, `int`, …): mode value **replaces** base value.
2. **`celExpressions`**: mode-specific expressions are **appended** to base ones.
3. **`inheritShared: false`**: base rules are completely ignored; only the mode branch is used.
4. If no mode branch exists for the current mode, only base rules apply.

### CEL Evaluation Context

CEL expressions have access to the following variables:

| Variable          | Description                                                                              |
| ----------------- | ---------------------------------------------------------------------------------------- |
| `<field path>`    | Each field is referenced by its `path` value (e.g., `spec.replicas`, `config.timeout`)   |
| `self`            | Current field value                                                                      |
| `original.<path>` | Original persisted instance data at the same path; available in edit mode for comparison |

The `original` namespace allows write-once or no-descale rules:

```yaml
validation:
  min: 1
  int: true
  modes:
    edit:
      celExpressions:
        - celExpr: "spec.sharding.shards >= original.spec.sharding.shards"
          message: Number of shards cannot be decreased
```

### Full Example: Mongo Node Edit Constraints

```yaml
numberOfNodes:
  path: "spec.components.engine.replicas"
  uiType: number
  fieldParams:
    label: "Number of nodes"
  validation:
    required: true
    min: 1
    int: true
    celExpressions:
      - celExpr: "spec.components.engine.replicas % 2 == 1"
        message: "The number of nodes must be odd"
    modes:
      edit:
        celExpressions:
          - celExpr: >-
              !(spec.components.engine.replicas == 1
              && original.spec.components.engine.replicas > 1)
            message: "Cannot scale down to a single node"
```

In this example:

- **All modes**: nodes must be ≥ 1, integer, and odd.
- **Edit only**: scaling from >1 to 1 is blocked by comparing against `original`.
