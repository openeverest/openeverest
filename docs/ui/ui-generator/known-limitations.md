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
