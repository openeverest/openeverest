# uischema

`Validate(Input{UISchema: ...})` checks a provider UI schema (`Provider.spec.uiSchema`: topology name → `{sections: {<key>: {components: ...}}}`) and returns the map fields and toggleable groups the UI will not render as the author declared them, the CEL rules that fail while such a group is switched off, and the parts of a schema the UI ignores. `provider-sdk generate` rejects a schema with any issue.

The UI applies the same rules at runtime, in `preprocessSchema` (`ui/apps/everest/src/components/ui-generator/utils/preprocess/`): first `stripEngineKeys` (`utils/preprocess/strip-engine-keys.ts`), then `resolveMapFields` (`utils/map-field/`), then `resolveToggleable` (`utils/toggleable/toggleable.ts`), with `lintMapFields` (`utils/map-field/lint-map-fields.ts`) and `findUnguardedCelReferences` as dev-time warnings. Every file in [testdata/](testdata) is a shared set of cases both implementations are tested against: `validate_test.go` here and `utils/preprocess/preprocess-shared-cases.test.ts` in the UI. CI runs both suites when a file there changes.

## Definitions

Rules are evaluated per topology `T`.

- **Container.** An item with `uiType ∈ {group, hidden}`. It is walked if it has a `components` map and skipped otherwise. So a `hidden` field writes nothing.
- **Field.** Any other item that has a `uiType` key. An item without `uiType` writes nothing.
- **paths(f)**, the API paths a field `f` writes:
  - `path` is a non-empty string `s` → `{s}`
  - `path` is a list → the set of its non-empty string entries (duplicates collapse)
  - otherwise → `∅`

  Not covered: a marker widget (`uiType: widget` without `path`, e.g. `podSchedulingPolicy`) writes paths the UI resolves from the Provider at runtime, so `Validate` sees `∅` for it. A toggleable holding only such widgets is reported as `toggleable-no-fields`, and overlaps through widget paths are not detected.
- **Fields(C)**, all fields under a components map `C`, found by walking every container recursively. For the group and CEL rules it leaves out the map fields a [map rule](#map-rules) drops: the UI doesn't render them, so they write nothing.
- **P(g)**, the paths of group `g`: the multiset `⋃ paths(f)` over `f ∈ Fields(g.components)`.
- **usage_T(p)**: the number of fields `f` in all sections of `T` with `p ∈ paths(f)`.
- **own_g(p)**: the number of fields `f ∈ Fields(g.components)` with `p ∈ paths(f)`.
- **Candidate.** An item with `uiType = group` and `groupType = toggleable`. A `hidden` container is never a candidate.
- **Key path.** The section key followed by the group keys, joined with `.`, for example `advanced.monitoring`. A field's key path ends with its own key, for example `advanced.labels`.
- **Overlap.** Paths `p` and `q` overlap when `p = q`, or one starts with the other followed by `.`: `spec.labels` overlaps `spec.labels` and `spec.labels.team`, but not `spec.labelsMode`.

## Map rules

They run first, on every field `m` with `uiType = map`:

| # | Rule | Condition |
|---|------|-----------|
| 1 | `map-unsupported` | `m.itemTemplate` is not `{uiType: widget, widgetType: keyValue}` |
| 2 | `map-multi-path` | `paths(m)` has more than one path |
| 3 | `map-overlap` | another field `f` (any field but `m` that rules 1–2 didn't match, in any section of `T`) has a path in `paths(f)` that overlaps a path in `paths(m)` |
| — | *(rendered)* | none of the above |

**Effect.** The UI drops every map field that matched a rule, like an unknown `uiType`: it is neither rendered nor sent, and an edit keeps its saved value.

**Why.** Only rows from the `keyValue` template widget are implemented (rule 1). An edit reads a map from its first path and would leave the others drifting (rule 2). A map field writes its whole value, so a field writing its path, a path inside it or a path it sits in would clash with it (rule 3).

**Consequences:**

- Two map fields that overlap both match rule 3, whatever their order.
- A field matching rule 1 or 2 writes nothing, so it never makes another map field overlap.

## Group rules

Groups are walked top-down, carrying a flag `inside`. The flag is `false` at the top of each section.

A candidate `g` gets the first rule that matches:

| # | Rule | Condition |
|---|------|-----------|
| 1 | `toggleable-unsafe-key` | some key on the key path of `g` does not match `^[A-Za-z0-9_-]+$` |
| 2 | `toggleable-no-fields` | `P(g) = ∅` |
| 3 | `toggleable-nested` | `inside = true` |
| 4 | `toggleable-overlap` | there is a path `p ∈ P(g)` with `usage_T(p) > own_g(p)` |
| — | *(valid toggleable)* | none of the above |

How the `inside` flag passes to children:

| Group | Children are walked with |
|-------|--------------------------|
| Valid toggleable | `inside = true` |
| Candidate that matched a rule | the parent's `inside` value, unchanged |
| Any other container | the parent's `inside` value, unchanged |

**Effect.** Every group that matched a rule renders as `groupType: bordered`. Its fields are always visible, always validated and always sent.

**Why.** The keys name the group's form-only switch (rule 1). Switching a toggleable off deletes every `p ∈ P(g)` from the payload. So a toggleable must own its paths exclusively (rule 4), must have something to delete (rule 2), and must not sit inside another active toggleable (rule 3).

**Consequences:**

- Two toggleables that share a path both match rule 4, whatever their order.
- Two fields inside the same group that write the same path do not overlap each other.
- A toggleable inside a group that matched a rule is judged as if that outer group were plain.

## Map option rules

They run on every map field `m` the UI renders (no [map rule](#map-rules) matched it). Each matching rule gives one issue, in this order:

| Rule | Condition |
|------|-----------|
| `map-ignored-option` | `m` sets an option the UI ignores: `regex` or a non-empty `celExpressions` in `m.validation` or in one of its `modes.<mode>` branches; a non-empty `celExpressions` in `m.itemTemplate.validation.<f>` or its mode branches; or `defaultValue`, `badge` or `badgeToApi` in `m.itemTemplate.fieldParams.<f>`, for `f ∈ {key, value}`. An option counts as set when its key is present. |
| `map-invalid-default` | `m.fieldParams.defaultValue`, or `m.fieldParams.modes.<mode>.defaultValue`, is present and is not a map whose values are all strings |
| `map-invalid-limits` | in a form mode's validation, `minItems` or `maxItems` is present and is not a whole number of 0 or more, or both are and `minItems > maxItems`. A mode's validation is `m.validation` without `modes`, with `modes.<mode>` over it unless that branch sets `inheritShared: false`; `m.validation` itself is checked too. |

**Effect.** The UI renders the field and ignores the option (`map-ignored-option`), starts the field empty (`map-invalid-default`), or applies limits no map meets (`map-invalid-limits`).

**Why.** A row starts empty and writes only its key and value, and CEL can't read a map or a row yet. A default that isn't a map of strings would be written back as text. Limits that clash block every save.

## Template widget rule: `template-widget-misplaced`

A field with `uiType = widget` and a template `widgetType` (`keyValue`) stands for a piece of schema and is only read as a map field's `itemTemplate`. Anywhere else the UI renders nothing for it.

## Reserved key rule: `reserved-key`

A key starting with `_` on any item under `components` (a field, a container or an item without `uiType`), at any depth, is reserved: the UI derives such keys itself (`_normalized`, `_row`, `_toggleable`, `_widgetTargets`, `_renderOverrides`) and drops any a schema sets before reading it. One issue per item that sets at least one; `Field` is the item's key path, a group's included.

## CEL rule: `toggleable-cel-unguarded`

While a valid toggleable `g` is off, the UI evaluates CEL with every `p ∈ P(g)` removed. A CEL rule that reads `p` without checking it first then fails with a CEL error and blocks submit. Rules on fields inside `g` are skipped while it is off.

Definitions:

- **CEL(f)**: every `celExpr` in `f.validation.celExpressions` and in `f.validation.modes.<mode>.celExpressions` for every mode. Here `f` is any item that is not a container.
- **refs(e)**: every match of `spec(\.\w+)+` in `e` that is not preceded by a word character or a dot. So `original.spec.x` (the saved object, which is never removed) does not count.
- **guards(e)**: every path `q` that appears as `has(q)` in `e`.

A field `f` that sits outside `g` (it is not in `Fields(g.components)`) gets one issue for `g` when:

> there is an expression `e ∈ CEL(f)`, a reference `r ∈ refs(e)` and a path `p ∈ P(g)` such that (`r = p` or `r` starts with `p.`) and `p ∉ guards(e)`.

Notes:

- Only `has()` on the exact path `p` counts. `has(spec.monitoring)` does not guard `spec.monitoring.interval`: the parent object can still be there when the field is gone.
- The check is textual. It does not follow the logic of the expression, so any `has(p)` anywhere in `e` counts as a guard.
- Groups that matched a group rule never switch off, so they are not checked.
- This rule does not change rendering. The group still works; only the CEL rule is wrong.

## Output

- One `Issue{Rule, Topology, Group, Field, Message}` per map field that matched a map rule or a map option rule, per misplaced template widget and per item with a reserved key (`Group` is empty, `Field` is the key path), per group that matched a group rule (`Field` is empty), and per (field, group) pair for `toggleable-cel-unguarded` (`Field` is the key path of the field carrying the CEL rule).
- Issues with the same topology, group and field keep the order the rules are listed in here.
- Issues are sorted by topology, then group key path, then field key path.
- `Message` is informational. Tests compare only `rule`, `topology`, `group` and `field`.

## Adding a case

Add an entry to `cases` in the file of the rule's family, e.g. `testdata/map.yaml`, `testdata/map-options.yaml`, `testdata/reserved.yaml` or `testdata/toggleable.yaml`:

- `name`: a sentence describing the case;
- `uiSchema`: the input schema;
- `issues`: the expected list of `{rule, topology}` plus `group` for a group rule, `field` for a map rule, and both for `toggleable-cel-unguarded`, in the order above. Use `[]` when the schema is valid.

Both suites run every file in `testdata/`. A new rule family needs its UI outcome in `toUiOutcome` of `preprocess-shared-cases.test.ts`: for most rules the UI only shows what a rule does to the schema, not which rule it was. The map option, template widget and reserved key rules are the exception: the UI reports them by name (`lintMapFields`, `stripEngineKeys`), so they are compared by name.
