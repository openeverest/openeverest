# Setting up a development environment

This directory holds the configuration files for creating a development
environment for everest.

[tilt.dev](https://docs.tilt.dev/install.html) builds and deploys all
components to a local kubernetes cluster and also watches for changes in each
of the components' repo in order to trigger a rebuild/redeploy of the affected
components.

Build and runtime logs can be easily accessed using tilt's web UI.

## Prerequisites

1. Install [k3d](https://k3d.io)

2. Install [tilt.dev](https://docs.tilt.dev/install.html)
NOTE: for MacOS tilt needs to have installed and runing `docker-desktop` tool. This is not required and can be skiped since we use `k3d` instead.
 
3. Clone [everest-operator](https://github.com/percona/everest-operator).

4. Clone [helm-charts](https://github.com/openeverest/helm-charts).

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
3. Create Artifacts registry accodring to [instructions](https://cloud.google.com/artifact-registry/docs/docker/store-docker-container-images#create)  
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

Copy file dev/.env.example to dev/.env and set the following environment variables:
```sh
EVEREST_OPERATOR_DIR=<path to github.com/percona/everest-operator repository directory>
EVEREST_CHART_DIR=<path to github.com/openeverest/helm-charts>/charts/everest
```

or set environment variables manually in the terminal:

```sh
export EVEREST_OPERATOR_DIR=<path to github.com/percona/everest-operator repository directory>
export EVEREST_CHART_DIR=<path to github.com/openeverest/helm-charts>/charts/everest
```

2. Set namespaces for the Everest components:

Copy file dev/config.yaml.example to dev/config.yaml and:

- Set the needed DB namespaces that will be created automatically.
- (Mostly for FE devs) If you want to disable the Tilt frontend build, save time and avoid FE rebuilds (and, therefore, BE rebuilds), keeping the dev flow of using Vite, set `enableFrontend: false`

3. (Optional) If you want to test a specific version of a given DB operator you can set the following environment variables in .env file or in the terminal:
```sh
export PXC_OPERATOR_VERSION=1.19.0
export PSMDB_OPERATOR_VERSION=1.22.0
export PG_OPERATOR_VERSION=2.8.2
```

4. (Optional) If you want to debug Everest Server and/or Everest operator remotely, you can set the following environment variables in .env file or in the terminal: 
```sh
export EVEREST_DEBUG=true
export EVEREST_OPERATOR_DEBUG=true
```
In such a case you can setup your IDE to connect to port on your `localhost` and use debugging tools in your IDE.

Debugging port for Everest Server: `40000`.

Debugging port for Everest Operator: `40001`.

Refer to instructions in your IDE on how to setup remote debugging. 

For GoLand, you can refer to [this](https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html#step-2-create-the-go-remote-run-debug-configuration) link.

5. Start Tilt:
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
but then run a local dev instance of the frontend by running `make dev` from
the frontend repo. This dev instance will be available at http://localhost:3000
while still connecting to the everest API server running inside k8s.
