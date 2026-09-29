# SelectField

## Table of Contents

- [Properties](#properties)
- [Native Validation](#native-validation)
- [Examples](#examples)
  - [Basic Select Field](#basic-select-field)
  - [Select with Default Value](#select-with-default-value)
  - [Select with displayEmpty (Placeholder)](#select-with-displayempty-placeholder)
  - [Required Select with displayEmpty](#required-select-with-displayempty)
  - [Regexp](#regexp)
  - [CEL](#cel)
- [Behavior](#behavior)

A dropdown selection field that allows users to choose one option from a predefined list of values.

## Properties:

- `uiType`: `"select"` (**Required**)
- `path` OR `id`: Data path or unique identifier (**Required**)
- `fieldParams`: Configuration object with the following properties:
  - `label`: Display label for the field
  - `options`: Array of objects with `{ label: string, value: string }` format (**Required**)
  - `defaultValue`: Default selected value (must match one of the option values)
  - `disabled`: Whether the field is disabled (default: `false`)
  - `autoFocus`: Automatically focus this field on render
  - `helperText`: Help text displayed below the field
  - `multiple`: Allow multi-select (**Note:** While this prop is passed to MUI Select, full support requires array-type validation in the Zod schema builder, array default values handling, and array type definitions - planned for future implementation)
  - `displayEmpty`: Show placeholder/empty option when no value is selected (default: `false`)
  - `defaultOpen`: Open dropdown menu on component mount (default: `false`)
  - `readOnly`: Make field read-only - value displayed but cannot be changed (default: `false`)
- `validation` (optional): Validation rules object with the following properties:
  - `required`: Whether the field is required (default: `false`)
  - `regex`: Regular expression validation (see [Regex Validation](../validation.md#regex))
  - `celExpressions`: Array of CEL validation expressions for cross-field validation (see [CEL Expression Validation](../validation.md#cel-expression-validation))

## Native Validation:

Automatically validates that the selected value is one of the allowed options defined in the `options` array (enum validation).

## Examples:

[OpenEverest Select “Select uiType" Story](https://openeverest.io/openeverest/?path=/story/select--basic)

### Basic Select Field

```yaml
databaseType:
  uiType: select
  path: spec.database.type
  fieldParams:
    label: Database Type
    options:
      - label: MySQL
        value: mysql
      - label: PostgreSQL
        value: postgresql
      - label: MongoDB
        value: mongodb
```

### Select with Default Value

```yaml
region:
  uiType: select
  path: spec.region
  fieldParams:
    label: Region
    defaultValue: us-east-1
    options:
      - label: US East
        value: us-east-1
      - label: US West
        value: us-west-1
      - label: EU Central
        value: eu-central-1
```

### Select with displayEmpty (Placeholder)

```yaml
tier:
  uiType: select
  path: spec.tier
  fieldParams:
    label: Service Tier
    displayEmpty: true
    options:
      - label: Free
        value: free
      - label: Pro
        value: pro
      - label: Enterprise
        value: enterprise
  validation:
    required: false
```

This example demonstrates an optional select field with `displayEmpty: true`. An empty option (value: `""`, label: "None") will be automatically added, allowing users to clear their selection.

### Required Select with displayEmpty

```yaml
environment:
  uiType: select
  path: spec.environment
  fieldParams:
    label: Environment
    displayEmpty: true
    options:
      - label: Development
        value: dev
      - label: Staging
        value: staging
      - label: Production
        value: prod
  validation:
    required: true
```

For required fields, even with `displayEmpty: true`, no empty option is auto-injected since the field must have a value.

When `displayEmpty` is `true` and the field is **optional** (`validation.required` is not `true`), an empty option with value `""` and label "None" is automatically injected at the beginning of the options list. This allows users to clear their selection after choosing a value. If you've already included an empty option in your schema, no duplicate will be added.

**Important:** For **required** fields with `displayEmpty: true`, no empty option is injected since the field must have a value.

### Regexp

```yaml
version:
  uiType: select
  path: spec.version
  fieldParams:
    label: Version
    options:
      - label: v1.0
        value: v1.0
      - label: v2.0
        value: v2.0
      - label: v3.0-rc1
        value: v3.0-rc1
  validation:
    regex:
      pattern: "^v[0-9]+\\.[0-9]+$"
      message: Only stable versions allowed (format vX.Y)
```

### CEL validation

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

## Behavior:

- **Optional fields** (default): Default value is empty string (`''`). Form is valid even when no selection is made. When combined with `displayEmpty: true`, a "None" option is automatically added to allow clearing the selection.
- **Required fields**: User must select a value from the options. Empty string is not allowed. Form validation fails until a value is selected with error message "Field is required".
- **Default values**: If no `defaultValue` is specified, the field starts with an empty selection. The `defaultValue` must match one of the option values; invalid default values will cause form validation to fail.
- **Empty options**: If the `options` array is empty, the select displays a "No options" message and validates as optional string.
- **Auto-injected empty option**: When a field is optional AND `displayEmpty: true`, an empty option (value: `""`, label: "None") is automatically added to the beginning of the options list, unless you've already provided one.

**Note:** For cross-field validation or regex patterns on select values, see the [Validation](../validation.md) section in the main README.
