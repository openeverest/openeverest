# UI Generator Architecture

This document describes the high-level architecture and data flows of the
UI Generator — the schema-driven form engine that powers the database creation
wizard and the section-edit modals on the cluster overview page.

## 1. System Overview

```
┌─────────────────────────────────────────────────────────┐
│                    Provider CRD (K8s)                   │
│  spec.uiSchema: TopologyUISchemas (JSON/YAML)           │
└────────────────────────┬────────────────────────────────┘
                         │
              fetched via REST API
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│              Preprocessing Pipeline                     │
│  1. preprocessSchema                                    │
│  2. applyModeOverrides                                  │
└────────────────────────┬────────────────────────────────┘
                         │
                         │
          ┌──────────────┼──────────────┐
          ▼              ▼              ▼
   ┌────────────┐ ┌────────────┐ ┌──────────────┐
   │ Zod Schema │ │  Defaults  │ │  Rendering   │
   │  Builder   │ │  Builder   │ │  Pipeline    │
   └─────┬──────┘ └─────┬──────┘ └──────┬───────┘
         │               │               │
         ▼               ▼               ▼
   zodResolver    defaultValues     <UIGenerator />
         │               │               │
         └───────┬───────┘               │
                 ▼                       │
          useForm (RHF)  ◄───────────────┘
                 │
          user interaction
                 │
                 ▼
   ┌──────────────────────────┐
   │   Postprocessing         │
   │  (multi-path, badges,    │
   │   empty-value removal)   │
   └────────────┬─────────────┘
                │
                ▼
         API PUT / POST
```

## 2. Schema Structure

//TODO add link to the types files

## 3. Preprocessing Pipeline

```
Raw UISchema (from Provider CRD)
       │
       ▼
 preprocessSchema(schema, provider)
       │
       ├── For each component:
       │     1. withNormalizedPathMeta  →  _normalized.sourcePath / targetPaths
       │     2. resolveSelectOptions   →  optionsPath → static options[]
       │
       ▼
 (Optional) applyModeOverrides(sections, formMode)
       │
       │
       ▼
 Preprocessed sections (ready for rendering + validation)
```

## 4. Validation Pipeline

```
Preprocessed sections
       │
       ▼
 buildZodSchema(schema, topology, options?) ← full wizard
   — OR —
 buildSectionZodSchema(sectionKey, sections, options?) ← edit modal
       │
       ├── For each component:
       │     resolveValidationForMode(validation, formMode)
       │       ├── merge base + modes[formMode]
       │       └── no formMode  → base only
       │     buildShapeFromComponents
       │       ├── ZOD_SCHEMA_MAP[uiType]  →  base Zod type
       │       ├── applyValidationFromSchema  →  min/max/regex/required
       │       └── collect celExpValidations + celDependencyGroups
       │
       ├── convertToNestedSchema (flat → nested z.object)
       │
       └── applyCelValidation(schema, celExprs, originalData?)
              │
              ▼
        z.ZodTypeAny  (passed to zodResolver for react-hook-form)
```

`buildSectionZodSchema` differs from `buildZodSchema` in two ways:

- Zod field rules are built only for the target section.
- The root schema uses `.passthrough()` so non-section fields pass through.
- CEL expressions are still collected from ALL sections for cross-field rules.

## 5. Rendering Pipeline

```
<UIGenerator sectionKey sections providerObject formMode />
       │
       ├── UiGeneratorProvider (context: provider, formMode, loading)
       │
       ├── orderComponents(section.components, componentsOrder)
       │
       └── For each component:
             renderComponent({ item, name })
               │
               ├── group / hidden → recurse into nested components
               │
               └── leaf component:
                     generateFieldId(item, name)  →  form field name
                     <CustomField /> → registered in react-hook-form
```

## 6. Postprocessing Pipeline

```
Form data (from react-hook-form)
       │
       ▼
 postprocessSchemaData(formData, { schema, selectedTopology })
       │
       ├── extractMultiPathMappings  →  fan-out single value to multiple paths
       ├── applyMultiPathMappings    →  set value at all target paths
       ├── extractBadgeMappings      →  append badge suffix (e.g. "8" → "8Gi")
       ├── applyBadgesToFormData
       └── removeEmptyFieldValues    →  prune undefined/null/""
              │
              ▼
        Clean API payload  →  PUT /instances/:ns/:name
```

## 7. Form Modes (FormMode)

| Mode    | Value       | Description                      |
| ------- | ----------- | -------------------------------- |
| Create  | `"new"`     | Fresh database creation wizard   |
| Edit    | `"edit"`    | Section editing on overview page |
| Restore | `"restore"` | Restore from backup wizard       |
| Import  | `"import"`  | Import existing database         |

Modes follow a consistent pattern: **each `modes` object overrides only
properties of the object it belongs to.**

| `modes` in…         | Documented supported overrides                                 |
| ------------------- | -------------------------------------------------------------- |
| `component.modes`   | `uiType`                                                       |
| `fieldParams.modes` | `disabled`, `label`, `helperText`, `defaultValue`, `autoFocus` |
| `validation.modes`  | `required`, `min`, `celExpressions`, …                         |

```yaml
components:
  dbName:
    uiType: text
    path: metadata.name
    modes: # component-level: overrides uiType
      restore:
        uiType: hidden
    fieldParams:
      label: Database name
      modes: # fieldParams-level: overrides documented field params
        edit:
          disabled: true # name can't change after creation
```

Mode-aware validation is declared per-component in the `validation` block
(see [validation.md](../../../docs/ui/ui-generator/validation.md#mode-aware-validation)
for full documentation):

## 8. Section Edit Modal Flow

```
[Overview Page]
       │
       │  user clicks Edit on a SchemaDrivenCard
       │
       ▼
 isSectionEditable(section, FormMode.Edit)  →  show/hide Edit button
       │
       ▼
 <SectionEditModal>
       │
       ├── applyModeOverrides(sections, Edit)
       ├── buildSectionZodSchema(sectionKey, editSections, {
       │       formMode: Edit,
       │       originalData: instance
       │     })
       ├── extractInstanceValues(editSections, instance)
       │
       ├── <FormDialog schema={zodSchema} defaultValues={...}>
       │      <UIGenerator sectionKey sections formMode={Edit} />
       │   </FormDialog>
       │
       └── on submit:
             postprocessSchemaData(formData)
             deepMerge into instance
             useUpdateDbInstanceWithConflictRetry(mutate)
```

---

## 9. Key Files

| Area               | File                                               |
| ------------------ | -------------------------------------------------- |
| Types              | `ui-generator/ui-generator.types.ts`               |
| Main component     | `ui-generator/ui-generator.tsx`                    |
| Context            | `ui-generator/ui-generator-context.tsx`            |
| Preprocess         | `utils/preprocess/preprocess-schema.ts`            |
| Mode overrides     | `utils/preprocess/apply-mode-overrides.ts`         |
| Resolve validation | `utils/validation/resolve-validation-for-mode.ts`  |
| Zod builder        | `utils/schema-builder/build-zod-schema.ts`         |
| Section Zod        | `utils/schema-builder/build-section-zod-schema.ts` |
| CEL validation     | `utils/schema-builder/cel-validation/`             |
| Defaults           | `utils/default-values/index.ts`                    |
| Instance values    | `utils/default-values/extract-instance-values.ts`  |
| Section check      | `utils/section-editable/is-section-editable.ts`    |
| Renderer           | `utils/component-renderer/render-component.tsx`    |
| Postprocess        | `utils/postprocess/postprocess-schema.ts`          |
| Schema walker      | `utils/schema-walker/schema-walker.ts`             |
| Object path        | `utils/object-path/object-path.ts`                 |
| Edit modal         | `cluster-overview/sections/section-edit-modal/`    |
