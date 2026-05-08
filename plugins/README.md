# Generic Plugins PoC

This is a proof-of-concept for OpenEverest's generic plugin architecture.
It demonstrates two high-risk unknowns:

1. **Dynamic ESM loading in the React shell** — the host app dynamically
   `import()`s a plugin's JavaScript bundle at runtime and renders the
   plugin's React component inside the existing layout.

2. **Echo reverse proxy to plugin backend** — the Go API server proxies
   requests from `/v1/plugins/:name/*` to the plugin's backend service.

## Quick Start

### Option A: Tilt (recommended)

The plugin is automatically built and deployed when you run the standard
Tilt dev environment:

```bash
cd dev
tilt up
```

The hello plugin will appear as a `hello-plugin-deploy` resource in the
Tilt dashboard. Once the Everest server is ready at http://localhost:8080,
the sidebar will show a **"Hello Plugin"** entry.

### Option B: Manual (standalone dev)

#### 1. Start the hello plugin dev server

```bash
cd plugins/hello
npm install
npm run dev          # serves on http://localhost:3001
```

#### 2. Start the Go API server

```bash
# (from repo root, assuming in-cluster setup or dev config)
go run ./cmd/main.go
```

The Go server exposes:
- `GET /v1/plugins` — returns the plugin registry (hardcoded for PoC)
- `ANY /v1/plugins/:name/*` — reverse-proxies to the plugin backend

#### 3. Start the UI dev server

```bash
cd ui
pnpm install
pnpm --filter everest dev   # serves on http://localhost:5173
```

The Vite proxy forwards `/v1/*` to the Go server (port 8080).

### What to expect

- The sidebar shows a new **"Hello Plugin"** entry (puzzle piece icon)
- Clicking it navigates to `/plugins/hello`
- The page is rendered by a React component loaded from the plugin's ESM bundle
- The component has a click counter to prove it's live React, not an iframe

## Architecture

```
Browser
  │
  ├─ Everest Server (:8080, port-forwarded by Tilt)
  │    ├─ Serves frontend (React host app)
  │    ├─ GET /v1/plugins → plugin list (bundleUrl = /v1/plugins/hello/main.js)
  │    └─ ANY /v1/plugins/hello/* → reverse proxy → hello-plugin k8s Service (:3001)
  │
  └─ Plugin runs as a k8s Deployment + Service in everest-system namespace
       └─ Serves main.js (ESM bundle, React received from host at runtime)
```

## Files Changed

### Frontend (ui/apps/everest/src/)
- `contexts/plugins/` — PluginProvider context (loads + registers plugins)
- `components/plugin-host/PluginHost.tsx` — renders the plugin's route component
- `router/router.tsx` — added `plugins/:pluginName/*` wildcard route
- `components/drawer/Drawer.tsx` — merges plugin sidebar items dynamically
- `App.tsx` — wraps router with `<PluginProvider>`

### Backend (internal/server/)
- `plugins.go` — plugin registry, list handler, reverse-proxy handler
- `everest.go` — registers plugin routes on the Echo server

### Sample Plugin
- `plugins/hello/` — minimal Vite + React plugin that exports `register()`
