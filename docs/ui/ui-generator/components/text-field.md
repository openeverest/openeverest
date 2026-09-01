# TextField

## Table of Contents

- [Properties](#properties)
- [Behavior](#behavior)
- [Examples](#examples)
  - [Basic text input (required)](#basic-text-input-required)
  - [Multiline description (textarea)](#multiline-description-textarea)
  - [Password field](#password-field)
  - [Auto-trimming with case transform](#auto-trimming-with-case-transform)
  - [Regexp](#regexp)
  - [CEL](#regexp)

A flexible single-line or multi-line text input field. Suitable for names, descriptions, URLs, email addresses, passwords, and any other free-form text.

> Common `fieldParams` properties (`label`, `defaultValue`, `disabled`, `helperText`, `info`, `tooltip`, `badge`, `modes`) apply to all field types and are documented in [Common Field Parameters](common-field-params.md).

## Properties:

- `uiType`: `"text"` (**Required**)
- `path` OR `id`: Data path or unique identifier (**Required**)
- `fieldParams`: Configuration object with the following optional properties:
  - `label`: Display label for the field
  - `placeholder`: Placeholder text shown when the field is empty
  - `defaultValue`: Default string value (default: `""`)
  - `disabled`: Whether the field is disabled (default: `false`)
  - `autoFocus`: Automatically focus this field on render
  - `helperText`: Help text displayed below the field
  - `multiline`: Render as a `<textarea>` instead of a single-line `<input>` (default: `false`)
  - `rows`: Fixed number of visible text rows (only applies when `multiline: true`)
  - `minRows`: Minimum number of rows; textarea grows as content is added (requires `multiline: true`)
  - `maxRows`: Maximum number of rows before scrolling (requires `multiline: true`)
  - `type`: HTML input type — `"text"` | `"password"` | `"email"` | `"search"` | `"tel"` (default: `"text"`)
  - `readOnly`: Value is visible but cannot be changed by the user (default: `false`)
  - `variant`: MUI TextField visual variant — `"outlined"` | `"filled"` | `"standard"` (default: `"outlined"`)
  - `color`: Color theme for the input — `"primary"` | `"secondary"` | `"error"` | `"info"` | `"success"` | `"warning"` (default: `"primary"`)
  - `fullWidth`: Whether the input takes up the full width of its container (default: `false`)
  - `hiddenLabel`: Hide the floating label (default: `false`)
  - `margin`: Vertical spacing around the field — `"none"` | `"dense"` | `"normal"` (default: `"none"`)
- `validation` (optional): Validation rules object with the following properties:
  - `required`: Whether a non-empty value is required (default: `false`)
  - `min`: Minimum number of characters (inclusive)
  - `max`: Maximum number of characters (inclusive)
  - `length`: Exact number of characters required
  - `email`: Value must be a valid e-mail address (boolean: `true`)
  - `url`: Value must be a valid URL (boolean: `true`)
  - `uuid`: Value must be a valid UUID (boolean: `true`)
  - `trim`: Strip leading/trailing whitespace before submitting (boolean: `true`)
  - `toLowerCase`: Convert the value to lower case before submitting (boolean: `true`)
  - `toUpperCase`: Convert the value to upper case before submitting (boolean: `true`)
  - `regex`: Regular expression the value must match (see [Regex Validation](#regex-validation))
  - `celExpressions`: Array of CEL validation expressions for cross-field validation (see [CEL Expression Validation](#cel-expression-validation))

> **TODO (not yet implemented):** The following Zod string methods are not yet supported in the validation schema builder and will be ignored if specified: `startsWith`, `endsWith`, `includes`, `ip`, `cidr`, `datetime`, `date`, `time`, `duration`, `base64`, `base64url`, `nanoid`, `cuid`, `cuid2`, `ulid`, `emoji`, `jwt`.

## Behavior:

- **Optional fields** (default): Default value is `""` (empty string). An empty value is treated as "no input" and passes all format validators (`email`, `url`, `uuid`, `regex`, etc.). Validation only runs when the user provides a non-empty value.
- **Required fields**: User must provide a non-empty value. Format/length validators are combined with the required constraint.
- **Transforms** (`trim`, `toLowerCase`, `toUpperCase`): Applied before form submission. The UI shows the original typed value; the submitted data contains the transformed value.
- **Multiline fields**: Setting `multiline: true` renders a `<textarea>`. Use `rows` for a fixed height, or `minRows`/`maxRows` for an auto-growing textarea.
- **`type: "password"`**: Input is masked. Combined with `min`/`max` validation it provides a password strength constraint.

## Examples:

### Basic text input (required)

```yaml
username:
  uiType: text
  path: spec.username
  fieldParams:
    label: Username
    placeholder: Enter your username
  validation:
    required: true
    min: 3
    max: 32
```

### Multiline description (textarea)

```yaml
description:
  uiType: text
  path: spec.description
  fieldParams:
    label: Description
    multiline: true
    minRows: 3
    maxRows: 8
    placeholder: "Describe your database cluster…"
  validation:
    max: 500
```

### Password field

```yaml
password:
  uiType: text
  path: spec.password
  fieldParams:
    label: Password
    type: password
  validation:
    required: true
    min: 8
```

### Auto-trimming with case transform

```yaml
tag:
  uiType: text
  path: spec.tag
  fieldParams:
    label: Tag
  validation:
    trim: true
    toLowerCase: true
    max: 50
```

### Regexp

```yaml
clusterName:
  uiType: text
  path: spec.clusterName
  fieldParams:
    label: Cluster Name
  validation:
    required: true
    regex:
      pattern: "^[a-z][a-z0-9-]{2,29}$"
      message: "Must be 3–30 lowercase alphanumeric characters or hyphens, starting with a letter"
```

### CEL

> Documentation coming soon.
