# UI Generator — Known Limitations

Current gaps and unsupported cases. Check here before relying on a capability that is not
documented in the main [Readme](./Readme.md).

## Maps and lists

`uiType: map` (**Tech Preview**) edits a string map as key/value rows, see
[Map Field](components/map-field.md). `uiType: list` (**Tech Preview**) edits a list of strings as
rows of one input, see [List Field](components/list-field.md). Not supported yet:

- **Other map rows:** an `itemTemplate` other than a group of a text key and a text value, e.g. a
  typed value, a third field or a map of objects. Both inputs are text, so the map writes
  strings; a number, quantity, toggle or select value (with a `dataSource` too) is being
  considered. Such a map isn't rendered or sent, and an edit keeps its saved value. See
  [Not Supported Yet](components/map-field.md#not-supported-yet) for the whole list.
- **Lists of objects and typed items:** an `itemTemplate` of a list other than one text field
  with no `path` or `id`, e.g. a group of fields (a list of objects) or a number, quantity, toggle
  or select item. The input is text, so the list writes strings. Such a list isn't rendered or
  sent, and an edit keeps its saved value. See
  [Not Supported Yet](components/list-field.md#not-supported-yet) for the whole list.
- **On a row's field** (a map's key or value, a list's item): `defaultValue` (a new row starts
  empty; the field's own `defaultValue` sets the initial entries), `celExpressions`, `dataSource`,
  `badge` / `badgeToApi` and component-level `modes`. The UI ignores them, as it does `required`
  on a list's item and `unique` on a map, which have nothing to check.
- **Several paths** on a map or a list: an edit reads only the first, so such a field isn't
  rendered.
- **CEL on a row's field**, and `regex` on the map or the list. The UI ignores them and
  `provider-sdk generate` rejects them. A rule on the field itself or on another field can read
  it, see [Reading a Map Field](validation.md#reading-a-map-field) and
  [Reading a List Field](validation.md#reading-a-list-field).
- **Moving rows:** a list is written in the order of its rows, and a row can't be moved. To
  change the order, the user removes an entry and adds it again.
- **A saved value that isn't a map of strings** (possible only where the API takes free-form
  `parameters`, e.g. written with kubectl): an edit reads it as rows, so saving the section writes
  a number or boolean back as a string (`3` → `"3"`), an object as `"[object Object]"`, and
  removes a value that isn't an object at all. Bind a map field only to a path whose API schema is
  a map of strings.
- **A saved value that isn't a list of strings**, in the same way: an edit reads every entry as
  text, so saving the section writes a number or boolean back as a string (`3` → `"3"`), `null`
  as `"null"`, an object as `"[object Object]"`, drops an entry that is an empty string, and
  removes a value that isn't a list at all. Bind a list field only to a path whose API schema is an array of strings.
- **Backup class parameter forms** (backup schedules, on-demand backups, PITR): they don't write
  rows in the API shape, and a field with no `defaultValue` shows no row there. Use map and list
  fields in Instance schemas and monitoring class destination forms.
- **Saved monitoring entry parameters:** the Instance's monitoring editor, a restore and a preset
  load saved entries as they are, so a map field in a monitoring class's `instance` form shows no
  rows for a saved map and fails validation. A list field there shows an empty row for each saved
  entry and fails validation with `These entries can't be edited in this form`. A map or a list
  field the saved entry has no value for shows no row, only its Add button. Adding an entry
  works; only saved ones are affected.
- **A class's parameters schema** checks a list as a whole (`required`): the entries, described
  by `items`, are left to the server.
- **Paths into an array:** a `path` like `storages.0.name` — in a reusable section **or** in a
  main topology schema — will **not** bind. Arrays such as `storages[]` and `schedules[]` are
  managed by a host React component via `useFieldArray`, which renders one element's relative
  section per index.
- **Older OpenEverest versions** don't know `uiType: map` or `uiType: list` and don't show the
  field; a saved value is kept on edit.

**Before adding other map- or array-bound fields to a schema, discuss with the team.**
