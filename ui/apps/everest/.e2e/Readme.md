# E2E tests

## How to Run Locally

**in `openeverest` dir**

- `make k3d-cluster-up` - it will create K3D cluster and place kubeconfig into ~/.kube/ dir

- `make deploy-all` - will build everest packages (CLI, server), prepare docker images (server, operator) and deploy all this into your current K8S cluster (current context in kubeconfig -> K3D cluster you created on prev step)

**in `everest/ui/apps/everest/.e2e`**

- `make ci-init` - it will install tests dependencies and install an additional stuff into everest deployment (additional DB namespaces, MinIO, PMM helm chart, ...)

- `make test` - run UI e2e tests
