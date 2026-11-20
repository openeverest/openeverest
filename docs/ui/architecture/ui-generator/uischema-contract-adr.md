# ADR: uiSchema contract — ownership, versioning and enforcement

- **Status:** proposed, in review by UI and BE
- **Tracking:** [#3480](https://github.com/openeverest/openeverest/issues/3480), part of the epic [#3492](https://github.com/openeverest/openeverest/issues/3492)
- **Discussion:** [proposal](https://github.com/openeverest/openeverest/issues/3480#issuecomment-6848813928) and [BE reply](https://github.com/openeverest/openeverest/issues/3480#issuecomment-6918257307)

## Context

A uiSchema describes a form that UI Generator renders. It is produced outside the UI and stored in CRDs, so it is a public contract between provider authors, `provider-sdk`, core and the UI.

It is stored in six places:

| #   | Field                                  | Forms it holds                                         | Produced by                                                                  |
| --- | -------------------------------------- | ------------------------------------------------------ | ---------------------------------------------------------------------------- |
| 1   | `Provider.spec.uiSchema`               | one per topology                                       | `provider-sdk generate`, from the `ui:` blocks in `definition/topologies/*/` |
| 2   | `Provider.spec.secrets[*].uiSchema`    | one form                                               | `provider-sdk generate`                                                      |
| 3   | `Provider.spec.configMaps[*].uiSchema` | one form                                               | `provider-sdk generate`                                                      |
| 4   | `BackupClass.spec.uiSchema`            | by form, e.g. `backup`, `pitr`, `restore`              | `provider-sdk generate`, from `definition/backupclasses/*/`                  |
| 5   | `BackupStorageClass.spec.uiSchema`     | `location`, `storage`, `credentials` per auth mode     | core, `hack/gen-backupstorageclasses`                                        |
| 6   | `MonitoringClass.spec.uiSchema`        | `destination`, `instance`, `credentials` per auth mode | the monitoring backends, written by hand in their charts                     |

Problems today:

- **No single definition.** The shape is described by hand in the UI types (`ui-generator.types.ts`), the schema builder's zod schema, the Go toggleable rules, the SDK path checks and the docs. These drift: `provider-sdk generate` wrote `optionsPathConfig.defaultValuePath` before the UI could read it ([#3265](https://github.com/openeverest/openeverest/issues/3265)).
- **No version.** A provider cannot say which vocabulary it was built for, and the UI cannot tell a schema that is newer than itself from a broken one.
- **No structural check outside the browser.** `uischema.Validate` checks only the toggleable rules, and only for `Provider.spec.uiSchema`. In the browser an unknown `uiType` renders nothing and passes validation.
- **The vocabulary changes from three sides:** the UI, the SDK and community PRs (`uiType: quantity` in [#3522](https://github.com/openeverest/openeverest/pull/3522), `gate` in [#3526](https://github.com/openeverest/openeverest/pull/3526)).

## Decisions

### 1. Scope

The contract is the shared vocabulary (field types, `fieldParams`, `validation`, groups, widgets, `modes`), the shape of each of the six envelopes above, and the version of both.

Not part of the contract: fields the UI adds internally while processing a schema, and how a prop is rendered.

### 2. One definition, owned by the UI, with a generated JSON Schema

- The shape is defined once in a private workspace package, `ui/packages/ui-schema` (zod 4). The UI takes its TS types and its runtime validation from it.
- A JSON Schema (draft-07) is generated from the package into `provider-runtime/uischema/schema/v<major>/`, one file per envelope, and committed. Go embeds it.
- The generated files are read-only. CI regenerates them and fails on a diff.
- The package uses only constraints that JSON Schema can express, so TS and Go accept exactly the same documents. Both run the same cases from `provider-runtime/uischema/testdata/`, as they already do for the toggleable rules.
- Rules that JSON Schema cannot express (toggleable today, `gate` later) stay hand-written in Go and TS against shared test data.
- `provider-runtime` is a public SDK: a schema change reaches every provider when it bumps core. A PR that changes the package or the generated schema is treated as an API change and needs approval from one UI and one BE maintainer.

The UI owns the definition because the meaning of a prop is set by the code that renders it, and the UI needs both the types and the runtime check.

### 3. Version

- The contract has a version `MAJOR.MINOR`.
  - **Minor:** additive. A new `uiType`, prop or allowed value. Every document that was valid stays valid and keeps its meaning.
  - **Major:** anything else. Removing or renaming a prop, changing its type or its meaning.
- The version is frozen as `1.0` before GA, once the vocabulary is settled. Until then it stays `1.0` and the vocabulary may still change without a bump. After the freeze every vocabulary change bumps it.
- The version is declared once, in the contract package, and generated into Go next to the schema, so Go tools read it instead of repeating the number.
- Objects carry it in a typed CRD field, `spec.uiSchemaVersion`, on `Provider`, `BackupClass`, `BackupStorageClass` and `MonitoringClass`.
  - Optional string in the form `MAJOR.MINOR`.
  - One field per object. It covers every uiSchema the object holds; on a `Provider` that is `spec.uiSchema` and the secret and configMap schemas.
  - Absent means `1.0`. This also covers objects created before the field existed.
- Authors never write the version. The tool that generates the manifest writes the contract version of the core it is built against.

### 4. Who writes the version and runs the check

| Producer                                    | Objects                                                           | Today                                                       |
| ------------------------------------------- | ----------------------------------------------------------------- | ----------------------------------------------------------- |
| `provider-sdk generate`                     | `Provider` (with its secret and configMap schemas), `BackupClass` | Calls `uischema.Validate` for `Provider.spec.uiSchema` only |
| `hack/gen-backupstorageclasses` in core     | built-in `BackupStorageClass`                                     | No check                                                    |
| The class generator in core, `pkg/classgen` | `MonitoringClass` and other classes built outside the SDK         | Renders parameter schemas and validations only; no uiSchema |

Each producer writes `spec.uiSchemaVersion` and fails when `uischema.Validate` reports an issue.

### 5. Enforcement and behaviour on mismatch

**At build time.** The existing `uischema.Validate` is extended; there is no second entry point.

- It adds a structural check against the embedded JSON Schema: unknown keys, wrong value types, unknown `uiType`.
- It accepts every envelope through new optional `Input` fields, not only the Provider topologies.
- Any issue, structural or behavioural, fails generation.
- In `provider-sdk generate` the check runs after the SDK's own injections, so the SDK cannot emit a prop the contract does not know.

**In the browser.** The UI validates a form's schema when it loads and compares `spec.uiSchemaVersion` with the contract version it was built with.

| Schema version          | Behaviour                                                                                                    |
| ----------------------- | ------------------------------------------------------------------------------------------------------------ |
| Same major, minor ≤ UI  | Strict validation. A valid schema renders; an invalid one shows an explicit error state for that form.       |
| Same major, newer minor | Renders. Unknown props and unknown `uiType`s are ignored, and the form shows a notice to update OpenEverest. |
| Another major           | The form is not rendered and shows a clear message.                                                          |

**In core.** No check for now: the generators and the UI catch almost everything. Core never rejects an object because of its uiSchema, so an installed provider keeps working after a core upgrade. If invalid schemas are seen reaching clusters, core reports them with a `UISchemaValid` condition in `status.conditions`, not with an admission webhook or Events ([#3490](https://github.com/openeverest/openeverest/issues/3490)).

### 6. Change process

Contract first. A new prop or `uiType` lands in one PR: the contract package, the regenerated JSON Schema, the UI implementation, the docs and the shared cases, plus the minor bump after the freeze. The SDK and providers use it after they bump core.

The build-time check enforces this order: a prop the contract does not know fails generation.

## Deferred

- **Core warnings.** See decision 5; tracked in [#3490](https://github.com/openeverest/openeverest/issues/3490).
- **Editor support.** A JSON Schema for the SDK's `topology.yaml`, embedding the uiSchema one, so that `# yaml-language-server: $schema=…` works. To be done once the contract is stable.
- **CEL.** Where a rule lives (uiSchema `celExpr` or provider Go code) and whether `cel-js` and `cel-go` agree on it are separate questions, out of scope for this ADR.

## Alternatives considered

| Alternative                                        | Why not                                                                                                                                           |
| -------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| A reserved `schemaVersion` key inside `uiSchema`   | Clashes with topology and form names, is untyped and is not visible in `kubectl`.                                                                 |
| JSON Schema or Go types as the hand-written source | The UI would still need hand-written types and a runtime check, which is the duplication this ADR removes.                                        |
| A separate Go entry point for the structural check | `provider-sdk generate` already calls `uischema.Validate` and fails on its issues; one entry point means a core bump is enough to get new checks. |
| Rejecting an invalid uiSchema at admission         | An installed provider could stop working after a core upgrade, and a webhook outage would block provider installs.                                |

## Consequences

- **CRD API.** One new optional field on four kinds. It is public API, so its name is final once released.
- **`provider-runtime`.** Gains a generated directory that UI PRs can change. BE reviews those changes.
- **Providers.** They get the structural check when they bump core. `generate` can start failing on schemas with typos or unknown props, which is intended.
- **`provider-sdk`.** A small change: pass the BackupClass, secret and configMap schemas to `Validate`, and write `spec.uiSchemaVersion`.
- **Class generators.** `pkg/classgen` and `hack/gen-backupstorageclasses` need to write the version and run the check. Neither handles this today, and there is no issue for it yet.
- **UI.** A form with an invalid schema shows an error state instead of a partial form. This is user-visible and needs a UX review.

## Follow-up work

| Work                                                                             | Issue                                                                                                                                                                                             |
| -------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Contract package                                                                 | [#3483](https://github.com/openeverest/openeverest/issues/3483)                                                                                                                                   |
| JSON Schema generation and the structural check in Go                            | [#3485](https://github.com/openeverest/openeverest/issues/3485)                                                                                                                                   |
| `spec.uiSchemaVersion` on the four CRDs                                          | none yet                                                                                                                                                                                          |
| `provider-sdk generate`: pass every envelope, write the version                  | [#3489](https://github.com/openeverest/openeverest/issues/3489)                                                                                                                                   |
| Class generators: write the version, run the check                               | none yet                                                                                                                                                                                          |
| UI: types from the package, builder validation, browser validation, version gate | [#3486](https://github.com/openeverest/openeverest/issues/3486), [#3487](https://github.com/openeverest/openeverest/issues/3487), [#3488](https://github.com/openeverest/openeverest/issues/3488) |
| Docs for provider authors, contract changelog                                    | [#3491](https://github.com/openeverest/openeverest/issues/3491)                                                                                                                                   |
| Core condition, if needed                                                        | [#3490](https://github.com/openeverest/openeverest/issues/3490)                                                                                                                                   |

## Metadata

- Owner: UI
- Status: proposed
- Last updated: 2026-11-20
