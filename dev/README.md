# Setting up a development environment

This directory holds the configuration files for creating a development
environment for everest.

[tilt.dev](https://docs.tilt.dev/install.html) builds and deploys all
components to a local kubernetes cluster and also watches for changes in each
of the components' repo in order to trigger a rebuild/redeploy of the affected
components.

Build and runtime logs can be easily accessed using tilt's web UI.

## Prerequisites

1. Install [Go](https://go.dev/dl/) (version 1.26 or later)

2. Install [Docker](https://docs.docker.com/engine/install/)

3. Install [kubectl](https://kubernetes.io/docs/tasks/tools/)

4. Install [Helm](https://helm.sh/docs/intro/install/)

   ```sh
   curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
   ```

5. Install [k3d](https://k3d.io)

   > **NOTE (Linux)**: make sure the `br_netfilter` kernel module is loaded
   > *before* creating the cluster. k3d nodes are containers sharing the host
   > kernel and cannot load it themselves; without it, Service ClusterIPs
   > silently blackhole for pods scheduled on the same node as their backend.
   > See [Troubleshooting](#troubleshooting).
   >
   > ```sh
   > sudo modprobe br_netfilter
   > echo br_netfilter | sudo tee /etc/modules-load.d/k8s.conf  # persist
   > ```

6. Install [tilt.dev](https://docs.tilt.dev/install.html)
NOTE: for MacOS tilt needs to have installed and running `docker-desktop` tool. This is not required and can be skipped since we use `k3d` instead.

7. Install [pnpm](https://pnpm.io/installation) (required for the frontend build)

   ```sh
   npm install -g pnpm
   ```

8. Clone [helm-charts](https://github.com/openeverest/helm-charts).

> **NOTE**: This Tiltfile is focused on OpenEverest **core** development (the
> API server, controller, CRDs and UI). Database providers (PSMDB, PXC, ...) are
> developed in their own repositories, each with its own `dev/Tiltfile`. See
> [Developing providers](#developing-providers) below.

## Set up the environment

### 1. Set up k8s & registry   
#### Option A: Local (Tilt development)
```sh
make dev-up
```
This creates a k3d cluster and starts Tilt. The Everest UI will be available at http://localhost:8080.

> **NOTE**: The default k3d registry uses port `5000`, which may already be occupied on some systems (e.g., macOS Control Center). Update the `hostPort` in `k3d_config.dev.yaml`.

#### Option B: Local (CI-style testing)
```sh
make k3d-cluster-up
make deploy-all
```
This creates a k3d cluster and deploys Everest using the `deploy` target, which exposes the service via NodePort.

#### Option C: Remote (GKE)  
1. Setup your default gcloud project, e.g.  
```sh
export CLOUDSDK_CORE_PROJECT=percona-everest
```  
2. Create GKE cluster  
```sh
gcloud container clusters create <NAME> --cluster-version 1.27 --preemptible --machine-type n1-standard-4  --num-nodes=3 --zone=europe-west1-c --labels delete-cluster-after-hours=12 --no-enable-autoupgrade
```  
3. Create Artifacts registry according to [instructions](https://cloud.google.com/artifact-registry/docs/docker/store-docker-container-images#create)  
4. Configure access  
```sh
gcloud auth configure-docker <REGISTRY_REGION>-docker.pkg.dev
```
5. Uncomment and edit `allow_k8s_contexts` and `default_registry` in the Tiltfile

⚠️ To avoid extra costs do not forget to:
- Destroy external cluster when not used
- Cleanup the registry periodically since tilt pushes a new image each time something is changed in the project. 


### 2. Configure and start Tilt
1. Set environment variables:

Copy file dev/.env.example to dev/.env and set the following environment variable:
```sh
EVEREST_CHART_DIR=<path to github.com/openeverest/helm-charts>/charts/everest
```

or set it manually in the terminal:

```sh
export EVEREST_CHART_DIR=<path to github.com/openeverest/helm-charts>/charts/everest
```

The chart branch must match the OpenEverest line you are developing: `main` for
v2, `v1.x` for v1. If your Tilt environment starts with errors, make sure the
chart checkout is up to date.

2. Set namespaces for the Everest components:

Copy file dev/config.yaml.example to dev/config.yaml and:

- Set the needed DB namespaces that will be created automatically.
- (Mostly for FE devs) If you want to disable the Tilt frontend build, save time and avoid FE rebuilds (and, therefore, BE rebuilds), keeping the dev flow of using Vite, set `enableFrontendBuild: false`
- Set `enablePluginHub: false` to skip deploying the OpenEverest Plugin Hub.

3. (Optional) If you want to debug the Everest Server remotely, set the following environment variable in .env file or in the terminal:
```sh
export EVEREST_DEBUG=true
```
In such a case you can setup your IDE to connect to port on your `localhost` and use debugging tools in your IDE.

Debugging port for Everest Server: `40000`.

Refer to instructions in your IDE on how to setup remote debugging. 

For GoLand, you can refer to [this](https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html#step-2-create-the-go-remote-run-debug-configuration) link.

4. Start Tilt:
```sh
make dev-up
```

The everest UI/API will be available at http://localhost:8080.

## Tear down the environment

### For Tilt development:
```sh
make dev-down       # Stop Tilt (cluster remains running)
make dev-destroy    # Stop Tilt and destroy the cluster
```

### For CI-style testing:
```sh
make undeploy       # Undeploy Everest
make k3d-cluster-down  # Destroy the cluster
```

## Notes for frontend development

Rebuilding the frontend takes ~30s which makes this strategy not very efficient
for frontend development. Therefore, we recommend frontend developers to run
tilt as described in [Set up the environment](#set-up-the-environment) section
(with `enableFrontendBuild: false` in `dev/config.yaml`) but then run a local dev
instance of the frontend by running `make -C ui dev` from the repository root.
This dev instance will be available at http://localhost:3000 while still
connecting to the everest API server running inside k8s.

## Developing providers

This Tiltfile builds and deploys the OpenEverest core only. Each provider
repository ships its own `dev/Tiltfile` that installs a released OpenEverest
core and then builds and deploys just that provider. For day-to-day provider
development, use the provider repo's `make dev-up` (see its `dev/README.md`).

### Testing a provider against a locally built core

When you need a provider to run against the core you are building from source,
run two Tilt instances against the same cluster:

1. Start this core dev environment as usual (`make dev-up`). It manages
   `everest-system` and the core CRDs.
2. In the provider repo, start its Tilt instance on a different port with the
   core installation disabled:

   ```sh
   INSTALL_OPENEVEREST=false tilt up -f dev/Tiltfile --port 10351
   ```

The two instances manage disjoint Kubernetes objects (core owns
`everest-system` + the core CRDs; the provider owns its own namespace + the
database operator), so they run side by side without conflicting.

## Troubleshooting

### Some pods can't reach Service ClusterIPs (DNS times out)

An instance hangs in `Provisioning` with replicas restarting on liveness probe
timeouts, and which pods are affected changes on every `make dev-up`. This
means `br_netfilter` is not loaded on the host (Linux only), so kube-proxy's
DNAT is bypassed for traffic that stays on one node's bridge — pods sharing a
node with CoreDNS lose DNS entirely. Confirm with:

```sh
docker logs k3d-everest-dev-server-0 2>&1 | grep br_netfilter
```

Then load the module and recreate the cluster:

```sh
sudo modprobe br_netfilter
echo br_netfilter | sudo tee /etc/modules-load.d/k8s.conf  # persist
make dev-destroy && make dev-up
```

