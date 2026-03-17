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

When a component uses `path: string[]`, RHF stores one source field (generated ID, e.g. `g-engineVersion`).

On submit, postprocess copies that value to every target path and removes the generated source field.

Example:

```ts
// schema component
{
  uiType: 'text',
  path: ['spec.engine.version', 'spec.proxy.version']
}

// RHF submit values
{ 'g-engineVersion': '8.0.41' }

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
