# OpenAPI Schemas

This directory contains OpenAPI schema YAML files consumed by both the backend and frontend.

## TypeScript types

TypeScript types are generated from these files and output to [`api/ui/`](../ui/).
Generation is managed from `ui/Makefile` using [`openapi-typescript`](https://openapi-ts.dev/).

### Generate types for all files

```sh
# from ui/
make generate-openapi-types

# or from the repository root
make gen-openapi-ts-types
```

This produces:

- `api/ui/crds.gen.types.ts` — from `crds.gen.yaml`
- `api/ui/http-api.types.ts` — from `http-api.yaml`
- `api/ui/index.ts` — barrel re-exporting all types under named namespaces

### Regenerate types for a specific file

```sh
# from ui/
make generate-openapi-type FILE=crds.gen.yaml
make generate-openapi-type FILE=http-api.yaml
```
