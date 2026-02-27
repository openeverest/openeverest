# OpenAPI TypeScript Types Generation

This directory contains OpenAPI schema files and tooling to generate TypeScript types from them using [`openapi-typescript`](https://openapi-ts.dev/).

Generated types are placed in the `generated/` directory.

## Usage

### First-time setup

Install dependencies:

```sh
make init
```

### Generate types for all files

```sh
make generate-all
```

This produces:

- `generated/crds.gen.types.ts` — from `crds.gen.yaml`
- `generated/http-api.types.ts` — from `http-api.yaml`

### Regenerate types for a specific file

```sh
make generate FILE=crds.gen.yaml
make generate FILE=http-api.yaml
```

### From the repository root

```sh
make gen-openapi-ts-types
```
