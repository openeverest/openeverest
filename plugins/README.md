# Generic Plugins PoC

This is a proof-of-concept for OpenEverest's generic plugin architecture.
It demonstrates two high-risk unknowns:

1. **Dynamic ESM loading in the React shell** — the host app dynamically
   `import()`s a plugin's JavaScript bundle at runtime and renders the
   plugin's React component inside the existing layout.

2. **Echo reverse proxy to plugin backend** — the Go API server proxies
   requests from `/v1/plugins/:name/*` to the plugin's backend service.

## Hello Plugin

The sample hello plugin has been moved to its own repository:

**https://github.com/openeverest/plugin-hello**

See that repo for build instructions, Helm chart installation, and local development setup.

## Quick Start (Tilt)

Set `HELLO_PLUGIN_DIR` to point to a local clone of the plugin-hello repo:

```bash
export HELLO_PLUGIN_DIR=/path/to/plugin-hello
cd dev
tilt up
```

The hello plugin will appear as a `hello-plugin-deploy` resource in the
Tilt dashboard. Once the Everest server is ready at http://localhost:8080,
the sidebar will show a **"Hello Plugin"** entry.

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

### Sample Plugin (external repo)
- [openeverest/plugin-hello](https://github.com/openeverest/plugin-hello) — minimal Vite + React plugin that exports `register()`

