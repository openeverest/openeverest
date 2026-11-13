# UI Generator — Known Limitations

Current gaps and unsupported cases. Check here before relying on a capability that is not
documented in the main [Readme](./Readme.md).

## Maps and lists

`uiType: map` (**Tech Preview**) edits a string map as key/value rows, see
[Map Field](components/map-field.md). Not supported yet:

- **Other rows:** an `itemTemplate` other than the `keyValue` widget, e.g. a custom group or a map
  of objects. Such a map isn't rendered or sent, and an edit keeps its saved value.
- **Lists:** `uiType: list` (strings or objects) is planned; until then it's an unknown `uiType`.
- **Several paths** on a map: an edit reads only the first, so such a map isn't rendered.
- **CEL:** `celExpressions` on the map, or on a row's key or value. `regex` on the map.
- **Backup class parameter forms** (backup schedules, on-demand backups, PITR): they don't write
  rows in the API shape. Use map fields in Instance schemas and monitoring class destination forms.
- **Monitoring entry parameters on edit:** the Instance's monitoring editor loads saved entries as
  they are, so a map field in a monitoring class's `instance` form shows no rows for a saved map and
  fails validation. Adding an entry works; only saved ones are affected.
- **Paths into an array:** a `path` like `storages.0.name` — in a reusable section **or** in a
  main topology schema — will **not** bind. Arrays such as `storages[]` and `schedules[]` are
  managed by a host React component via `useFieldArray`, which renders one element's relative
  section per index.
- **Older OpenEverest versions** don't know `uiType: map` and don't show the field; a saved value
  is kept on edit.

**Before adding other map- or array-bound fields to a schema, discuss with the team.**
