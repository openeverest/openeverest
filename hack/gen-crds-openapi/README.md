# CRD Schema Extractor

This tool extracts OpenAPI v3 schemas from Kubernetes Custom Resource Definition (CRD) YAML manifests and generates a separate OpenAPI specification file.

It uses the official Kubernetes `CustomResourceDefinition` type from `k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1` to parse CRD manifests.

## Usage

```bash
# Extract schemas from default location
go run hack/gen-crds-openapi/main.go

# Specify custom CRD directory and output file
go run hack/gen-crds-openapi/main.go -crd-dir=config/crd/bases -output=api/openapi/crds.gen.yaml
```

## Flags

- `-crd-dir` - Directory containing CRD YAML manifests (default: `config/crd/bases`)
- `-output` - Output OpenAPI YAML file (default: `api/openapi/crds.gen.yaml`)

## How it works

1. Reads all `.yaml` files from the specified CRD directory
2. Parses each CRD manifest using the official Kubernetes API types
3. Extracts the `openAPIV3Schema` from each CRD version
4. Creates corresponding List schemas (e.g., `DatabaseClusterList`)
5. Generates a complete OpenAPI 3.0.2 specification file with all schemas under `components.schemas`
