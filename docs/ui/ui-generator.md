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
- [Field Types](#field-types)
  - [NumberField](#numberfield)
  - [SelectField](#selectfield)
  - [HiddenField](#hiddenfield)
- [Groups](#groups)
  - [Line Group](#line-group)
  - [Accordion Group](#accordion-group)
- [Validation](#validation)
  - [Default Validation](#default-validation)
  - [Schema Custom Validation](#schema-custom-validation)
  - [Common Validation Rules](#common-validation-rules)
    - [Required Field Validation](#required-field-validation)
    - [Regex Validation](#regex-validation)
  - [CEL Expression Validation](#cel-expression-validation)
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

```json
{
  "replica": {
    "sections": { ... },
    "sectionsOrder": ["basicInfo", "resources"]
  },
  "sharded": {
    "sections": { ... },
    "sectionsOrder": ["basicInfo", "resources"]
  }
}
```

### Topology

A **topology** is a top-level key representing a specific form configuration. Each topology contains:

- **`sections`**: An object where each key is a section containing form components
- **`sectionsOrder`** (optional): An array defining the order in which sections should be displayed

example for psmdb operator:

```json
{
  "replica": {
    "sections": {
      "basicInfo": { ... },
      "resources": { ... }
    },
    "sectionsOrder": ["basicInfo", "resources"]
  }
}
```

### Sections

**Sections** are logical groupings of form fields, typically representing steps in a multi-step form. These sections also describe exactly how the data will be arranged on the db overview page. Each section can have:

- **`label`** (optional): Display name for the section
- **`description`** (optional): Description text for the section
- **`components`**: An object containing Component or ComponentGroup definitions
- **`componentsOrder`** (optional): An array specifying the order of components

Example:

```json
"basicInfo": {
  "label": "Basic Information",
  "description": "Provide the basic information for your new database.",
  "components": {
    "version": { ... },
    "nodes": { ... }
  },
  "componentsOrder": ["version", "nodes"]
}
```

//TODO if you don't want to use multiSteps you can put everything what you need into the one form

### Components

**Components** are the building blocks of the form. The `components` object contains key-value pairs where:

- **Key**: A unique identifier for the component (used for internal references)
- **Value**: Either a single field or a group of nested fields or groups.

## Component vs ComponentGroup

### Component (Single Field)

A **Component** represents a single form field with the following properties:

- **`uiType`**: Type of UI control (`'number'`, `'select'`, `'hidden'`)
- **`path`** OR **`id`**: The data path in the resulting form values (e.g., `"spec.replica.nodes"`)
- **`fieldParams`**: Configuration for the field (label, placeholder, defaultValue, etc.)
- **`validation`** (optional): Validation rules (min, max, etc.)
- **`techPreview`** (optional): Flag to indicate if the field is in technical preview

Example:

```json
"numberOfnodes": {
  "uiType": "number",
  "path": "spec.replica.nodes",
  "fieldParams": {
    "label": "Number of nodes",
    "defaultValue": 3
  },
  "validation": {
    "min": 1,
    "max": 7
  }
}
```

### ComponentGroup (Nested Fields)

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

```json
"resources": {
  "uiType": "group",
  "groupType": "line",
  "label": "Resources",
  "components": {
    "cpu": {
      "uiType": "number",
      "path": "spec.resources.cpu",
      "fieldParams": {
        "label": "CPU",
        "defaultValue": 1
      }
    },
    "memory": {
      "uiType": "number",
      "path": "spec.resources.memory",
      "fieldParams": {
        "label": "Memory (GB)",
        "defaultValue": 2
      }
    }
  },
  "componentsOrder": ["cpu", "memory"]
}
```

## Field Types

### NumberField

A numeric input field for integer and decimal values.

**Properties:**

- `uiType`: `"number"` (required)
- `path` OR `id`: Data path or unique identifier (required)
- `fieldParams`: Configuration object with the following optional properties:
  - `label`: Display label for the field
  - `placeholder`: Placeholder text shown when field is empty
  - `defaultValue`: Default numeric value
  - `required`: Whether the field is required (default: `false`)
  - `disabled`: Whether the field is disabled (default: `false`)
  - `autoFocus`: Automatically focus this field on render
  - `helperText`: Help text displayed below the field
  - `step`: Increment/decrement step for arrow buttons (e.g., `0.1`, `5`, `10`)
- `validation` (optional): Validation rules object with the following properties:
  - `min`: Minimum value (inclusive) - value must be >= specified number
  - `max`: Maximum value (inclusive) - value must be <= specified number
  - `gt`: Greater than (exclusive) - value must be > specified number
  - `lt`: Less than (exclusive) - value must be < specified number
  - `int`: Must be an integer (boolean: `true`)
  - `multipleOf`: Value must be a multiple of specified number
  - `safe`: Must be a safe integer within JavaScript's safe integer range (boolean: `true`)
  - `celExpressions`: Array of CEL validation expressions for cross-field validation

**Native Validation:** Validates that input is numeric

**Validation Auto-Mapping:** The following validation rules are automatically applied to HTML input attributes for browser-level validation:

- `validation.min` → HTML `min` attribute (inclusive lower bound)
- `validation.max` → HTML `max` attribute (inclusive upper bound)
- `validation.gt` → HTML `min` attribute (converted to exclusive lower bound)
- `validation.lt` → HTML `max` attribute (converted to exclusive upper bound)

When converting exclusive bounds (`gt`/`lt`) to HTML attributes:

- For integer validation (`int: true`): offset by 1 (e.g., `gt: 5` becomes `min="6"`)
- With `step` defined: offset by step value (e.g., `gt: 5` with `step: 0.5` becomes `min="5.5"`)
- For arbitrary decimals: offset by 0.000001 (e.g., `gt: 5` becomes `min="5.000001"`)
- Explicit `min`/`max` always take priority over converted `gt`/`lt`

**Examples:**

[OpenEverest TextInput “Number type” Story](https://openeverest.io/openeverest/?path=/story/textinput--number-type)

**Basic Number Field**

```json
"replicas": {
  "uiType": "number",
  "path": "spec.replicas",
  "fieldParams": {
    "label": "Number of Replicas",
    "defaultValue": 3,
    "step": 1,
    "autoFocus": true
  },
  "validation": {
    "min": 1,
    "max": 16,
    "int": true
  }
}
```

**Exclusive Bounds**

```json
  "validation": {
    "gt": 0,
    "lt": 32,
    "multipleOf": 0.5
  }
```

**Note:** Fields are optional by default. Validation rules (min/max/etc.) only apply when a value is entered. To make a field required, set `required: true` in `fieldParams`.

### SelectField

A dropdown selection field.

**Properties:**

- `uiType`: `"select"`
- `fieldParams`:
  - `label`: Display label
  - `options`: Array of `{ label: string, value: string }` objects
  - `defaultValue`: Default selected value

**Example:**

```json
"version": {
  "uiType": "select",
  "path": "spec.engine.version",
  "fieldParams": {
    "label": "Database Version",
    "options": [
      {
        "label": "MongoDB 6.0.19-16",
        "value": "6.0.19-16"
      },
      {
        "label": "MongoDB 7.0.18-11",
        "value": "7.0.18-11"
      }
    ],
    "defaultValue": "6.0.19-16"
  }
}
```

**Native Validation:** Validate that selected value is in the options list

### HiddenField

`uiType`: `"hidden"`

Fields hidden in the schema are not displayed on the UI and do not participate in generating data for the API request.

//TODO example of hidden field

If you need to use the default value in the form that the user cannot change, use the disabled parameter.

//TODO example of disabled field

## Groups

Groups allow you to organize multiple fields together with different layout options.

### Line Group

//TODO will be renamed, documentation should be checked before merging
Displays components in a horizontal line (flex layout).

```json
"resourceGroup": {
  "uiType": "group",
  "groupType": "line",
  "label": "Resources",
  "components": {
    "cpu": { ... },
    "memory": { ... },
    "disk": { ... }
  },
  "componentsOrder": ["cpu", "memory", "disk"]
}
```

//TODO visual example

### Accordion Group

Displays components in a collapsible accordion panel.

```json
"advancedSettings": {
  "uiType": "group",
  "groupType": "accordion",
  "label": "Advanced Settings",
  "description": "Optional advanced configuration",
  "components": {
    "setting1": { ... },
    "setting2": { ... }
  }
}
```

//TODO visual example

## Validation

### Default Validation

Each field type has built-in validation based on its type:

- **Number fields**: Validate that input is numeric
- **Select fields**: Validate that selected value is in the options list

### Schema Custom Validation

Custom validation rules can be defined in the `validation` property. These have higher priority than default validation and will override defaults if the same properties are specified.

### Common Validation Rules

The following validation rules are supported for **all field types**:

**Required Field Validation**

Control whether a field must have a value using the `required` parameter in `fieldParams`:

```json
"fieldParams": {
  "required": true  // Makes the field required (default is false)
}
```

**Regex Validation**

Apply regular expression validation to any field using the `regex` property in the `validation` object:

| Property  | Type              | Description                                           |
| --------- | ----------------- | ----------------------------------------------------- |
| `pattern` | string            | Regular expression pattern (without delimiters)       |
| `message` | string (optional) | Custom error message to display on validation failure |

**Number field with regex:**

```json
"portNumber": {
  "uiType": "number",
  "path": "spec.port",
  "fieldParams": {
    "label": "Port Number"
  },
  "validation": {
    "regex": {
      "pattern": "^[1-9][0-9]{3,4}$",
      "message": "Port must be between 1000-99999"
    }
  }
}
```

### CEL Expression Validation

CEL (Common Expression Language) validation allows you to define cross-field validation rules using CEL expressions. These expressions can reference multiple fields and return `true` when validation passes or `false` when it fails.

**Important:** CEL expressions should return `true` for valid data and `false` for invalid data.

**Properties:**

| Property  | Type   | Description                                    |
| --------- | ------ | ---------------------------------------------- |
| `celExpr` | string | CEL expression that returns boolean            |
| `message` | string | Error message to display when validation fails |

**Example:**

```json
{
  "numberOfConfigServers": {
    "uiType": "number",
    "path": "spec.sharding.configServer.replicas",
    "fieldParams": {
      "label": "Number of configuration servers",
      "defaultValue": 3
    },
    "validation": {
      "celExpressions": [
        {
          "celExpr": "!(spec.replica.nodes > 1 && spec.sharding.configServer.replicas == 1)",
          "message": "The number of configuration servers cannot be 1 if the number of database nodes is greater than 1"
        }
      ]
    }
  }
}
```

In this example, the validation fails (returns false) when there are more than 1 database nodes AND the number of config servers is 1. The `!` operator negates the condition so it returns `false` when the invalid condition is true.

**Note:** All validation rules only apply when a value is entered. Empty fields will pass validation by default since fields are optional unless explicitly marked as `required: true`.

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

```json
{
  "sections": {
    "basicInfo": { ... },
    "resources": { ... },
    "advanced": { ... }
  },
  "sectionsOrder": ["basicInfo", "resources", "advanced"]
}
```

The next is also valid:

```json
  "sectionsOrder": ["resources", "advanced"]
```

### CEL Condition Rendering

//TODO

## Complete Example

//TODO some lines about this example

```json
{
  "replica": {
    "sections": {
      "basicInfo": {
        "label": "Basic Information",
        "description": "Provide the basic information for your new database.",
        "components": {
          "version": {
            "uiType": "select",
            "path": "spec.engine.version",
            "fieldParams": {
              "label": "Database Version",
              "options": [
                { "label": "MongoDB 6.0.19-16", "value": "6.0.19-16" }
              ],
              "defaultValue": "6.0.19-16"
            }
          }
        }
      },
      "resources": {
        "label": "Resources",
        "description": "Configure the resources for your database.",
        "components": {
          "numberOfnodes": {
            "uiType": "number",
            "path": "spec.replica.nodes",
            "fieldParams": {
              "label": "Number of nodes",
              "defaultValue": 3
            },
            "validation": {
              "min": 1,
              "max": 7
            }
          },
          "resourceGroup": {
            "uiType": "group",
            "groupType": "line",
            "components": {
              "cpu": {
                "uiType": "number",
                "path": "spec.resources.cpu",
                "fieldParams": {
                  "label": "CPU",
                  "badge": "cores",
                  "defaultValue": 1
                },
                "validation": {
                  "min": 0.6,
                  "max": 16
                }
              },
              "memory": {
                "uiType": "number",
                "path": "spec.resources.memory",
                "fieldParams": {
                  "label": "Memory",
                  "badge": "GB",
                  "defaultValue": 2
                },
                "validation": {
                  "min": 1,
                  "max": 128
                }
              }
            },
            "componentsOrder": ["cpu", "memory"]
          }
        },
        "componentsOrder": ["numberOfnodes", "resourceGroup"]
      }
    },
    "sectionsOrder": ["basicInfo", "resources"]
  }
}
```
