# Setting up a development environment

This directory holds the configuration files for creating a development
environment for everest.

[tilt.dev](https://docs.tilt.dev/install.html) builds and deploys components to
a local Kubernetes cluster and watches for changes to trigger rebuilds. Logs are
available in the Tilt web UI.

On the `release-2.0` branch, this Tiltfile deploys the OpenEverest **core** only
(API server, controller, core CRDs, and UI). Database providers live in separate
repositories and are installed separately.

## Prerequisites

Install [Go](https://go.dev/dl/) (1.26.3+), [Docker](https://docs.docker.com/engine/install/),
[kubectl](https://kubernetes.io/docs/tasks/tools/), [Helm](https://helm.sh/docs/intro/install/),
[k3d](https://k3d.io), [Tilt](https://docs.tilt.dev/install.html), and
[pnpm](https://pnpm.io/installation) (for the frontend build).

## Quick start

### 1. Get repos and set proper v2 branches

Check out this repository on **`release-2.0`**.

Clone [helm-charts](https://github.com/openeverest/helm-charts) on **`v2`**:

```sh
git clone -b v2 https://github.com/openeverest/helm-charts.git
```

| Repository | Branch |
|------------|--------|
| [openeverest/openeverest](https://github.com/openeverest/openeverest) | `release-2.0` |
| [openeverest/helm-charts](https://github.com/openeverest/helm-charts) | `v2` |

Copy and configure the core dev files:

```sh
cp dev/.env.example dev/.env
cp dev/config.yaml.example dev/config.yaml
```

Set in `dev/.env`:

```sh
EVEREST_CHART_DIR=<path to helm-charts>/charts/everest
```

Set DB namespaces in `dev/config.yaml` as needed.

### 2. Create k3d with `make dev-up`

```sh
make dev-up
```

The Makefile runs `k3d-cluster-up-dev` first, which creates a k3d cluster named
`everest-dev`. Tilt does not create the cluster itself.

### 3. Deploy core with Tilt

The same `make dev-up` command then starts Tilt for the core. The UI/API is at
http://localhost:8080. Keep Tilt running.

### 4. Deploy provider with Tilt

After the core is up (steps 2–3), in a provider repository (see
[Appendix: Provider repositories](#provider-repositories)):

```sh
INSTALL_OPENEVEREST=false tilt up -f dev/Tiltfile --port 10351
```

Use another port (e.g. `10352`, `10353`) for each additional provider.

### 5. Done

```sh
kubectl get providers
```

Refresh the UI. Installed engines should appear and you can create database
instances.

If you only completed steps 2–3, the UI shows **"No providers installed"** until
step 4.

## Tear down

```sh
make dev-down       # Stop Tilt (cluster remains running)
make dev-destroy    # Destroy the k3d cluster (run dev-down first if Tilt is still up)
```

## Troubleshooting

| Symptom | What to do |
|---------|------------|
| **No providers installed** | Complete [Quick start](#quick-start) step 4 |
| **localhost:8080 unreachable** | Keep Tilt running |
| **Provider Helm / CRD errors** | Run `make dev-destroy` and start fresh on `release-2.0` |

---

## Appendix

### Branches and `config.yaml`

On `main`, `dev/config.yaml.example` includes an `operators:` block and the
Tiltfile installs database operators. On `release-2.0`, `config.yaml` defines
namespaces only. Do not copy `config.yaml` from `main`.

### Provider repositories

| Engine | Provider repository |
|--------|----------------------|
| PostgreSQL | [provider-percona-postgresql](https://github.com/openeverest/provider-percona-postgresql) |
| MongoDB | [provider-percona-server-mongodb](https://github.com/openeverest/provider-percona-server-mongodb) |
| MySQL (PXC) | [provider-percona-xtradb-cluster](https://github.com/openeverest/provider-percona-xtradb-cluster) |

Each provider chart installs the provider controller (registers a `Provider` CR)
and the database operator subchart.

Build chart dependencies once per provider before starting Tilt:

```sh
helm repo add percona https://percona.github.io/percona-helm-charts/
helm repo update
helm dependency build charts/<provider-chart-name>
```

For day-to-day provider-only work, use the provider repo's `make dev-up` (see its
`dev/README.md`). To run a provider against a locally built core, follow
[Quick start](#quick-start) step 4 after steps 2–3.

More detail:
[Developing the core and the provider together](https://github.com/openeverest/provider-percona-server-mongodb/blob/main/dev/README.md#developing-the-core-and-the-provider-together).

### CI-style local testing

```sh
make k3d-cluster-up
make deploy-all
```

Creates k3d cluster `everest-server-test` and deploys via NodePort (UI at
http://localhost:8080).

Tear down:

```sh
make undeploy
make k3d-cluster-down
```

### Remote development on GKE

1. Set your default gcloud project:

   ```sh
   export CLOUDSDK_CORE_PROJECT=percona-everest
   ```

2. Create a GKE cluster:

   ```sh
   gcloud container clusters create <NAME> --cluster-version 1.27 --preemptible --machine-type n1-standard-4 --num-nodes=3 --zone=europe-west1-c --labels delete-cluster-after-hours=12 --no-enable-autoupgrade
   ```

3. Create an Artifact Registry repo according to [Google's instructions](https://cloud.google.com/artifact-registry/docs/docker/store-docker-container-images#create)

4. Configure Docker access:

   ```sh
   gcloud auth configure-docker <REGISTRY_REGION>-docker.pkg.dev
   ```

5. In `dev/.env`, set `K8S_CONTEXT` to your GKE context and `DOCKER_REGISTRY_URL`
   to your registry (see `dev/.env.example`).

6. Complete [Quick start](#quick-start) step 1, then run `make dev-up`.

Destroy the cluster when not in use. Clean up the registry periodically; Tilt
pushes a new image on each rebuild.

### Frontend development with Vite

Rebuilding the frontend in Tilt takes ~30s. For UI work, complete
[Quick start](#quick-start) steps 2–3, then:

```sh
cd ui
make dev
```

Dev UI: http://localhost:3000 (proxies API calls to http://localhost:8080).

Set `enableFrontendBuild: false` in `config.yaml` to skip Tilt frontend rebuilds.

### Remote debugging

Set in `dev/.env`:

```sh
export EVEREST_DEBUG=true
```

Connect your IDE to localhost port `40000` (Everest Server). See your IDE docs;
GoLand users can follow
[this guide](https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html#step-2-create-the-go-remote-run-debug-configuration).

### k3d registry port

The default k3d registry uses port `5000`, which may be occupied on some systems
(e.g. macOS Control Center). Update `hostPort` in `k3d_config.dev.yaml`.

### Extended troubleshooting

| Symptom | Likely cause | What to do |
|---------|--------------|------------|
| UI shows **"No providers installed"** | Core running, no `Provider` CRs yet | Install providers (see [Provider repositories](#provider-repositories)). `kubectl get crd providers.core.openeverest.io` should exist; `kubectl get providers` is empty until providers are installed |
| http://localhost:8080 unreachable | Tilt stopped | Keep Tilt running, or `kubectl port-forward -n everest-system svc/everest 8080:8080` |
| Provider Helm fails: **CRDs already exist** | Stale cluster or mixed `main` / `release-2.0` | `make dev-destroy`, use `release-2.0` `config.yaml` (no `operators:`), start fresh |
| Core Tilt fails on startup | Wrong branch for `helm-charts` | Check out **`v2`** on `helm-charts` |
| `operators:` in `config.yaml` | Copied config from `main` | Use [config.yaml.example](config.yaml.example) from `release-2.0` |
| Legacy DatabaseEngine present | v1 resources in cluster | `make dev-destroy`, recreate, install v2 providers |

If you switch between `main` and `release-2.0`, or change provider setup, run
`make dev-destroy` before starting again.
