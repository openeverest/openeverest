# UI Generator Submit Postprocess

## Purpose

`postprocessSchemaData` runs after React Hook Form submission and before sending payload to API.

It does two things:

1. Removes empty values.
2. Applies multipath mapping for fields defined with `path: string[]`.

## Empty Value Rules

Default empty values (removed from payload):

- `undefined`
- `null`
- `''` (empty string)

Values that are **not** considered empty (kept in payload):

- `false`
- `0`
- `[]`
- Any non-empty object

Nested objects that become empty after cleanup are removed too.

## Multipath Mapping

When a component uses `path: string[]`, multipath now uses the **first path** as the React Hook Form field id (for example, `spec.engine.version`).

On submit, postprocess reads the value from this first (source) path and copies it to every additional target path. The source path remains in the payload.

For id-only multipath fields (controls that do not bind directly to a concrete data path), React Hook Form still uses a generated `g-*` field id as the source field. In that case, postprocess copies the value from the `g-*` field to all configured paths and then removes the generated `g-*` field from the payload.

Example:

```ts
// schema component
{
  uiType: 'text',
  path: ['spec.engine.version', 'spec.proxy.version']
}

// RHF submit values
{ 'spec.engine.version': '8.0.41' }

// API payload after postprocess
{
  spec: {
    engine: { version: '8.0.41' },
    proxy: { version: '8.0.41' }
  }
}
```

## Note For Plugin Authors

If a required field resolves to an empty value (`undefined`, `null`, `''`), validation should fail before submit. If such value still reaches postprocess, it is removed from payload by design.
