# Validation

## Table of Contents

- [Default Validation](#default-validation)
- [Schema Custom Validation](#schema-custom-validation)
- [Common Validation Rules](#common-validation-rules)
  - [Required](#required)
  - [Regex](#regex)
- [Mode-Aware Validation](#mode-aware-validation)
- [Class Parameters Schema](#class-parameters-schema)
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

**Note:** The rules other than `celExpressions` only apply when a value is entered: an empty field passes them unless it is marked `required: true`. A CEL rule runs for an empty field too, see [Optional Fields and `has()`](#optional-fields-and-has).

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

### Disabled Fields

A field that is disabled, always or in the current mode (`fieldParams.modes.<mode>.disabled`), skips its own rules, including its `celExpressions`: the user can't change its value. Rules on other fields still read its value.

So put a rule that links two fields on the one the user can edit. For example, if `version` is disabled in `restore`, a rule on `replicas` that reads `spec.version` still runs there, and its error shows under `replicas`.

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

### Optional Fields and `has()`

A rule may run while a field it reads has no value. Such a field is then either **unset** or
holds its **empty value**:

- an empty text or select field is unset or `''`, and a toggle that is off is unset or `false`.
  Which one depends on the field's own `validation`, on the form mode and on whether the instance
  was saved with it, so allow both;
- an empty number, quantity or [map](components/map-field.md) field is unset in every mode;
- every field of a [toggleable group](groups.md#toggleable-group) that is off is unset.

Reading an unset field fails with a CEL error: the form can't be submitted, and the user sees
`CEL expression error: …` in place of your `message`. So a rule that reads a field which may have
no value asks for it with `has()` first, and for a text or select field allows the empty string
too:

```yaml
celExpressions:
  - celExpr: "!has(spec.replicas) || spec.replicas % 2 == 1"
    message: Use an odd number of nodes
  - celExpr: "!has(spec.note) || spec.note == '' || size(spec.note) <= 80"
    message: Keep the note under 80 characters
```

- `has()` of a nested path is false when an object above it is missing, so
  `has(spec.components.proxy.service.type)` needs no check of `spec.components.proxy` before it.
  The first segment is the exception: `has()` can't ask for it, and a rule that reads a path
  under a missing one fails. `spec` is there in practice. Another one, such as `config`, is
  missing in `edit` when the instance holds no value for any field under it.
- The same goes for a path under `original`: the saved instance may not hold the field.
  `original` itself exists only in `edit`, where the user changes a saved instance, and `has()`
  can't ask for it. So keep a rule that reads `original` under `validation.modes.edit`, as in the
  example above.
- Try the rule in a new form and when editing an instance saved without the field: a rule
  without `has()` can pass in the first and fail in the second.

### Reading a Map Field

A rule reads a [map field](components/map-field.md) as the map it writes, `{ key: value }`, at
the map's `path`. The rule may sit on another field or on the map itself; its error shows under
the field it sits on. A map with no entries writes nothing, so the rule finds it unset: ask for
it with `has()` first, as for any [optional field](#optional-fields-and-has). The same goes for
the saved map at `original.<path>`.

| To check                              | Write                                                                                                 |
| ------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| there are no entries                  | `!has(spec.labels)`                                                                                   |
| a key is set                          | `has(spec.labels) && 'team' in spec.labels`                                                           |
| the value of a key                    | `has(spec.labels) && 'team' in spec.labels && spec.labels['team'] == 'db'`                            |
| the number of entries                 | `!has(spec.labels) \|\| size(spec.labels) <= 5`                                                       |
| every key                             | `!has(spec.labels) \|\| spec.labels.all(k, k.startsWith('example.com/'))`                             |
| every value                           | `!has(spec.labels) \|\| spec.labels.all(k, spec.labels[k] != '')`                                     |
| no saved entry is removed (edit mode) | `!has(original.spec.labels) \|\| (has(spec.labels) && original.spec.labels.all(k, k in spec.labels))` |

- **A key that isn't set.** `spec.labels['team']` fails with a CEL error instead of your
  `message`, so check the key with `in` first, as in the table.
- **Rows the user hasn't finished.** A row with no key isn't an entry, so the rule doesn't see it.

In the section edit dialog the rule runs again whenever the map's rows change. A rule on the map
itself does so in every form.

```yaml
externalAccess:
  uiType: select
  path: spec.components.proxy.service.type
  fieldParams:
    label: External access
    options:
      - { label: ClusterIP, value: ClusterIP }
      - { label: LoadBalancer, value: LoadBalancer }
  validation:
    celExpressions:
      - celExpr: >-
          !has(spec.components.proxy.service.type) ||
          spec.components.proxy.service.type != 'LoadBalancer' ||
          (has(spec.components.proxy.service.annotations) &&
          'service.beta.kubernetes.io/aws-load-balancer-type' in spec.components.proxy.service.annotations)
        message: Add the aws-load-balancer-type annotation for a LoadBalancer
```

A rule on a row's key or value isn't supported yet: check the entries from the map with
`.all(k, …)`, see [A CEL Rule on the Map](components/map-field.md#a-cel-rule-on-the-map).

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

## Class Parameters Schema

A monitoring class's destination and instance forms write the parameters the API checks against the
class's OpenAPI v3 schema for them (`destinationParametersSchema`, `instanceParametersSchema`). The
UI checks every field of the form against that schema too, so a value the API would refuse shows an
error on the field before saving. Nothing has to be repeated in `validation`:

| Schema keyword            | Checked                                                                                                                                |
| ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `required`                | when every object on the way to the field is required                                                                                  |
| `pattern`                 | as the server's Go regexp reads it, as far as JavaScript can (an escaped punctuation character, such as `\_`, is the character itself) |
| `minLength` / `maxLength` | in characters                                                                                                                          |
| `enum`                    | the value must be one of the listed values                                                                                             |

Map entries are checked against `additionalProperties`. The check runs on the value the API gets:
with `badge: s` and `badgeToApi: true`, a field showing `30` is checked as `30s`.

The server alone checks:

- `anyOf`, `oneOf`, `allOf` and other keywords;
- a `pattern` only Go reads, such as one with `\z` or `[[:alpha:]]`;
- disabled and read-only fields, which the user can't change;
- fields of a switched-off toggleable group;
- what the schema requires that the form has no field for.

A class without a destination form gets a form generated from `destinationParametersSchema`. Its
text fields are checked against the same keywords.
