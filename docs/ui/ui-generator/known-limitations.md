# UI Generator — Known Limitations

Current gaps and unsupported cases. Check here before relying on a capability that is not
documented in the main [Readme](./Readme.md).

## Arrays / repeatable fields — not supported

UIGenerator has no array / repeatable component. A `path` that points into an array — in a
reusable section **or** in a main topology schema — will **not** bind.

- **Today:** arrays (e.g. `storages[]`, `schedules[]`) are managed by a host React component
  via `useFieldArray`, which renders one element's relative section per index.
- **Open idea:** a declarative array field (e.g. `uiType: array` with an `itemPath` the
  renderer iterates over). Reserved, not built.

**Before adding array-bound fields to a schema, discuss with the team.**

## CEL and Kubernetes qualified-name keys

Field `path` values may use bracket-quoted segments for keys that contain dots
(`resources.limits['nvidia.com/gpu']`). The form reads and writes those keys correctly.

CEL dependency extraction still only recognizes simple dotted identifiers
(`[A-Za-z0-9_]+(?:\.[A-Za-z0-9_]+)+`). A CEL expression that references a bracket-quoted
qualified key will not register that field as a dependency (it may only see the prefix before
the brackets). Prefer simple paths in CEL until this is extended.

## Toggleable group keys with special characters

Section and group keys for toggleable groups remain limited to letters, digits, `_` and `-`
until switch-name encoding is wired onto the shared path tokenizer — see
[#3221](https://github.com/openeverest/openeverest/issues/3221) and [groups.md](groups.md).
