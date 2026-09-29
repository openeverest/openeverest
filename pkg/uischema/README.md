# uischema

`Validate(uiSchema)` checks a provider UI schema (`Provider.spec.uiSchema`: topology name → `{sections: {<key>: {components: ...}}}`) and returns the toggleable groups the UI will not render as the author declared them, and the CEL rules that fail while such a group is switched off.

The UI applies the same rules at runtime in `ui/apps/everest/src/components/ui-generator/utils/toggleable/`: `resolveToggleable` (in `toggleable.ts`, called by `preprocessSchema`) and `findUnguardedCelReferences` (a dev-time warning). [testdata/toggleable.yaml](testdata/toggleable.yaml) is the shared set of cases both implementations are tested against: `validate_test.go` here and `toggleable-shared-cases.test.ts` in the same UI folder. CI runs both suites when the file changes.

## Definitions

Rules are evaluated per topology `T`.

- **Container.** An item with `uiType ∈ {group, hidden}`. It is walked if it has a `components` map and skipped otherwise. So a `hidden` field writes nothing.
- **Field.** Any other item that has a `uiType` key. An item without `uiType` writes nothing.
- **paths(f)**, the API paths a field `f` writes:
  - `path` is a non-empty string `s` → `{s}`
  - `path` is a list → the set of its non-empty string entries (duplicates collapse)
  - otherwise → `∅`
- **Fields(C)**, all fields under a components map `C`, found by walking every container recursively.
- **P(g)**, the paths of group `g`: the multiset `⋃ paths(f)` over `f ∈ Fields(g.components)`.
- **usage_T(p)**: the number of fields `f` in all sections of `T` with `p ∈ paths(f)`.
- **own_g(p)**: the number of fields `f ∈ Fields(g.components)` with `p ∈ paths(f)`.
- **Candidate.** An item with `uiType = group` and `groupType = toggleable`. A `hidden` container is never a candidate.
- **Key path.** The section key followed by the group keys, joined with `.`, for example `advanced.monitoring`.

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

- One `Issue{Rule, Topology, Group, Field, Message}` per group that matched a group rule (`Field` is empty), and per (field, group) pair for `toggleable-cel-unguarded` (`Field` is the key path of the field carrying the CEL rule).
- Issues are sorted by topology, then group key path, then field key path.
- `Message` is informational. Tests compare only `rule`, `topology`, `group` and `field`.

## Adding a case

Add an entry to `cases` in `testdata/toggleable.yaml`:

- `name`: a sentence describing the case;
- `uiSchema`: the input schema;
- `issues`: the expected list of `{rule, topology, group}`, plus `field` for `toggleable-cel-unguarded`. Use `[]` when the schema is valid.
