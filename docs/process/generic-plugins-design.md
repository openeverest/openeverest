# Generic Plugins — Architecture Design

- **Status:** Draft
- **Authors:** @spron-in
- **Created:** 2026-05-08
- **Related specs:** [001 — Modular core / Provider plugins](https://github.com/openeverest/specs/blob/main/specs/001-plugins-architecture.md), [003 — Generic plugins problem space](https://github.com/openeverest/specs/blob/main/specs/003-generic-plugins.md)

---

## 1. Summary

Spec 001 introduced a plugin system for provisioning and lifecycle management of
databases backed by Kubernetes operators. This document designs the complementary
**generic plugin** layer — self-contained, installable extensions that consume
OpenEverest's existing databases and data without managing them. Generic plugins
can contribute UI pages, sidebar entries, database-detail panels, CLI subcommands,
and backend API logic, all without rebuilding or redeploying the OpenEverest core.

---

## 2. Terminology

| Term | Meaning |
|---|---|
| **Host** | The OpenEverest core: API server, React web UI shell, and `everestctl`. |
| **Plugin** | A generic extension authored by anyone, distributed as an OCI artifact. |
| **Manifest** | The `Plugin` Kubernetes CR that describes a plugin. Always required. |
| **Backend** | An optional HTTP service that implements custom logic on behalf of the plugin. |
| **Frontend bundle** | An optional ESM JavaScript module loaded at runtime into the web UI shell. |
| **Extension point** | A named, typed slot in the host UI or CLI where a plugin registers a contribution. |
| **PluginInstallation** | A namespace-scoped CR that enables a plugin within a specific namespace and holds per-tenant config. |

---

## 3. Goals & Non-Goals

**Goals:**

- Define what a generic plugin is and how it differs from a spec 001 Provider plugin.
- Design concrete manifest schema, CRDs, and lifecycle mechanics.
- Define the UI extension-point taxonomy and the frontend SDK surface.
- Define how plugin backends are hosted, reached, and authenticated.
- Integrate with the existing RBAC / auth model without introducing new auth concepts.
- Establish a clear security model for both frontend bundles and backend services.
- Provide a phased roadmap so v1 is deliverable quickly.

**Non-Goals:**

- Replacing or modifying the spec 001 Provider model.
- Defining a plugin marketplace or catalog UI.
- Tooling for plugin development (scaffolding, testing framework).
- Full CRD YAML schemas or SDK implementation code — those belong in follow-up issues.

---

## 4. Plugin Anatomy

A plugin has three optional parts. The **manifest** is always required; the other
two are present only when the plugin actually needs them.

```
plugin/
├── manifest.yaml        # Plugin CR, the single source of truth
├── main.js              # (optional) Frontend ESM bundle
└── server               # (optional) Backend binary / container image
```

### 4.1 Manifest

The manifest is a Kubernetes `Plugin` CR (cluster-scoped). It declares:

- Identity: name, version, vendor, description, icon.
- Frontend: URL or OCI/ConfigMap reference to the JS bundle.
- Backend: in-cluster `Service` reference or external HTTPS URL + credentials secret.
- Extension-point registrations (which slots in the UI/CLI the plugin fills).
- RBAC permissions the plugin requires (what it needs to call against the OpenEverest API).
- CLI contribution (optional).
- Compatibility range: which OpenEverest host versions this plugin supports.

### 4.2 Backend (optional)

Any HTTP service. The host never calls it directly from the browser — all traffic
goes through the OpenEverest API proxy. Two hosting modes:

- **In-cluster**: a `Service` name + port in a declared namespace.
- **External**: an HTTPS URL with credentials stored in a `Secret` referenced by the `PluginInstallation`.

A backend can operate in one or more **operation modes** (see §8 for the full
model):

- **Request handler** — responds to HTTP traffic proxied from the UI / `everestctl`.
  This is the default mode used by interactive plugins (SQL browser, AI copilot).
- **Daemon** — runs continuously in the background with no inbound user traffic.
  Used for metering, periodic syncs, scheduled reports, AWS RDS discovery pollers.
- **Event consumer** — holds an open subscription to the host's lifecycle event
  stream and reacts to resource changes (cluster created, deleted, backup
  completed, etc.). Used for audit, billing, external-system synchronisation.
  This is *pull-based*: the plugin opens the stream; the host does not push
  outbound HTTP to the plugin.

The modes are not mutually exclusive: a billing plugin typically combines all
three — a daemon to roll up usage, a held-open event subscription to capture
lifecycle, and a request-handler endpoint to serve the in-UI invoice page.

### 4.3 Frontend bundle (optional)

A single ESM JavaScript file. It exports a `register(api)` function that calls
`api.registerExtension(point, component)` to fill extension points. The React
shell dynamically imports it at startup after fetching the enabled-plugins list
from `GET /v1/plugins`.

### 4.4 CLI contribution (optional)

A container image (can be the same image as the backend) that the plugin exposes
as a CLI subcommand through `everestctl`.

---

## 5. CRD Sketches

### 5.1 `Plugin` (cluster-scoped)

```yaml
apiVersion: core.openeverest.io/v1alpha1
kind: Plugin
metadata:
  name: sql-explorer
spec:
  displayName: "SQL Explorer"
  description: "Query and browse your databases directly from the OpenEverest UI."
  version: "1.2.0"
  compatibleHostVersions: ">=2.0.0 <3.0.0"
  vendor: "Acme Corp"
  icon: "https://example.com/icon.png"   # or omit for default

  # Frontend contribution (optional).
  # Exactly one of 'bundleUrl', 'bundleConfigMapRef', or 'bundleOciRef' must be set.
  frontend:
    bundleOciRef: "ghcr.io/acmecorp/sql-explorer-ui:1.2.0"
    # SRI hash is mandatory when bundleUrl is used; recommended otherwise.
    bundleIntegrity: "sha384-<hash>"
    extensionPoints:
      - type: route
        name: sql-explorer
        path: /sql-explorer              # rendered at /plugins/sql-explorer
        label: "SQL Explorer"
        icon: "database"
      - type: sidebarItem
        label: "SQL Explorer"
        icon: "database"
        routeName: sql-explorer
      - type: clusterDetailTab
        label: "Query"
        component: ClusterQueryTab       # exported name in the bundle

  # Backend contribution (optional).
  backend:
    # In-cluster mode:
    serviceRef:
      namespace: everest-plugins
      name: sql-explorer-svc
      port: 8080
    # External mode (use instead of serviceRef):
    # externalUrl: "https://sql-explorer.saas.example.com"
    # credentialsSecretRef: "sql-explorer-creds"

    # Operation modes. Omit any block the plugin doesn't need.
    # 'requestHandler' is implicit when 'serviceRef'/'externalUrl' is set
    # and 'daemon'/'eventSubscriptions' are absent.
    modes:
      requestHandler:
        # The HTTP path prefix on the backend that receives proxied calls
        # from /v1/plugins/{name}/*. Default "/".
        basePath: "/api"

      # Daemon mode — the host issues a long-lived, scoped service token to
      # the backend pod and expects it to run continuously.
      daemon:
        # Optional readiness probe path the host uses to consider the daemon
        # alive. Independent from k8s pod readiness — this is a host-level check.
        healthPath: "/healthz"

      # Event consumer — declares which event types the plugin intends to
      # consume from GET /v1/events. Purely informational: used for docs,
      # RBAC scoping, and as a hint to the host on whether to keep the
      # plugin's stream slot warm. The plugin is responsible for opening
      # and holding the SSE connection itself.
      eventConsumer:
        types:
          - database-cluster.created
          - database-cluster.ready
          - database-cluster.updated
          - database-cluster.deleted
          - backup.completed
        # Optional namespace filter. Omit to consume from all namespaces the
        # plugin's permissions allow.
        namespaces: ["team-alpha", "team-bravo"]

  # RBAC: what OpenEverest API resources this plugin needs to call.
  # Used both for the request-handler JWT (user-bound) and the daemon
  # service token (autonomous identity).
  permissions:
    - verb: read
      resource: database-clusters
    - verb: read
      resource: database-cluster-connection-details

  # CLI contribution (optional).
  cli:
    image: "ghcr.io/acmecorp/sql-explorer-cli:1.2.0"
    subcommand: "sql-explorer"
    description: "Interact with SQL Explorer from the terminal."
```

### 5.2 `PluginInstallation` (namespace-scoped)

Enables a plugin for a specific namespace and holds tenant-specific secrets
(e.g., external API keys). Multiple `PluginInstallation` CRs for the same plugin
allow per-team configuration.

```yaml
apiVersion: core.openeverest.io/v1alpha1
kind: PluginInstallation
metadata:
  name: sql-explorer
  namespace: team-alpha
spec:
  pluginName: sql-explorer      # references Plugin CR
  enabled: true
  # Optional: pass arbitrary config to the backend (mounted as env from Secret).
  configSecretRef: "sql-explorer-team-alpha-config"
```

---

## 6. UI Extension Points

Each extension point is a named, versioned slot in the React shell. The host
exports the full taxonomy from `@everest/plugin-sdk` as TypeScript types so
plugin authors get compile-time safety.

| Extension point | Where it appears | Props passed to component |
|---|---|---|
| `route` | Top-level React Router route at `/plugins/{name}/{path}` | `{ pluginName, params }` |
| `sidebarItem` | Main navigation sidebar | `{ navigate, currentPath }` |
| `clusterDetailTab` | Extra tab on a `DatabaseCluster` detail page | `{ cluster, namespace }` |
| `clusterAction` | Context-menu action in the clusters table | `{ cluster, namespace, onClose }` |
| `clusterCard` | Widget card on the cluster overview | `{ cluster, namespace }` |
| `globalDashboardWidget` | Card on the home / dashboard page | `{ namespaces }` |
| `settingsPanel` | Tab inside the Settings page | `{ currentUser }` |
| `themeOverride` | MUI theme override (logos, palette) | `{ defaultTheme }` — Phase 4 |

Extension points are **additive** — a plugin can register for multiple points. A
host version that does not recognise a point silently skips it.

---

## 7. Frontend SDK & Loading Model

### 7.1 Loading strategy

**Decision: dynamic ESM module loading (Headlamp model), not iframes.**

Rationale:
- Tight UX integration — shared MUI theme, React context, router state.
- Iframes break deep-link navigation, inject a separate auth session, and cannot
  contribute sidebar entries or theme overrides in a seamless way.
- Import maps let the host provide singleton instances of `react`, `react-dom`,
  `@mui/material`, and `react-router` so plugin bundles stay small and the host
  retains control over versions.

At shell startup:

```
1. GET /v1/plugins  →  [{ name, bundleUrl, extensionPoints }, ...]
2. For each enabled plugin:
     const mod = await import(bundleUrl)   // dynamic ESM import
     mod.default(pluginApi)               // calls register(api)
3. Plugin calls api.registerExtension("clusterDetailTab", MyComponent)
4. Shell renders registered components at the declared extension points.
```

### 7.2 `@everest/plugin-sdk`

New package at `ui/packages/plugin-sdk`. Public surface:

```ts
// Registration
registerExtension(point: ExtensionPoint, component: React.ComponentType<any>): void

// Hooks — bridge to the host's React context
useEverestApi(): EverestApiClient    // pre-authed HTTP client for /v1/...
useCurrentUser(): User
useCluster(id: string): DatabaseCluster | undefined
useNamespaces(): string[]
useRBAC(): { can: (verb: string, resource: string) => boolean }

// Re-exported singletons (resolved from host import map)
export { React, ReactDOM, MUI, ReactRouter }
```

### 7.3 Bundle requirements

Plugin authors produce a single ESM file with:

- Default export: `register(api: PluginApi): void`
- No bundled copies of `react`, `@mui/material`, or `react-router` — import them
  from the SDK re-exports so the import map resolves to the host's singleton.
- Target: `esnext` modules, no dynamic `require()`.

---

## 8. Backend Model

### 8.1 Proxy architecture

The OpenEverest API server exposes:

```
/v1/plugins/{pluginName}/*   →   proxied to plugin backend
```

The browser never calls a plugin backend directly. All requests are:

1. Authenticated by the host (session cookie / OIDC token validated).
2. RBAC-checked: the requesting user must have `use` on `plugin/{pluginName}`.
3. Forwarded to the backend with an `X-Everest-User` header containing a
   short-lived, signed JWT carrying `{ sub, namespaces, pluginName, exp }`.
4. Audit-logged by the host.

This gives the host complete visibility over plugin traffic and prevents plugins
from acting outside the user's own RBAC scope.

### 8.2 Backend authentication to OpenEverest

When a plugin backend needs to call OpenEverest APIs on behalf of the user, it
uses the `X-Everest-User` JWT as a bearer token. The host validates it and
enforces the user's own permissions — the plugin cannot escalate privilege.

### 8.3 Hosting modes

| Mode | How it works |
|---|---|
| **In-cluster** | `serviceRef` names a Kubernetes `Service`. Host resolves DNS internally. No public ingress needed. |
| **External** | `externalUrl` is an HTTPS endpoint. Credentials (e.g. API key) from a `Secret` are passed as `Authorization` header. |

### 8.4 Daemon mode

A daemon backend has no user session driving it. It runs continuously and
typically polls or reacts to events. To support this:

**Autonomous identity.** The host issues each daemon a long-lived **plugin
service token** — a JWT bound to the plugin name (not to any user) and scoped
to the permissions declared in `spec.permissions`. The token:

- Is mounted into the backend pod via a projected `Secret` named
  `<plugin-name>-token`, refreshed automatically before expiry (default TTL 24 h).
- Carries claims `{ sub: "plugin:<name>", permissions: [...], exp }`.
- Is accepted by the OpenEverest API as a bearer token. RBAC checks the
  declared permissions, not any user identity.
- Can read but **cannot write** to spec-001 resources, regardless of declared
  permissions (enforced by a hard-coded denylist — see §14 Q9).

**Why not just call the kube API directly?** A daemon could `watch` CRs natively
from inside the cluster. We deliberately route everything through the OpenEverest
API so:

- Plugins remain portable (work the same way against in-cluster and remote-SaaS
  OpenEverest deployments).
- The audit trail is centralised.
- The hard "no writes to spec-001 resources" guarantee can be enforced uniformly.

**Lifecycle.** When a `PluginInstallation` is created and the plugin declares
`backend.modes.daemon`, the host:

1. Generates the service token and writes it to the projected `Secret`.
2. Ensures the backend `Deployment` has at least one replica.
3. Polls `healthPath` to track liveness; surfaces status on the `PluginInstallation`.
4. On `PluginInstallation` deletion, scales to zero and revokes the token.

### 8.5 Event stream

The host exposes lifecycle events for resources it manages over a single
**stateless streaming endpoint**. Plugins (and any other client) consume the
stream by holding an open HTTP connection. There is no outbound push, no
delivery queue, and no per-plugin server-side state.

This design is a thin wrapper over Kubernetes' native watch mechanism:
OpenEverest's reconcilers already learn about state changes via kube watches
on the underlying CRs. The event endpoint normalises those watch events into
a stable, plugin-facing schema and streams them out. State lives in etcd,
where it already does — not in OpenEverest.

**Endpoint.**

```
GET /v1/events?since=<resourceVersion>&types=<csv>&namespaces=<csv>
   Accept: text/event-stream
```

Delivered as Server-Sent Events (SSE). Each event carries the underlying
Kubernetes `resourceVersion` as its cursor. The connection is held open by the
client; the host streams events as they happen.

**Initial event taxonomy** (extensible — plugins ignore types they don't know):

| Event type | Triggered when |
|---|---|
| `database-cluster.created` | A `DatabaseCluster` resource is created. |
| `database-cluster.ready` | A cluster transitions to `ready` state. |
| `database-cluster.updated` | Spec changes on an existing cluster (resize, version upgrade). |
| `database-cluster.deleted` | A cluster is deleted (post-finalizer). |
| `database-cluster.failed` | A cluster transitions to a failed state. |
| `backup.started` | A backup job starts. |
| `backup.completed` | A backup job completes successfully. |
| `backup.failed` | A backup job fails. |
| `restore.started` / `restore.completed` / `restore.failed` | Same for restores. |
| `instance.created` / `instance.deleted` | spec-001 `Instance` lifecycle. |

**Event envelope.**

```json
{
  "resourceVersion": "482719",
  "type": "database-cluster.deleted",
  "occurredAt": "2026-05-08T12:00:00Z",
  "namespace": "team-alpha",
  "resource": {
    "kind": "DatabaseCluster",
    "name": "orders-prod",
    "uid": "a1b2c3d4-...",
    "engine": "postgresql",
    "version": "15.5"
  },
  "prevState": { "phase": "ready" },
  "newState":  { "phase": "deleting" },
  "actor":     { "type": "user", "id": "alice@example.com" }
}
```

**Authentication.** The stream is authenticated like any other `/v1` endpoint
— a daemon plugin uses its service token (§8.4); a user-facing tool uses the
user's session token. The events the client receives are filtered by the
token's permissions and (if applicable) the namespaces the plugin is enabled
in via `PluginInstallation`.

**Why pull and not push?** A push model requires the host to track which
events have been delivered to which plugin, manage retry queues, and persist
state across restarts. By pulling, the plugin owns its cursor and the host
stays stateless. This mirrors how Kubernetes itself exposes change streams
and how every kube controller already handles restarts.

### 8.6 Catch-up & restart semantics

The stream uses `resourceVersion` as its cursor — the same mechanism
Kubernetes watch already provides. Plugins follow the standard kube-watch
restart pattern.

**Normal operation.** The plugin opens `GET /v1/events?since=<rv>` with the
last `resourceVersion` it persisted, receives any events that occurred since,
and then streams new events live. After processing each event the plugin
persists the new cursor (typically to its own local storage — PVC, embedded
KV, etc.).

**Stream drop.** If the connection drops (network, host restart, scale event),
the plugin reconnects with its last cursor. As long as the cursor is still
within the kube watch cache window (default 5 min, configurable on the kube
API server), the stream resumes from exactly that point with no gap.

**Cold start or stale cursor.** If the plugin has no cursor yet, or its
cursor is older than the watch cache, the plugin must:

1. Call `GET /v1/databases`, `/v1/backups`, etc. to snapshot current state.
2. Note the `resourceVersion` returned in the list response.
3. Open `GET /v1/events?since=<that rv>` to resume streaming.

This is exactly how the Kubernetes client-go informer handles the same
problem; the SDK provides a helper that wraps the dance.

**Implications.**

- No event store or dead-letter queue in the host. No state to back up, no
  state to migrate during host upgrades.
- The plugin must stay connected (or reconnect quickly) to capture events.
  Plugins that don't tolerate stream gaps must persist their cursor before
  acknowledging work, and must implement the snapshot-then-watch fallback.
- A plugin that holds the connection but processes events slowly will
  back-pressure the stream; the host will drop the slowest connections under
  memory pressure (advertised via a configurable per-connection buffer size).
  This is acceptable: dropped clients reconnect with `since=` and catch up.

---

## 9. RBAC Integration

OpenEverest already uses a Casbin policy model (`pkg/rbac`). Generic plugins
slot into it without new concepts.

### 9.1 Plugin install permission

Installing a plugin (creating a `Plugin` CR) requires the `create` verb on the
new resource type `plugins` at cluster scope. Only `admin` has this by default.

### 9.2 Per-user plugin access

A new resource type `plugin/{name}` is added to the Casbin model. Admins grant
the `use` verb to roles or individual users:

```
p, role:viewer, plugin/sql-explorer, use
```

The host checks this before proxying any request to the plugin backend and before
rendering extension-point components for the user.

### 9.3 Defense in depth

Even if a plugin backend attempts to call OpenEverest APIs directly, it can only
act under the user identity carried in the JWT, which is bound to that user's
existing RBAC policy. A plugin cannot read resources the user cannot read, and
cannot mutate resources the user cannot mutate.

For **daemon mode**, the autonomous plugin service token (§8.4) is bound to
the declared `spec.permissions` and to the plugin name — it is not a user
identity. It cannot be used to assume a user's privileges, and it is
unconditionally denied write access to spec-001 resources. The token is
rotated automatically and revoked on `PluginInstallation` deletion.

---

## 10. `everestctl` Integration

Go's `plugin` package is Linux-only and fragile across compiler versions. WASM
toolchains are promising but immature. The pragmatic choice is **subcommand-via-shellout**.

### How it works

The plugin manifest declares a `cli.image` (OCI). When the user runs:

```sh
everestctl plugin run sql-explorer -- query --db my-db "SELECT 1"
```

`everestctl`:
1. Pulls/caches the CLI image locally (or finds it in a local cache).
2. Execs the container, passing `--host`, `--token` (short-lived API token) and
   the user-supplied arguments via `stdin`/`stdout`.
3. Streams output back to the terminal.

Discovery:

```sh
everestctl plugin list                     # shows installed plugins with CLI support
everestctl plugin install <oci-ref>        # installs a plugin
everestctl plugin uninstall <name>
everestctl plugin enable   <name> -n <ns>  # creates PluginInstallation
everestctl plugin disable  <name> -n <ns>
```

---

## 11. Lifecycle & Distribution

### 11.1 OCI artifact as the canonical unit

A plugin is distributed as a single OCI artifact containing:

```
layers/
  manifest.yaml            # Plugin CR
  main.js                  # Frontend bundle (if any)
  annotations:
    org.opencontainers.image.title: "sql-explorer"
    org.everest.plugin.schema-version: "v1"
```

Backend and CLI images are separate OCI images referenced by digest from
`manifest.yaml`. This keeps the plugin artifact small and lets image layers be
cached independently.

### 11.2 Versioning

- Plugins are versioned with SemVer.
- `spec.compatibleHostVersions` is a semver range (same syntax as npm).
- The host rejects installation of a plugin whose range does not include the
  running host version.
- Plugins are upgraded independently of the host: `everestctl plugin upgrade <name>`.

### 11.3 Air-gapped environments

```sh
# Export on an internet-connected machine
everestctl plugin export sql-explorer --output sql-explorer.tar

# Import on an air-gapped machine
everestctl plugin install --from-tar sql-explorer.tar
```

The `--from-tar` flag pushes images into the in-cluster registry if one is
configured (e.g., Harbor), then creates the `Plugin` CR.

---

## 12. Security Considerations

| Concern | Mitigation |
|---|---|
| Malicious frontend bundle | Bundles served through host proxy (not a CDN); SRI hash in manifest verified before serving; strict CSP allows only host origin. |
| XSS via plugin code | Plugin JS runs in the same origin — normal XSS mitigations apply (React escaping, CSP). Considered acceptable given admin-only install gate. |
| Credential exfiltration | Plugin backends never receive raw DB credentials; only short-lived, scoped JWTs. DB connection details brokered on-demand via `/v1/databases/{id}/connection-details`. |
| Privilege escalation | All plugin API calls re-checked against the acting user's own RBAC. Plugin cannot escalate beyond the user's permissions. |
| In-cluster lateral movement | Plugin backend runs in its own `ServiceAccount` with a minimal `Role` auto-generated from `spec.permissions`. `NetworkPolicy` restricts egress to declared endpoints only. |
| Supply chain | Manifest must include OCI image digests. Host verifies cosign signatures when `spec.signatureVerification: true` is set on the cluster. |
| Secrets in manifests | Credentials for external backends are stored exclusively in `Secret` resources, never in the `Plugin` CR itself. |
| Admin-only install | Creating a `Plugin` CR requires cluster-admin RBAC. Enabling per namespace (`PluginInstallation`) requires namespace admin. Regular users only get `use`. |
| Daemon token theft | Plugin service token is mounted via a projected `Secret` (short-lived, auto-rotated, default TTL 24 h). Token is bound to plugin name + declared permissions, never a user identity, and revoked on `PluginInstallation` deletion. |
| Forged events | The event stream is delivered over the plugin's authenticated HTTPS connection to OpenEverest — there is no inbound push the plugin needs to validate. |
| Event-driven privilege escalation | Events are informational only — they do not authorise the plugin to perform any action. Any follow-up API call still goes through normal RBAC checks. |
| Slow event consumer / DoS on host | Per-connection bounded buffer; slow consumers are dropped and reconnect with `since=`. No unbounded queue grows in the host. |

---

## 13. Reference Architecture

```mermaid
graph TB
    subgraph Browser
        Shell[React UI Shell]
        Bundle[Plugin Frontend Bundle\nloaded via dynamic import]
    end

    subgraph OpenEverest Core
        API[API Server\n/v1/plugins/{name}/*]
        Auth[Auth / Session]
        RBAC[RBAC Engine\nCasbin]
        Proxy[Plugin Proxy]
        EventStream[GET /v1/events\nSSE over kube watch\nstateless]
        TokenSvc[Plugin Token Service\nautonomous identities]
        PluginAPI[GET /v1/plugins\nGET /v1/plugin-context\nGET /v1/databases/{id}/connection-details]
    end

    subgraph Kubernetes
        PluginCR[Plugin CR]
        PIcr[PluginInstallation CR]
        Secret[Secret\nCredentials]
    end

    subgraph Plugin Backend
        InCluster[In-Cluster Service]
        External[External HTTPS Endpoint]
        Daemon[Daemon Pod\nautonomous, no user session]
    end

    Shell -->|1. fetch plugin list| PluginAPI
    Shell -->|2. dynamic import| Bundle
    Bundle -->|3. API calls via SDK| API
    API --> Auth
    Auth --> RBAC
    RBAC -->|4. proxy if allowed| Proxy
    Proxy -->|X-Everest-User JWT| InCluster
    Proxy -->|X-Everest-User JWT| External
    InCluster -->|5. call back with JWT| PluginAPI
    Daemon -->|GET /v1/events?since=rv\nheld-open SSE| EventStream
    EventStream -->|kube watch| PluginCR
    TokenSvc -->|projected Secret\nplugin service token| Daemon
    Daemon -->|API calls with service token| API
    API --> PluginCR
    API --> PIcr
    API --> Secret
```

---

## 14. Answering Spec 003 Open Questions

> Spec 003 §7 raises ten open questions. This section records the decisions.

1. **Can generic plugins ship their own Kubernetes CRDs?**
   No. CRDs are reserved for spec 001 Providers. Generic plugins interact with
   Kubernetes exclusively through the OpenEverest API.

2. **What is the minimal backend surface OpenEverest must expose?**
   The existing v1 API, plus four new additions:
   - `GET /v1/plugins` — plugin discovery (list enabled plugins + bundle URLs).
   - `GET /v1/plugin-context` — current user identity, accessible namespaces.
   - `GET /v1/databases/{id}/connection-details` — brokered, short-lived credentials.
   - `GET /v1/events?since=<resourceVersion>` — stateless SSE event stream
     over kube watch (§8.5).
   No outbound push from the host; no separate "plugin API" needed; everything
   else goes through `/v1`.

3. **How do plugins get DB credentials?**
   OpenEverest brokers them on demand via `GET /v1/databases/{id}/connection-details`,
   gated by the user's existing RBAC. Tokens are short-lived (15 min). Plugins
   never cache or store credentials.

4. **Single plugin model or distinct shapes?**
   Single model with optional parts. A plugin that sets only `frontend` is a
   UI-only extension. One that sets only `backend` (with no `frontend`) is a
   headless integration or CLI tool. A full plugin sets both. `everestctl`
   already handles all cases.

5. **Can a plugin interact with spec 001 resources (`DatabaseCluster`, `Instance`)?**
   Read-only, via the OpenEverest API only. Plugins may `GET` cluster and instance
   resources through `/v1/...`; they may not call the Kubernetes API directly and
   may not mutate spec 001 resources.

6. **Is dynamic JS module loading viable for OpenEverest's frontend?**
   Yes. ESM dynamic `import()` is supported by all modern browsers. Import maps
   handle shared dependency deduplication. The React shell already uses Vite; the
   ESM loader is a natural extension. (See §7 for details.)

7. **What does the `everestctl` extension surface look like?**
   Subcommand-via-shellout to a plugin-declared OCI image. `everestctl plugin run
   <name> -- <args>` execs the container with a short-lived API token injected.
   (See §10 for details.)

8. **How are plugins versioned independently of the host?**
   SemVer on the plugin; `spec.compatibleHostVersions` semver range in the
   manifest; host enforces the range at install time. Plugins upgrade
   independently via `everestctl plugin upgrade`.

9. **Are there categories of plugins we would explicitly disallow?**
   Yes: plugins may not write to `DatabaseCluster`, `Instance`, `Provider`, or
   any other spec 001 CRs. The RBAC policy for the auto-generated plugin
   `ServiceAccount` excludes `create`, `update`, `patch`, and `delete` verbs on
   those resource types unconditionally. The same denylist applies to the
   daemon plugin service token (§8.4) regardless of what the manifest
   declares.

10. **Air-gapped / regulated environments?**
    Supported via `everestctl plugin export` / `--from-tar`. Images pushed to the
    in-cluster registry; no internet egress required after initial export.

---

## 15. Phased Roadmap

### Phase 1 — MVP

Deliver the minimal complete path for a plugin author to ship a UI page.

- `Plugin` CRD (no `PluginInstallation` yet — cluster-wide enable only).
- `GET /v1/plugins` discovery endpoint.
- Dynamic ESM loader in the React shell.
- `@everest/plugin-sdk` stub: `registerExtension`, `useEverestApi`.
- Extension points: `route` and `sidebarItem` only.
- Simple backend proxy (`/v1/plugins/{name}/*`) with session auth.
- Admin-only `Plugin` create gate in RBAC.
- `everestctl plugin install / list / uninstall`.

### Phase 2 — Multi-tenant & access control

- `PluginInstallation` CRD — per-namespace enable/disable + config secret.
- `plugin/{name}` resource in Casbin model — per-user `use` grants.
- In-cluster backend `serviceRef` discovery (DNS resolution, health check).
- Credentials broker: `GET /v1/databases/{id}/connection-details`.
- `GET /v1/plugin-context` endpoint.

### Phase 3 — Daemon mode & event stream

Unlocks the metering / billing / audit / external-sync class of plugins.

- Plugin token service — mint, mount, and rotate autonomous service tokens.
- Daemon mode: host-managed `Deployment` lifecycle, health tracking, status
  conditions on `PluginInstallation`.
- `GET /v1/events` SSE endpoint backed by a kube watch on the relevant CRs
  (`DatabaseCluster`, `Instance`, `Backup`, `Restore`).
- Event normaliser: maps kube watch events into the plugin-facing schema (§8.5).
- SDK helpers for the snapshot-then-watch restart pattern (§8.6).
- Hard denylist on writes to spec-001 resources from daemon tokens.
- **No event store, no delivery worker, no DLQ in the host.** State lives in
  etcd; cursor lives in the plugin.
### Phase 4 — Rich UI extension points & distribution

- Extension points: `clusterDetailTab`, `clusterAction`, `clusterCard`,
  `globalDashboardWidget`, `settingsPanel`.
- `everestctl plugin run` shellout for CLI extensions.
- OCI artifact packaging: `everestctl plugin export` / `--from-tar`.
- Bundle SRI verification + optional cosign signature check.
- Auto-generated `ServiceAccount` + `Role` + `NetworkPolicy` for backend pods.

### Phase 5 — Polish & ecosystem

- `themeOverride` extension point (branding / logos).
- Plugin-to-plugin event bus (opt-in pub/sub via the SDK).
- External backend support (`externalUrl` + `credentialsSecretRef`) for daemon
  and event-consumer modes (the external backend opens a held SSE connection
  back to the host — no inbound push required).
- Marketplace / catalog UI (`Plugin` browser in the web UI).
- `everestctl plugin upgrade` with version-range enforcement.

---

## 16. Open Questions

1. **OCI media type**: custom `application/vnd.openeverest.plugin.v1` media type vs.
   reusing a Helm chart. Custom type is cleaner semantically; Helm is more
   familiar for GitOps workflows. Needs decision before Phase 4.

2. **Bundle hosting**: serve plugin bundles from a `ConfigMap` (size-limited to
   ~1 MB after compression) or store them in an in-cluster object store / PVC?
   For Phase 1 a `ConfigMap` suffices; Phase 4 distribution needs OCI artifact
   storage or an in-cluster registry.

3. **React shell router sandboxing**: expose the host React Router `<Outlet>` to
   plugins directly, or wrap plugin routes in a sandboxed sub-router with a
   restricted history scope? Direct exposure is simpler; sandboxing gives better
   isolation for plugin navigation errors.

4. **Plugin-to-plugin communication**: should plugins be allowed to call each
   other's backends via `/v1/plugins/{otherName}/*`? If yes, the requesting
   plugin must have `use` on the target plugin and carry a valid user session.
   Decision deferred to Phase 5.

5. **Bundle size / performance**: no size limit defined yet. Large bundles delay
   shell startup. Consider a lazy-load model where extension-point components are
   loaded only when the user navigates to the relevant page (route-level code
   splitting within the plugin bundle).

6. **Event retention window**: bounded by the kube watch cache window (default
   5 minutes on the kube API server, configurable). Plugins that have been
   disconnected longer must do the snapshot-then-watch fallback (§8.6). No
   separate retention policy needed in OpenEverest — etcd is the source of
   truth and the watch cache covers the gap.

7. **Event delivery model**: pull-based SSE stream over kube watch (decided,
   §8.5). The earlier draft proposed a push-based bus with a server-side
   queue; that introduced unwanted state in the host (delivery log, retry
   queue, DLQ, durability across restarts). The pull model puts the cursor on
   the plugin side and keeps the host stateless. Open sub-question: do we
   ever need a push variant for SaaS / off-cluster consumers that cannot hold
   an inbound connection? If so, a small webhook bridge plugin ("event
   forwarder") could be built on top of the pull stream without bringing
   state into the core.

8. **Synchronous (pre-) hooks**: this design covers post-hoc, fire-and-forget
   events only. Should we also support **synchronous validating hooks** (e.g.,
   "before creating a cluster, ask the policy plugin to approve")? That class
   of plugin sits closer to a Kubernetes admission webhook and may justify a
   separate spec; explicitly out of scope for v1.

9. **Daemon scaling**: does the host enforce single-replica daemons (simpler,
   no distributed-lock concerns) or allow plugin authors to declare a replica
   count? Multi-replica daemons need event delivery to be load-balanced across
   replicas in a partition-aware way — non-trivial. Single-replica is the
   recommended Phase 3 starting point.

---

## 17. Implementation Cost & Plugin Author DX

This section captures the engineering reality of delivering this design in the
existing OpenEverest codebase, and what's needed for plugin authors to actually
enjoy building plugins.

### 17.1 Impact on the existing codebase

The host stack is already well-shaped for this work — Echo + oapi-codegen for
the API, controller-runtime for CRDs, Casbin for RBAC, Vite + React Router v6
for the UI, Cobra for `everestctl`. Most of the plugin system slots in cleanly;
only two areas require genuinely new infrastructure.

**Existing infrastructure that we reuse as-is:**

- **HTTP / middleware** ([internal/server/](internal/server/)) — Echo handler chain
  (`newHandlerChain(valH, rbacH, k8sH)`) is the natural place to drop in the
  plugin proxy and discovery handler.
- **Auth / JWT** ([pkg/session/](pkg/session/), [pkg/oidc/](pkg/oidc/)) — already mints and
  validates JWTs; the plugin token service can reuse the same signing key
  infrastructure.
- **CRDs / controllers** ([api/core/v1alpha1/](api/core/v1alpha1/), [internal/controller/](internal/controller/))
  — kubebuilder-style; adding `Plugin` and `PluginInstallation` follows the
  same pattern as existing CRDs.
- **RBAC** ([pkg/rbac/](pkg/rbac/), [data/rbac/model.conf](data/rbac/model.conf)) — Casbin model
  already supports glob matching on the resource field. Adding `plugin/{name}`
  is a constants change plus a few policy lines, no model rewrite.
- **CLI** ([commands/](commands/)) — Cobra; `everestctl plugin ...` slots in alongside
  existing subcommand groups like [accounts/](commands/accounts/).
- **UI build** ([ui/apps/everest/](ui/apps/everest/)) — Vite is ESM-native, dynamic
  `import()` works out of the box.

**Genuinely new infrastructure that we have to build:**

1. **Event stream endpoint**. OpenEverest today has no plugin-facing change
   stream. The Phase-3 work is a `GET /v1/events` SSE handler that opens a
   kube watch on the relevant CRs (`DatabaseCluster`, `Instance`, `Backup`,
   `Restore`), normalises each watch event into the plugin-facing schema
   (§8.5), filters by the caller's RBAC and namespace scope, and writes the
   result as SSE frames. **No event bus, no delivery worker, no queue, no
   store** — state lives in etcd where it already does, and the cursor lives
   in the plugin. The reconcilers themselves don't need any modification
   because the kube watch sees the same status transitions they do.
2. **Dynamic UI route registration**. React Router v6 prefers compile-time
   route trees. The pragmatic approach is a single wildcard route
   `/plugins/:pluginName/*` that dispatches to a `<PluginHost>` component
   which mounts whichever extension component the plugin registered. Sidebar
   items are pulled from React state populated at startup. Some Vite plumbing
   for an import map is also needed so plugin bundles can `import 'react'`
   and resolve to the host singleton.

**Estimated change scope** (rough, not a commitment):

| Area | New / modified files | Approx. LoC |
|---|---|---|
| `Plugin` + `PluginInstallation` CRDs ([api/core/v1alpha1/](api/core/v1alpha1/)) | new | 300–500 |
| Plugin reconciler ([internal/controller/](internal/controller/)) | new | 400–600 |
| Plugin proxy + discovery + token service ([internal/server/](internal/server/)) | new | 500–800 |
| `GET /v1/events` SSE handler + kube-watch normaliser (`internal/server/events.go`) | new | 250–400 |
| RBAC additions ([pkg/rbac/](pkg/rbac/)) | modified | 100–150 |
| `everestctl plugin ...` ([commands/](commands/)) | new | 400–600 |
| `@everest/plugin-sdk` ([ui/packages/](ui/packages/)) | new package | 400–600 |
| UI dynamic loader + `<PluginHost>` ([ui/apps/everest/](ui/apps/everest/)) | modified | 300–500 |
| Tests + fixtures | new | 800–1200 |
| **Total** | | **~3,450–5,350 LoC** |

Dropping the durable event-bus subsystem trims roughly 700–1,100 LoC and a
significant chunk of operational complexity (no store to back up, no DLQ to
monitor, no delivery state to migrate during host upgrades). The phased
roadmap (§15) defers the most invasive remaining parts (daemon mode, dynamic
UI loader) to Phase 3+ so a Phase-1 MVP can ship with route + sidebar
extension points only and validate the architecture.

### 17.2 Highest-risk integration points

1. **Slow event consumers**. A plugin that holds the SSE connection but
   processes events slowly back-pressures the host's per-connection buffer.
   **Mitigation**: bounded buffer per connection (configurable, default ~1k
   events); drop the slowest connections; clients reconnect with `since=`
   and resume. No unbounded queue can grow on the host.
2. **Watch cache window**. Plugins disconnected longer than the kube watch
   cache (default 5 min) must do a snapshot-then-watch fallback to resume.
   **Mitigation**: ship the snapshot-then-watch helper as a first-class SDK
   function (§8.6); document the pattern as the standard restart flow.
3. **RBAC scoping for `plugin/{name}`**. The current Casbin model uses glob
   matching on the resource field. A naive `plugin/*` policy line would grant
   access to all plugins. **Mitigation**: use specific per-plugin policy
   lines, not wildcards, and reject wildcard plugin policies in the policy
   editor.
4. **Dynamic React Router**. Workable but easy to get wrong (history scope,
   error boundaries, nested routes). **Mitigation**: prototype the
   `<PluginHost>` wrapper early in Phase 1.
5. **API backward compatibility**. Once plugins ship, breaking response
   shapes (including the event envelope in §8.5) breaks plugins.
   **Mitigation**: lock the plugin-facing subset of `/v1` early (discovery,
   plugin-context, connection-details, events) and treat it as a stability
   boundary; version the event envelope explicitly.

### 17.3 Plugin author developer experience

The architecture is only as valuable as the number of plugins built on it. If
authoring a plugin requires reading 80 pages of docs, hand-writing a YAML CRD
manifest, configuring an import map, and standing up a local OpenEverest
cluster to test against — nobody will write plugins.

The DX investments below are **as important as the architecture itself** and
should be tracked alongside the implementation roadmap.

**P0 — minimum required for any external plugin author to succeed:**

- **`everestctl plugin scaffold <name>`** — generates a working plugin in one
  command: `manifest.yaml`, a TypeScript frontend stub with the SDK wired
  up, an optional Go backend stub, a `Makefile`, and a sample
  `PluginInstallation`. This is the single highest-leverage DX item.
- **Plugin SDK with strong types** — every extension point's props are typed
  in `@everest/plugin-sdk`. Event payloads are typed. The `EverestApi` client
  is generated from the OpenAPI spec so the client and server types can never
  drift. Plugin authors never write `any`.
- **A "Hello World" reference plugin** in the
  [openeverest/plugin-examples](https://github.com/openeverest/plugin-examples)
  repo (to be created) covering a UI-only plugin, a daemon plugin, and an
  event-subscriber plugin. Each example is a working, tested codebase, not a
  snippet in docs.
- **A working dev-mode loop**: `everestctl plugin dev` runs the plugin's Vite
  dev server and patches the host's plugin discovery to point at
  `http://localhost:3001/main.js`. Hot-reload works in the browser without
  rebuilding/redeploying anything.

**P1 — significantly improves authoring quality:**

- **Manifest linter**: `everestctl plugin lint` validates the manifest
  against the CRD schema, checks that declared permissions exist in the
  OpenAPI spec, verifies SemVer ranges, and warns on unsigned bundles.
- **Mock SDK for unit tests**: `@everest/plugin-sdk/testing` exports
  `mockEverestApi()`, `mockUser()`, `renderInPluginHost()` so plugin
  components can be unit-tested in isolation without spinning up a cluster.
- **Backend SDK packages** (`@everest/plugin-backend-sdk` for Node, plus a Go
  module): wraps JWT verification, event signature verification, the
  service-token bootstrap, and the OpenEverest API client. Plugin backends
  shouldn't have to reimplement these.
- **Compatibility check at install time**: `everestctl plugin install`
  refuses to install plugins whose `compatibleHostVersions` excludes the
  running host, with a clear error message.

**P2 — ecosystem-grade polish:**

- **CI matrix template**: a reusable GitHub Actions workflow that runs a
  plugin's tests against multiple OpenEverest versions in kind clusters.
- **E2E test harness**: a Playwright fixture (`@everest/plugin-sdk/e2e`) that
  installs the plugin under test into a kind cluster and exposes the host UI
  for assertions.
- **Versioned event schema**: every event payload carries a `schemaVersion`
  field; the SDK exposes per-version typed accessors so plugins can adopt
  new event versions incrementally.

**Tooling cost estimate:**

| Item | Priority | Effort |
|---|---|---|
| `everestctl plugin scaffold` | P0 | 1–2 weeks |
| Plugin SDK + generated types + mock utils | P0 | 2–3 weeks |
| Reference plugins repo + docs | P0 | 1–2 weeks |
| `everestctl plugin dev` (hot-reload) | P0 | 2–3 weeks |
| Manifest linter | P1 | 1–2 weeks |
| Backend SDK (Node + Go) | P1 | 1–2 weeks |
| Install-time compatibility check | P1 | < 1 week |
| E2E test harness (Playwright fixture) | P2 | 2–3 weeks |
| CI matrix template | P2 | 1 week |
| Versioned event schema | P2 | 1 week |

Total DX investment: **roughly 4–6 person-weeks for P0, plus another 4–6 for
P1, on top of the architecture work itself**. Treating DX as a first-class
deliverable — rather than a "we'll write some docs later" item — is the
single biggest predictor of plugin ecosystem adoption.

### 17.4 Recommended sequencing

1. Land Phase 1 + the P0 DX bundle together. A scaffolder, SDK, and one
   reference plugin shipping with the MVP make the difference between a
   feature nobody uses and a feature people start building on immediately.
2. Use the reference plugins as the canary for Phase 2 / Phase 3 changes —
   if the metering reference plugin breaks under a daemon-mode change, the
   spec is wrong, not the plugin.
3. Don't ship Phase 3 (daemon mode + events) without the corresponding SDK
   updates and a working metering reference plugin. The architecture and the
   reference implementation must land together.
