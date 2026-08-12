# MinIO object storage provider (proof of concept)

Tracks [openeverest/openeverest#2255](https://github.com/openeverest/openeverest/issues/2255)
("Support for object storage engines (MinIO, SeaweedFS)"). This package is a
proof-of-concept implementation of `provider-runtime`'s `ProviderInterface`
(`provider-runtime/controller`) for MinIO, built to validate the design
before proposing it as real scope.

**Status: proof of concept, not production-ready.** It is functional and has
been verified end-to-end against a real MinIO Operator install (see
"Verified behavior" below), but several deliberate scope cuts (see "Known
limitations") mean it is not a drop-in replacement for the `Expected
Outcomes` listed on issue #2255 yet.

## What it does

- `Sync` renders a real MinIO Operator `Tenant` custom resource
  (`minio.min.io/v2`) from an `Instance`'s `server` component
  (`replicas`/`storage`/`version` map onto the Tenant's pool, storage
  request, and image), and applies it.
- It also creates the root-credentials `Secret` the Tenant requires
  (`spec.configuration.name`) and declares a dedicated backup bucket on the
  Tenant (`spec.buckets`).
- `Status` reads the Tenant's `status.currentState`/`availableReplicas`
  back onto the `Instance`, and — once the Tenant is `Initialized` —
  registers it as a `BackupStorage` (`api/backup/v1alpha1`) so other
  OpenEverest-managed databases can use it as a backup target, reusing the
  existing `BackupStorage` mechanism rather than inventing a new one.
- `Cleanup` is a no-op: every object this provider creates is owned by the
  `Instance` (via `controller.Context.Apply`'s automatic owner reference),
  so Kubernetes garbage collection handles teardown.

It intentionally does not import `github.com/minio/operator`: that module
predates Go's module graph pruning, so importing even its `pkg/apis`
subpackage drags in the MinIO server binary's own dependency graph. Instead,
`tenant_types.go` hand-declares the subset of the real `Tenant` CRD this
provider needs, with field names/tags/group-version taken verbatim from the
real CRD so the objects on the wire are indistinguishable from ones built
with the upstream types.

## Verified behavior

Against a real `kind` cluster with the MinIO Operator installed via Helm:

- `kubectl apply -f manifest/sample-instance.yaml` → the provider creates a
  real `Tenant`, which reaches `Initialized`/`green` health.
- `Instance.status.phase` correctly reaches `Ready`.
- A real S3 client (`mc`) round-trips data (`mb`/`cp`/`ls`/`cat`) through
  both the Tenant's primary bucket and the auto-created backup bucket.
- The auto-registered `BackupStorage` is immediately usable by a real S3
  client authenticating with its own (non-root-Secret) credentials
  reference — i.e. it is a real, writable backup target, not just a
  declared record.
- `go test ./providers/minio/...` covers the same scenarios (Tenant
  rendering with defaults and explicit overrides, credentials-secret
  idempotency across repeated reconciles, and the Provisioning → Ready →
  BackupStorage-registered status transitions) against a fake client.

## Known limitations / open design questions

- **Single `server` component, single pool.** No support yet for MinIO's
  distributed-pool topology, multiple component types, or per-pool
  configuration.
- **Root credentials reused for the backup bridge**, instead of a dedicated
  per-Instance IAM user/policy. Simpler for a PoC, not appropriate long
  term — see "Future enhancements" below.
- **No bucket/user/policy management surface.** `Instance` has no
  first-class concept of buckets or IAM beyond the one backup bucket this
  provider provisions for itself; `Instance.spec.parameters` (validated
  against the Provider's `parametersSchema`) is the likely mechanism, not
  yet implemented.
- **No scaling/upgrade path tested.** Changing `Instance.spec.components.
  server.replicas` or `.version` after initial creation and confirming the
  Tenant reconciles cleanly has not been exercised.
- **No watch on the owned `Tenant`.** Status only recomputes on the next
  `Instance` change or the reconciler's periodic resync, not immediately
  when the Tenant's own status changes — a real (if minor) latency gap.
- **No real database has backed up through this `BackupStorage` via
  OpenEverest's own backup path** — verified so far with a raw S3 client
  standing in for one.
- **Single provider (MinIO).** SeaweedFS, named alongside MinIO in issue
  #2255, is unimplemented; nothing here has been checked for whether the
  abstraction actually generalizes to a second, differently-shaped object
  storage engine.

## Future enhancements

Roughly in the order they'd need to land to satisfy issue #2255's full
scope:

1. **Bucket and user/IAM management.** Model buckets and MinIO IAM
   users/policies as either `Instance.spec.parameters` (validated against a
   declared `parametersSchema` — the mechanism already exists and needs no
   new CRDs) or dedicated namespaced CRDs reconciled by this same provider
   binary (closer to how `Backup`/`Restore` relate to `Instance` today).
   Either way, this replaces the current root-credential reuse in the
   backup bridge with a dedicated, least-privilege service account per
   `BackupStorage`.
2. **Replication and access-policy configuration** through the same
   `Instance.spec.parameters` mechanism, once (1) establishes the pattern.
3. **Scaling and upgrade support**, exercised the same way the existing
   pg/pxc/psmdb providers are: growing `replicas`/pool count and moving
   between declared `Provider` version bundles without data loss, with
   before/after verification against a real cluster.
4. **A second provider (SeaweedFS)** — the strongest test of whether this
   package's shape (a thin `ProviderInterface` implementation plus a
   hand-declared CRD subset) actually generalizes, versus being
   accidentally MinIO-specific.
5. **A real backup-integration test**: an actual OpenEverest-managed
   database (once one exists as a `provider-runtime`-based `Instance`, not
   just the legacy `dbProvider`-based pg/pxc/psmdb) backing up through
   OpenEverest's own backup path into a `BackupStorage` this provider
   registered, not just a raw S3 client standing in for one.
6. **Watch-based status propagation** on the owned `Tenant`
   (`controller.WatchProvider`, already part of the SDK) instead of relying
   on the reconciler's periodic resync.
7. **UI/CLI/API surface**: exposing object-storage `Instance` creation,
   status, and (once (1) lands) bucket/user management through the same
   dashboard, `everestctl`, and REST API surface already used for managed
   databases.
8. **Production-hardening**: replace the pinned MinIO image tag with a
   `Provider`-defined, upgradeable version catalog (the mechanism already
   exists via `ProviderSpec.ComponentTypes`/`Versions`, just needs a
   maintained set of entries), and revisit whether root-credential reuse in
   (pre-1) code paths is acceptable even as an interim state.

## Running it locally

Prerequisites: a Kubernetes cluster (a local `kind` cluster works — MinIO
Operator install docs assume nothing OpenEverest-specific), `kubectl`, and
`helm`.

```sh
# Install the MinIO Operator (see https://min.io/docs/minio/kubernetes/upstream/operations/installation.html)
helm repo add minio-operator https://operator.min.io
helm install minio-operator minio-operator/operator -n minio-operator --create-namespace

# Install the Provider/Instance/BackupStorage CRDs this provider depends on
kubectl apply -f ../../config/crd/bases/core.openeverest.io_providers.yaml
kubectl apply -f ../../config/crd/bases/core.openeverest.io_instances.yaml
kubectl apply -f ../../config/crd/bases/backup.openeverest.io_backupstorages.yaml

# Register the provider and run its controller
kubectl apply -f manifest/provider.yaml
go run ./cmd

# In another terminal, create an Instance and watch it reconcile
kubectl apply -f manifest/sample-instance.yaml
kubectl get instance,tenant -n default
```
