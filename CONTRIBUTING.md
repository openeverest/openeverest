# Contributing to OpenEverest API Server

OpenEverest API Server uses two types of methods:

- "own" methods, such as registering a Kubernetes cluster in Everest and listing the clusters.
-  proxy methods for the Kubernetes API, including all resource-related methods like database-cluster, database-cluster-restore, and database-engine.

The API server basic code is generated using [oapi-codegen](https://github.com/deepmap/oapi-codegen) from the api/openapi/http-api.yaml file.
The proxy methods align with Everest operator methods but don't support all original parameters, because these are not required.
You can find the definition of the custom resources in the [Everest operator repo](https://github.com/openeverest/openeverest-operator/tree/main/config/crd/bases).

### Run everest locally
0. Prerequisites:
    - Golang 1.24.x
    - Make 3.x
    - Docker 20.x
    - Git 2.x
    - k3d 5.x
1. Check out the repo:
`git clone https://github.com/openeverest/openeverest`
2. Navigate to the repo folder:
`cd openeverest`
3. Check out a particular branch if needed:
`git checkout <branch_name>`
4. Run the dev environment:
`make k3d-cluster-up`
5. Build the CLI: `make build-cli-debug`
6. Deploy Everest using the CLI : `make deploy`

### Add a new proxy method
1. Copy the corresponding k8s spec to the [openapi.yml](./api/openapi/http-api.yaml). For information on observing your cluster API, see [Kubernetes: How to View Swagger UI blog post](https://jonnylangefeld.com/blog/kubernetes-how-to-view-swagger-ui), which details the operator-defined methods (if the everest operator is installed).

2. Make necessary spec modifications. When designing new methods:

-  follow the [Restful API guidelines](https://opensource.zalando.com/restful-api-guidelines/). - - use kebab-case instead of everest operator API.
- determine parameters to expose via proxy.
3. If needed, copy the custom resources schema from the [Everest operator config](https://github.com/percona/dbaas-operator/tree/main/config/crd/bases) to the **Components** section of the [openapi.yml](./api/openapi/http-api.yaml) file.

4. Run the following command to generate the code:
```bash
 $ make gen
```
5. Implement the missing `ServerInterface` methods.
6. Run `make format` to format the code and group the imports.
7. Run `make check` to verify that your code works and meets all style requirements.


### Running integration tests

To run integration tests, see [OpenEverest API integration tests](api-tests/README.md).

### Working with local Kubernetes instances like Minikube or Kind

When working with local Kubernetes clusters, Everest API server cannot connect to them because they often use `127.0.0.1` or `localhost` addresses. However, it is possible to connect to the host machine using `host.docker.internal` hostname since Everest API Server runs inside a Docker container.

To do this, add the following host to the `/etc/hosts` file on your local machine:

```
127.0.0.1          host.docker.internal
```

### Troubleshooting

Here are some commands that can help you fix potential issues:
#### Operator installation process
```bash
kubectl -n namespace get sub         # Check that subscription was created for an operator
kubectl -n namespace get ip          # Check that install plan was created and approved for an operator
kubectl -n namespace get csv         # Check that Cluster service version was created and phase is Installed
kubectl -n namespace get deployment  # Check that deployment exist
kubectl -n namespace get po          # Check that pods for an operator is running
kubectl -n namespace logs <podname>  # Check logs for a pod
```
#### Database Cluster troubleshooting

```bash
kubectl -n namespace get db          # Get list of database clusters
kubectl -n namespace get po          # Get pods for a database cluster
kubectl -n namespace describe db     # Describe database cluster. Provides useful information about conditions or messages
kubectl -n namespace describe pxc    # Describe PXC cluster
kubectl -n namespace describe psmdb  # Describe PSMDB cluster
kubectl -n namespace describe pg     # Describe PG cluster
kubectl -n namespace logs <podname>  # Check logs for a pod
```

#### PVC troubleshooting
```bash
kubectl -n namespace get pvc  # PVCs should be Bound
```

#### MySQL database cluster is not up
If a PXC cluster remains in initializing, use these checks and fixes.

* HAProxy resources: ensure at least 600m CPU and 600M memory (prefer 1 CPU and 1G) because low resources can throttle HAProxy or its external health-checks and cause timeouts. This issue is common on Arm systems because the HAProxy image is currently built only for amd64.

  ```bash
  kubectl -n everest get pxc -o jsonpath='{.items[*].spec.haproxy.resources}'
  ```

* Storage size: set at least `2Gi` to provide space for gcache (Galera uses `gcache=1Gi`).

  ```bash
  kubectl -n everest get db -o jsonpath='{.items[*].spec.engine.storage.size}'
  ```

* Storage unit: use `Gi` (binary), not `G` (decimal); k3d uses local-path provisioner and resizing is not supported. Using `G` causes repeated resize to `Gi`, leading to failures.

* Networking: before creating the k3d cluster, load `br_netfilter` and disable AppArmor/SELinux so networking works and HAProxy initialization succeeds. On GitHub Actions with k3d, following adjustments were made.

  ```bash
  sudo modprobe br_netfilter
  sudo systemctl stop apparmor && sudo aa-teardown
  ```
