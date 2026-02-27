# OpenAPI Schemas

This directory contains generated TypeScript types from OpenAPI schema YAML files consumed by frontend.

## TypeScript types

TypeScript types are generated from api/openapi files and output to [`ui/api/`](.).
Generation is managed from `ui/Makefile` using [`openapi-typescript`](https://openapi-ts.dev/).

### Generate types for all files

```sh
# from ui/
make generate-openapi-types

# or from the repository root
make gen-openapi-ts-types
```

This produces:

- `ui/api/crds.gen.types.ts` — from `api/openapi/crds.gen.yaml`
- `ui/api/http-api.types.ts` — from `api/openapi/http-api.yaml`
- `ui/api/index.ts` — barrel re-exporting all types under named namespaces

### Regenerate types for a specific file

```sh
# from ui/
make generate-openapi-type FILE=crds.gen.yaml
make generate-openapi-type FILE=http-api.yaml
```
