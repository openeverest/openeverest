# UI Generator — Known Limitations

Current gaps and unsupported cases. Check here before relying on a capability that is not
documented in the main [Readme](./Readme.md).

## Arrays

`uiType: array` supports one shape: a string map edited as key/value rows, see
[Array Field](components/array-field.md). Not supported yet:

- **Other shapes:** lists of strings or of objects, and items other than `keyValue`.
- **CEL:** `celExpressions` on the list, or on a row's key or value. `regex` on the list.
- **Backup class parameter forms** (backup schedules, on-demand backups, PITR): they don't write
  rows in the API shape. Use array fields in Instance and monitoring class schemas.
- **Paths into an array:** a `path` like `storages.0.name` — in a reusable section **or** in a
  main topology schema — will **not** bind. Arrays such as `storages[]` and `schedules[]` are
  managed by a host React component via `useFieldArray`, which renders one element's relative
  section per index.
- **Older OpenEverest versions** don't know `uiType: array` and don't show the field; a saved
  value is kept on edit.

**Before adding other array-bound fields to a schema, discuss with the team.**
