# performance-testing plugin (PoC)

A generic OpenEverest plugin (`docs/process/generic-plugins-design.md`)
that runs [sysbench](https://github.com/akopytov/sysbench) benchmarks
against Everest-managed PostgreSQL databases as ephemeral Kubernetes Jobs,
and retains results for comparison over time. Built for CNCF LFX
Mentorship Term 3 issue
[openeverest/openeverest#2464](https://github.com/openeverest/openeverest/issues/2464).

## Scope of this PoC

PostgreSQL + sysbench only, against a real `DatabaseCluster` (the
`everest.percona.com` resource type every real OpenEverest database
actually is today — see "Design notes" below). MySQL/pgbench and
MongoDB/YCSB are the natural next engines once this mechanism is proven,
not built here.

## Layout

```
performance-testing/
├── Dockerfile
├── backend/              # Go backend (plain net/http, no framework)
│   ├── main.go
│   ├── handlers.go       # POST/GET /api/runs
│   ├── job.go             # builds the benchmark batchv1.Job
│   ├── parser.go           # parses sysbench's text report
│   ├── store.go             # in-memory run history
│   └── kube.go               # client-go clientset bootstrap
├── charts/performance-testing/  # Helm chart: Plugin CR, Deployment,
│                                  # Service, ServiceAccount, Role, RoleBinding
└── manifest/sample-plugin.yaml   # minimal Plugin CR for kubectl apply testing
```

## API

- `POST /api/runs` — `{"instance": "pg-poc", "namespace": "everest", "workload": "smoke"}`.
  Returns `{"id": "run-..."}` immediately; the Job runs asynchronously.
  Workloads: `smoke`, `read-heavy`, `write-heavy`, `mixed-oltp`.
- `GET /api/runs/{id}` — status + parsed throughput/latency once the Job finishes.
- `GET /api/runs` — history, most recent first.

## Design notes

**Credentials.** `GET /clusters/{cluster}/namespaces/{namespace}/instances/{instance}/connection`
(`internal/server/handlers/k8s/instance.go`) only brokers credentials for
the newer spec-001 `Instance` type. Every real database on a default
OpenEverest v2 install — including the `pg-poc` this PoC was verified
against — is still the older `DatabaseCluster` type
(`databaseclusters.everest.percona.com`), which has no equivalent broker
anywhere in `release-2.0`. This plugin reads the target's credentials
Secret (`everest-secrets-<instance>`) directly, scoped by its own RBAC to
a `get` on `secrets` in its install namespace — the same pattern
`internal/controller/backup/backup_controller.go` already uses for
`Instance`-based backups, just applied to the resource type that doesn't
have a broker yet. See `backend/job.go`'s `credentialsSecretName` doc
comment for the full reasoning. Building the missing
`DatabaseCluster`-equivalent broker is real, well-scoped follow-up work.

**Job isolation.** The benchmark Job declares a soft `podAntiAffinity`
against the target instance's `app.kubernetes.io/instance` label, so it
prefers scheduling away from the database's own pods (the "observer
effect" question raised publicly on #2464). Not verified against actual
multi-node placement — `everest-poc`, the cluster this was tested on, is
single-node, so only the Job's *declared* affinity was confirmed, not
enforced placement.

**Result storage.** In-memory only, ephemeral by default — matches the
maintainers' explicit rejection of ConfigMap/PVC-by-default storage in the
#2464 issue thread. A durable, opt-in backend is a straightforward
`RunStore` implementation away, not built here.
