# UI Generator — Schema Reference (props tree)

> Schema props are defined in `ui-generator.types.ts`.
> **Legend:** no marker means implemented; `🛠️` means planned / not implemented yet. In flowcharts, non-implemented nodes also use a dashed border.

Read the diagrams top-down: topology → sections → components (leaf fields) / groups (containers) → their params.

## Overview Map

```mermaid
flowchart TD
  T["TopologyUISchemas"] --> TP["Topology<br/>(by topology name)"]
  TP --> SO["sectionsOrder?"]
  TP --> S["sections"]
  S --> SEC["Section<br/>(by section name)"]
  SEC --> SM["label? / description?"]
  SEC --> CO["componentsOrder?"]
  SEC --> C["components"]
  C --> CMP["Component<br/>(leaf field)"]
  C --> GRP["ComponentGroup<br/>(container)"]
  GRP --> C

  CMP --> FP["fieldParams"]
  CMP --> VAL["validation?"]
  CMP --> PID["path | id"]
  CMP --> DS["dataSource?"]
  CMP --> MOD["modes?"]
```

## Level 2 — ComponentGroup (container)

```mermaid
flowchart TD
  G["ComponentGroup"] --> U["uiType<br/>(group, hidden)"]
  G --> LD["label? / description?"]
  G --> GT["groupType?<br/>(accordion, line, bordered🛠️, collapsible🛠️, toggleable🛠️)"]
  G --> GP["groupParams?<br/>(Record&lt;string, unknown&gt;, GroupParams🛠️)"]
  G --> CH["components / componentsOrder?"]
  CH --> GATE["child toggle: gate? 🛠️"]

  classDef planned stroke-dasharray:5 4,stroke:#6c8ebf,stroke-width:1.5px;
  class GATE planned;
```

`gate` 🛠️ is a flag on a child `toggle` (part of the group-kernel proposal, tracked separately).

## Level 2 — Component (leaf field)

```mermaid
flowchart TD
  CMP["Component"] --> UT["uiType<br/>(number, select, text, toggle, hidden)"]
  CMP --> PID{"path | id"}
  CMP --> TP2["techPreview? 🛠️"]
  CMP --> DS2["dataSource?"]
  DS2 --> PROVIDER["provider<br/>(monitoringConfigs, storageClasses)"]
  CMP --> MOD2["modes?"]
  MOD2 --> FORM_MODE["FormMode<br/>(new, edit, restore, import🛠️)"]
  CMP --> FP2["fieldParams"]
  CMP --> VAL2["validation?"]

  classDef planned stroke-dasharray:5 4,stroke:#6c8ebf,stroke-width:1.5px;
  class TP2 planned;
```

- **`path` | `id`** — `path` (`string` | `string[]`) writes to the API; an array is a multi-path (`[0]` is the source, all entries are targets). `id` has no API binding and is used only for validation / CEL.
- **`dataSource.provider`** — a key in the open runtime registry (`register()`), not a closed enum.
- **`modes?`** — `{ [FormMode]: { uiType? } }`; `import` is declared in the enum but is not used by the code.

## Level 3 — fieldParams (by field type)

```mermaid
classDiagram
  class CommonFieldParams {
    label?
    defaultValue?
    disabled?
    autoFocus?
    helperText?
    tooltip?
    badge?
    badgeToApi?
    modes?
  }
  class NumberFieldParams {
    step?
    placeholder?
  }
  class TextFieldParams {
    placeholder?
    multiline?
    rows?
    minRows?
    maxRows?
    type?
    readOnly?
    variant?
    color?
    fullWidth?
    hiddenLabel?
    margin?
  }
  class TextType {
    <<enumeration>>
    text
    password
    email
    search
    tel
  }
  class SelectFieldParams {
    multiple?
    displayEmpty?
    defaultOpen?
    readOnly?
  }
  class SelectStaticOptions {
    options?
  }
  class SelectDynamicOptions {
    optionsPath?
    optionsPathConfig?
  }
  class ToggleFieldParams {
    <<CommonFieldParams>>
  }
  class HiddenFieldParams {
    <<CommonFieldParams>>
  }
  CommonFieldParams <|-- NumberFieldParams
  CommonFieldParams <|-- TextFieldParams
  CommonFieldParams <|-- SelectFieldParams
  CommonFieldParams <|-- ToggleFieldParams
  CommonFieldParams <|-- HiddenFieldParams
  TextFieldParams --> TextType : type
  SelectFieldParams <|-- SelectStaticOptions
  SelectFieldParams <|-- SelectDynamicOptions
  note for SelectFieldParams "options XOR optionsPath + optionsPathConfig"
```

## Level 3 — validation (by field type, mode-aware)

```mermaid
classDiagram
  class CommonValidation {
    required?
    regex?
    celExpressions?
    modes?
  }
  class NumberValidation {
    min?
    max?
    gt?
    lt?
    int?
    multipleOf?
    safe?
  }
  class TextValidation {
    min?
    max?
    length?
    email?
    url?
    uuid?
    trim?
    toLowerCase?
    toUpperCase?
  }
  class SelectValidation {
    <<CommonValidation>>
  }
  class ToggleValidation {
    regex?
    celExpressions?
    modes?
    required? never
  }
  class HiddenValidation {
    <<CommonValidation>>
  }
  CommonValidation <|-- NumberValidation
  CommonValidation <|-- TextValidation
  CommonValidation <|-- SelectValidation
  CommonValidation <|-- ToggleValidation
  CommonValidation <|-- HiddenValidation
```

- **modes (mode-aware)** — `{ [FormMode]: <same rules> + inheritShared? }`.
- **`toggle`** — `required` is always optional (`never`).
- **`select` / `hidden`** — common rules only.

## Cross-Cutting Concepts

| Concept                        | Meaning                                                                                                                                                                                                                          | Status      |
| ------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------- |
| **FormMode**                   | `new · edit · restore · import` controls `modes` at three levels: component `uiType`, `fieldParams`, and `validation`                                                                                                            | implemented |
| **multi-path**                 | `path: [a, b]` writes the same value to all targets                                                                                                                                                                              | implemented |
| **dataSource / API providers** | `dataSource: { provider }` names an API-backed option source; preprocess dev-validates the provider key, and at runtime `DataSourceField` loads the options through the registry (`DataSourcePrefetcher` sets defaults on mount) | implemented |
| **CEL validation**             | Cross-field validation rules declared via `celExpressions` (with an `original` namespace available in edit mode)                                                                                                                 | implemented |
| **CEL conditional rendering**  | Show / hide fields based on another field value through a generic mechanism                                                                                                                                                      | 🛠️          |
| **group kernel**               | `groupType: bordered/collapsible/toggleable` + `gate` + `direction`                                                                                                                                                              | 🛠️          |

## To Consider

- **`fieldParams.badge` / `badgeToApi`** — currently inherited by all field types through `CommonFieldParams`; visual badge rendering is supported for `number` and `select`, while `text` / `toggle` / `hidden` have asymmetric behavior.

## Metadata

- Owner: UI
- Status: current
- Last updated: 2026-09-02
