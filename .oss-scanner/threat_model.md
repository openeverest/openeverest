# OpenEverest threat model

OpenEverest is an open-source platform for running and managing databases (PostgreSQL, MySQL/MariaDB,
MongoDB and others, via pluggable "providers") on Kubernetes. It provides a web UI, a REST API and a CLI for
creating database instances, backups, restores and monitoring. The API server runs inside the cluster with a
Kubernetes ServiceAccount that can read and write database resources and their Secrets across every namespace
OpenEverest manages, and it hands out database credentials to authorized users. A flaw can therefore lead to
theft of database credentials and data, or to Kubernetes workload execution, across every managed namespace.

Project security policy: `SECURITY.md`.

## Components and where untrusted input enters

- **API server** (`cmd/main.go`, `internal/server/`, Echo framework). This is the main network-facing component.
  It serves the REST API defined in `api/openapi/http-api.yaml` under `/v1`, the embedded UI (`public/`) and
  plugin assets and backend proxies. Treat every HTTP request, path parameter, query, header, body, JWT and
  OIDC response as attacker-controlled. Request handling is a chain (`internal/server/everest.go`
  `setHandlers`): RBAC (`internal/server/handlers/rbac`), then validation
  (`internal/server/handlers/validation`), then the Kubernetes implementation (`internal/server/handlers/k8s`).
  Any route or operation that reaches the `k8s` handler without the RBAC check for the right
  cluster/namespace/object is a vulnerability.
  - Unauthenticated routes are the OpenAPI operations marked `security: []` (`/v1/auth/token` for password
    and refresh-token login, `/v1/auth/revoke`, `GET /v1/settings`), the UI (`/`, `/static/*`) and
    `GET /v1/clusters/{cluster}/plugins/{name}/*`, which is unauthenticated so the UI can `import()` plugin
    bundles. That GET route goes through the same plugin proxy as the authenticated backend routes, so reaching
    plugin backend APIs, injected credentials or other upstreams through it without a token is in scope. Check
    the JWT middleware's skipper (`newSkipperFunc`) and the hand-registered routes in
    `internal/server/everest.go`.
  - `GET /v1/events` is an authenticated event stream registered outside OpenAPI validation and the handler
    chain. Events about resources the user is not allowed to see must not be sent to them.
  - `GET .../instances/{instance}/connection` returns database host, username and password. It is the most
    sensitive read endpoint.
  - `.../secrets` endpoints must never return `Secret.data`. List and get return metadata only.
  - The validation handler dials endpoints supplied in requests (backup storage, monitoring destinations) with
    credentials supplied in requests. It runs after RBAC on purpose; SSRF that reaches it without the
    corresponding create/update permission, or that leaks responses or credentials, is in scope.
- **Authentication** (`pkg/session`, `pkg/accounts`, `pkg/kubernetes/accounts.go`, `pkg/oidc`,
  `internal/server/auth.go`, `internal/server/session_failures_storage.go`). Built-in accounts are stored in
  the `everest-accounts` Secret with PBKDF2-SHA256 password hashes. Sessions are RS256 JWTs signed with the key
  in the `everest-jwt` Secret (`/etc/jwt/id_rsa`). Revoked tokens live in the `everest-blocklist` Secret;
  tokens issued before a password change must be rejected. OIDC validates tokens from an
  administrator-configured issuer. Login brute-force protection is an in-memory, per-process limiter. Token
  forgery, algorithm confusion, accepting expired, revoked or wrong-audience tokens, username/subject confusion
  between built-in, OIDC and `plugin:` subjects, and limiter bypass are all in scope.
- **Authorization** (`pkg/rbac`, `data/rbac/model.conf`). Casbin policies come from the `everest-rbac`
  ConfigMap. Objects are scoped as `cluster/namespace/name`. **The core security boundary is RBAC plus
  namespace scoping**: a user must only see or change the resources, namespaces and secrets their policy
  grants, and only in namespaces OpenEverest manages. The server acts with its own privileged ServiceAccount,
  so any RBAC gap is a confused-deputy escalation. RBAC is disabled by default in the Helm chart
  (`server.rbac.enabled: false`), which makes every authenticated user an administrator. Evaluate
  authorization findings with RBAC enabled; with RBAC disabled, only the authentication boundary applies.
- **Plugins** (`internal/server/plugins.go`, `internal/server/plugin_api.go`, `pkg/plugintoken`,
  `internal/tokenregistry`, `api/extensions/v1alpha1/plugin_types.go`). Plugins add UI bundles (loaded
  into the UI origin) and backend services that the API server reverse-proxies to (`serviceRef` or
  `externalURL`), with an optional injected bearer token. Plugin service tokens are RS256 JWTs with a
  `plugin:` subject. In scope: one plugin reaching another plugin's backend or token, proxy path traversal
  that reaches a different service or host, leaking the user's Everest token or the injected credential to a
  plugin backend that should not get it, and plugin tokens being accepted with more privilege than intended.
- **Controller manager** (`cmd/controller`, `internal/controller/{backup,monitoring,extension}`) and
  **admission webhooks** (`internal/webhook`). These reconcile the CRDs in `api/` (instances, backups,
  restores, backup classes and storages, monitoring configs, providers and plugins). Backup and restore jobs
  build Pod specs and container commands from CR fields (`internal/controller/backup/backup_controller.go`,
  `restore_controller.go`). Fields a namespace-level user controls (Backup, Restore, BackupStorage,
  MonitoringConfig, instance specs) must not let them inject commands, args, env, volumes, images or service
  accounts beyond what the administrator-defined BackupClass allows, or reach Secrets in other namespaces.
- **Provider runtime** (`provider-runtime/`). This SDK is used by database-provider operators. It runs an HTTP
  server (`provider-runtime/server/server.go`) with `/validate`, `/healthz` and `/readyz`; treat
  validation request bodies as untrusted.
- **CLI** (`cmd/cli`, `commands/`, `pkg/cli`). `everestctl` installs, upgrades and manages OpenEverest
  using Helm and the user's kubeconfig. A malicious API server or Helm repository response that causes local
  file writes, command execution, or leaks the kubeconfig or admin password is in scope.
- **UI** (`ui/`, React/TypeScript). Access tokens are kept in `localStorage`, so any XSS means token theft.
  Data shown in the UI that comes from Kubernetes objects, database providers, monitoring and plugins (names,
  labels, annotations, status messages, provider-supplied UI schemas) is untrusted. Stored or reflected XSS
  through that data is in scope. Plugin bundles are trusted code by design (see below).

## Attackers to consider

- Unauthenticated network clients of the API server.
- Authenticated users with limited RBAC: read-only users, users scoped to some namespaces or instances,
  and users who may create database instances or backups but not manage storage, monitoring or settings.
- An installed plugin's backend, trying to escalate beyond the plugin's token scope or to reach other plugins.
- Responders to outbound requests: OIDC provider, S3-compatible backup storage, monitoring endpoints, version
  service and Helm repositories.
- Workloads running inside managed namespaces, such as a compromised database pod.

## Trusted / out of scope

- OpenEverest administrators (`role:admin` or equivalent policy). So is anyone who can change the
  `everest-rbac` ConfigMap, the `everest-settings` ConfigMap, the `everest-accounts`, `everest-jwt` or
  `everest-blocklist` Secrets, or anything in the OpenEverest system namespace.
- Anyone with direct Kubernetes API access to create or modify OpenEverest CRDs. Kubernetes RBAC is their
  boundary, not ours. Reports must start from the OpenEverest API or from a lower-privileged position.
- Installing a plugin or provider is an administrator action, and plugin UI bundles run as trusted code in the
  UI origin by design. Report this only if a non-administrator can register, replace or alter a plugin,
  provider or BackupClass.
- Upstream database operators and engines (the Percona and MariaDB operators and others) and their container
  images. Provider implementations live in separate repositories.
- Documented insecure options chosen by an administrator, such as `insecureSkipVerify`/`skipTLSVerify` on
  backup storage, monitoring or PMM endpoints, or disabling TLS on the server.
- Vulnerabilities in dependencies (Go modules, npm packages, Helm charts) that OpenEverest code cannot reach.
  Dependency CVEs reported only from a version number are out of scope.
- `test/`, `api-tests/`, `cli-tests/`, `hack/`, `dev/`, `docs/`, Storybook, mocks (`*mock*.go`) and generated
  code (`*.gen.go`, `ui/**/api/`). Generated code is in scope only when the generator's input in `api/` is wrong.
- Self-XSS, missing security headers without a concrete exploit, login-limiter weaknesses that require
  running multiple replicas, and username enumeration through timing alone.

## How to exercise it

- `/usr/local/bin/everest` (API server), `/usr/local/bin/manager` (controller) and `/usr/local/bin/everestctl`
  (CLI) are built from `/src`. The UI is built into `/src/public/dist` and embedded in the API server.
- There is no Kubernetes cluster in the image. Prefer reproducers written as Go unit tests that use the fake
  controller-runtime client, following existing tests:
  - Handler-level tests: `internal/server/handlers/rbac/*_test.go` (RBAC decisions per endpoint),
    `internal/server/handlers/validation/*_test.go` and `internal/server/handlers/k8s/*_test.go`.
  - Server and middleware: `internal/server/everest_test.go`, `middlewares_test.go`, `auth_test.go`.
  - RBAC policy evaluation: `pkg/rbac/*_test.go`. Sessions and tokens: `pkg/session/*_test.go`,
    `pkg/plugintoken/*_test.go`.
- All Go test binaries are pre-compiled; `go test ./internal/server/...` and similar run offline. Go is set to
  `GOPROXY=off`.
- UI unit tests: `cd ui/apps/everest && pnpm test:unit` (Vitest, jsdom). The browser suite
  (`test:browser`) needs Chromium and does not run offline.
- The scan has no network access, so code that contacts OIDC providers, S3, Helm repositories, the version
  service or telemetry fails offline. That is expected and not a finding.

## How we rate severity

We use CVSS 3.1 and publish fixes through GitHub Security Advisories. As guidance:

- **Critical**: unauthenticated remote code execution; authentication bypass or JWT forgery on the API server;
  unauthenticated access to database connection credentials, Secrets, or the JWT signing key; any
  unauthenticated path to creating workloads in the cluster.
- **High**: an authenticated non-admin user escaping RBAC or namespace scoping. Examples: reading another
  namespace's or instance's connection credentials or Secret contents, acting on unmanaged namespaces, gaining
  admin privileges, or making the server create or modify resources outside their permissions. Also high:
  command, argument or Pod-spec injection into backup/restore jobs from non-admin-controlled fields; stored XSS
  in the UI from Kubernetes, provider or monitoring data; SSRF that returns response bodies or leaks
  credentials from internal services; a plugin token or plugin backend gaining user or admin privileges.
- **Medium**: disclosure of non-secret metadata across RBAC boundaries; blind SSRF; token revocation or
  password-change invalidation that can be bypassed; open redirects or CSRF on state-changing endpoints;
  denial of service of the API server or controller by an authenticated non-admin user; reflected XSS that
  needs user interaction.
- **Low**: unauthenticated or authenticated DoS that needs unusual configuration, issues that need an
  administrator to choose an insecure setting, and defense-in-depth or hardening findings with no
  demonstrated exploit.

## How we would like reports

Keep reports short. State the affected component and endpoint or CRD, the attacker's required access (none,
read-only user, namespace-scoped user, plugin backend, workload in a managed namespace), and give a
reproducer, ideally a Go unit test next to the affected code. Patches should be minimal and follow the existing
handler-chain pattern (RBAC checks in `internal/server/handlers/rbac`, input checks in
`internal/server/handlers/validation`). Do not report theoretical issues without a concrete path from
attacker input to impact.
