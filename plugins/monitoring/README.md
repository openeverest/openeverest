# OpenEverest Generic Plugin Template

A template repository for building [OpenEverest](https://github.com/openeverest/openeverest) generic plugins.

**This is a working plugin out of the box.** Install it as-is to see a "Hello" page in the OpenEverest UI with a sidebar entry, a dedicated route, and a cluster detail tab. Then customize it to build your own plugin.

## Quick Start

1. **Clone or use this template:**
   ```bash
   gh repo create my-org/plugin-my-feature --template openeverest/generic-plugin-template
   # or
   git clone https://github.com/openeverest/generic-plugin-template.git plugin-my-feature
   cd plugin-my-feature
   ```

2. **Rename the plugin** — replace all occurrences of `my-plugin` with your plugin name:
   ```bash
   # Example: renaming to "sql-browser"
   find . -type f -not -path './.git/*' -exec sed -i '' 's/my-plugin/sql-browser/g' {} +
   mv charts/my-plugin charts/sql-browser
   ```

3. **Install frontend dependencies:**
   ```bash
   npm install
   ```

4. **Download Go modules** (skip if you replaced the backend with another language):
   ```bash
   cd backend && go mod tidy && cd ..
   ```

5. **Start developing!**

## What You Get

When installed, this plugin registers:

- **Sidebar entry** — "My Plugin" appears in the OpenEverest navigation.
- **Route** — a dedicated page at `/plugins/my-plugin` showing a hello message and backend connectivity status.
- **Cluster detail tab** — a "My Plugin" tab on every database cluster page showing cluster metadata.
- **Backend API** — a `/api/hello` endpoint demonstrating the full request flow (browser → host proxy → backend).

## Architecture

```
plugin-my-feature/
├── src/
│   └── main.tsx              # Frontend — TypeScript/React plugin bundle
├── backend/
│   ├── main.go               # Backend — Go HTTP server (example; use any language)
│   └── go.mod
├── charts/
│   └── my-plugin/            # Helm chart (rename to your plugin name)
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/
│           ├── _helpers.tpl
│           ├── deployment.yaml
│           ├── plugin-cr.yaml
│           └── service.yaml
├── .github/
│   └── workflows/
│       ├── ci.yml            # Lint, type-check, build on push/PR
│       └── release.yml       # Build image + Helm chart on tag push
├── Dockerfile
├── package.json
├── tsconfig.json
└── vite.config.ts
```

### Request flow

```
Browser → GET /v1/plugins/{name}/api/...
         ↓ (host validates session, adds X-Everest-User JWT)
Backend → processes request, optionally calls OpenEverest API
         ↓
Browser ← JSON response
```

The backend also serves the frontend bundle at `GET /main.js`, which the OpenEverest shell fetches at startup to dynamically load the plugin UI.

> **The backend is language-agnostic.** It is simply an HTTP server that responds to requests proxied by OpenEverest. This template ships a Go example, but you can replace the `backend/` directory with **any language or framework** (Python/Flask, Node.js/Express, Rust/Axum, Java/Spring, etc.). The only contract is:
>
> 1. Serve `GET /main.js` — return the frontend bundle.
> 2. Serve `GET /healthz` — return 200 OK for liveness/readiness probes.
> 3. Handle your API routes under `/api/...`.
> 4. Read the `X-Everest-User` header to get the authenticated user's JWT.

## Plugin Spec

This template follows the [Generic Plugins Architecture Design](https://github.com/openeverest/specs/blob/main/specs/003-generic-plugins.md). Key concepts:

- **Plugin CR** — a cluster-scoped Kubernetes resource declaring your plugin's identity, frontend, backend, and permissions.
- **Frontend bundle** — an ESM module exporting a `register(api)` function that registers UI extension points.
- **Backend** — any HTTP service (language-agnostic) proxied by the OpenEverest API server; receives `X-Everest-User` JWT for auth.
- **Extension points** — `route`, `sidebarItem`, `clusterDetailTab`, `clusterAction`, `clusterCard`, `globalDashboardWidget`, `settingsPanel`, etc.

## Local Development

### Build the frontend bundle

```bash
npm install
npm run build        # outputs dist/main.js
```

### Run the backend locally

The example backend is written in Go, but you can use any language — it's just an HTTP server.

```bash
# Set the Everest API URL (defaults to in-cluster address)
export EVEREST_API_URL=http://localhost:8080

# The backend expects dist/main.js to exist (built above)
cd backend && go run .
```

### Dev server (frontend hot-reload)

```bash
npm run dev          # Vite dev server on http://localhost:3001
```

Point the Everest dev environment to `http://localhost:3001/main.js` for hot-reload during development.

## Build Docker Image

The provided `Dockerfile` assumes the Go backend. If you use a different language, update the Dockerfile accordingly.

```bash
npm run build                                   # produces dist/main.js
cd backend && go mod tidy && cd ..              # ensure go.sum exists
docker build -t my-plugin:dev .
```

## Install

### Try it now

You can install this template plugin into any OpenEverest v2+ cluster to see it in action:

```bash
helm install my-plugin oci://ghcr.io/openeverest/charts/my-plugin \
  --version 0.1.0 \
  -n everest-system
```

Open the OpenEverest UI — "My Plugin" will appear in the sidebar.

### Prerequisites

- An OpenEverest cluster with the Plugin CRD installed (Everest v2+)
- Helm 3
- `kubectl` configured to access the cluster

### From a release (recommended)

After pushing a `vX.Y.Z` tag, the CI builds and publishes the image and Helm chart to GHCR. Install directly from the OCI registry:

```bash
helm install my-plugin oci://ghcr.io/<your-org>/charts/my-plugin \
  --version 0.1.0 \
  -n everest-system
```

### From source (local development)

Build the image and install the chart from the local checkout:

```bash
# 1. Build the frontend and Docker image
npm install && npm run build
cd backend && go mod tidy && cd ..
docker build -t my-plugin:dev .

# 2. Load the image into your cluster (kind example)
kind load docker-image my-plugin:dev --name everest

# 3. Install via Helm
helm install my-plugin charts/my-plugin/ \
  -n everest-system \
  --set image.repository=my-plugin \
  --set image.tag=dev \
  --set image.pullPolicy=Never
```

### Verify

After installation, confirm the plugin is running:

```bash
kubectl get plugins                              # should show my-plugin
kubectl get pods -n everest-system -l app.kubernetes.io/name=my-plugin
```

Then open the OpenEverest UI — you should see "My Plugin" in the sidebar.

## Uninstall

```bash
helm uninstall my-plugin -n everest-system
```

## Configuration (`values.yaml`)

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Container image | `ghcr.io/openeverest/generic-plugin-template` |
| `image.tag` | Image tag | chart appVersion |
| `replicaCount` | Replicas | `1` |
| `service.port` | Service port | `8080` |
| `plugin.displayName` | Display name in the UI | `My Plugin` |
| `plugin.enabled` | Enable/disable the plugin | `true` |
| `everestAPIURL` | OpenEverest API server URL (in-cluster) | `""` (auto-discovered) |

## Releasing

The release workflow is triggered automatically when you push a semver tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

This will:
1. Build the frontend bundle
2. Build and push a multi-arch Docker image (`linux/amd64` + `linux/arm64`) to GHCR
3. Package and push the Helm chart to GHCR OCI registry
4. Create a GitHub release with the chart tarball

## References

- [Generic Plugins Spec (003)](https://github.com/openeverest/specs/blob/main/specs/003-generic-plugins.md)
- [Plugin SDK (`@openeverest/plugin-sdk`)](https://www.npmjs.com/package/@openeverest/plugin-sdk)
- [Example: MongoDB Explorer Plugin](https://github.com/openeverest/plugin-mongodb-explorer)

## License

Apache-2.0
