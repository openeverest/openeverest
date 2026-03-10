TEST_ROOT=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
##@ General
.PHONY: default
default: help

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Environment preparation

.PHONY: init
init:     ## Install tests dependencies.
	$(info Installing dependencies)
	pnpm install
	#npm ci
	PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 npx playwright install

# NOTE: API tests do not wait for DB clusters to be ready.
# The only exception is pg-data-importer test, so that only PG images are pre-uploaded to K3D cluster.
CI_INIT_DEPS := init deploy-monitoring create-test-users k3d-upload-images
CI_INIT_DEPS += deploy-data-importer deploy-minio
.PHONY: ci-init
ci-init: $(CI_INIT_DEPS) 	## Prepare environment for running tests.

PMM_HELM_CHART_VER=1.4.13
HELM=go tool -modfile=$(TEST_ROOT)/../go.mod helm
.PHONY: deploy-monitoring
deploy-monitoring:   ## Deploy monitoring instances to K3D test cluster.
	$(info Deploying PMM instances to K3D test cluster)
	$(HELM) repo add percona https://percona.github.io/percona-helm-charts/
	# PMM 3 starts really slowly, so we increase timeout to 15 minutes
	$(HELM) install pmm --set secret.pmm_password='admin',service.type=ClusterIP percona/pmm --version $(PMM_HELM_CHART_VER) --timeout 15m -n everest-monitoring --wait
	$(info Port-forwarding PMM instances to localhost)
	kubectl port-forward -n everest-monitoring svc/monitoring-service 8090:443 &
	kubectl port-forward -n everest-monitoring svc/monitoring-service 8091:443 &

EVERESTCTL=$(TEST_ROOT)/../bin/everestctl
.PHONY: create-test-users
create-test-users: ## Create test users and add everest_ci user to admin role.
	$(info Creating test users in Everest)
	$(EVERESTCTL) accounts create -u test -p password
	$(EVERESTCTL) accounts create -u everest_ci -p password
	kubectl patch configmap everest-rbac -n everest-system --patch "$$(kubectl get configmap everest-rbac -n everest-system -o json | jq '.data["policy.csv"] += "\ng, everest_ci, role:admin"' | jq '{data: { "policy.csv": .data["policy.csv"] } }')"
	kubectl get configmap everest-rbac -n everest-system -ojsonpath='{.data.policy\.csv}'

DATA_IMPORTER_RESOURCE=$(TEST_ROOT)/manifests/dataimporter.yaml
.PHONY: deploy-data-importer
deploy-data-importer:
	$(info Deploying Data Importer to K3D cluster)
	kubectl apply -f $(DATA_IMPORTER_RESOURCE)

MINIO_RESOURCE=$(TEST_ROOT)/manifests/minio.yaml
.PHONY: deploy-minio
deploy-minio:
	$(info Deploying MinIO to K3D cluster)
	kubectl apply -f $(MINIO_RESOURCE)
	if ! kubectl wait deploy/minio --for=condition=Available -n minio --timeout=600s; then \
		echo "❌ MinIO deployment did not become ready in time"; \
		echo "🔎 Deployment YAML:"; \
		kubectl get deploy minio -n minio -o yaml || true; \
		echo "🔎 Pods:"; \
		kubectl get pods -n minio -o wide || true; \
		echo "🔎 Events:"; \
		kubectl get events -n minio --sort-by=.lastTimestamp || true; \
		exit 1; \
    fi

.PHONY: k3d-upload-pg-images
k3d-upload-pg-images: ## Pre-upload Postgresql images to K3D test  cluster to speedup tests execution.
	$(info Uploading PG images to K3D test cluster)
	@{ \
	pg_engine_image=$$(kubectl -n everest get dbengine/percona-postgresql-operator -ojson | jq -r '[.status.availableVersions.engine[] | select(.status == "recommended").imagePath][-1] ') ;\
	pg_proxy_image=$$(kubectl -n everest get dbengine/percona-postgresql-operator -ojson | jq -r '[.status.availableVersions.proxy.pgbouncer[] | select(.status == "recommended").imagePath][-1] ') ;\
	pg_backup_image=$$(kubectl -n everest get dbengine/percona-postgresql-operator -ojson | jq -r '[.status.availableVersions.backup[] | select(.status == "recommended").imagePath][-1] ') ;\
	pg_operator_image=$$(kubectl -n everest get deploy/percona-postgresql-operator -o json | jq -r '.spec.template.spec.containers[] | select (.name == "operator").image') ;\
	for node in $(k3d node list | grep everest-server-test | grep -v registry | awk '{print $1}'); do \
		docker exec $node ctr image pull docker.io/$${pg_engine_image} ;\
		docker exec $node ctr image pull docker.io/$${pg_proxy_image} ;\
		docker exec $node ctr image pull docker.io/$${pg_backup_image} ;\
		docker exec $node ctr image pull docker.io/$${pg_operator_image} ;\
	done ;\
	}

.PHONY: k3d-upload-psmdb-images
k3d-upload-psmdb-images: ## Pre-upload PSMDB images to K3D test  cluster to speedup tests execution.
	$(info Uploading PSMDB images to K3D test cluster)
	@{ \
	psmdb_engine_image=$$(kubectl -n everest get dbengine/percona-server-mongodb-operator -ojson | jq -r '[.status.availableVersions.engine[] | select(.status == "recommended").imagePath][-1] ') ;\
	psmdb_backup_image=$$(kubectl -n everest get dbengine/percona-server-mongodb-operator -ojson | jq -r '[.status.availableVersions.backup[] | select(.status == "recommended").imagePath][-1] ') ;\
	psmdb_operator_image=$$(kubectl -n everest get deploy/percona-server-mongodb-operator -o json | jq -r '.spec.template.spec.containers[] | select (.name == "percona-server-mongodb-operator").image') ;\
	for node in $(k3d node list | grep everest-server-test | grep -v registry | awk '{print $1}'); do \
		docker exec $node ctr image pull docker.io/$${psmdb_engine_image} ;\
		docker exec $node ctr image pull docker.io/$${psmdb_backup_image} ;\
		docker exec $node ctr image pull docker.io/$${psmdb_operator_image} ;\
	done ;\
	}

.PHONY: k3d-upload-pxc-images
k3d-upload-pxc-images: ## Pre-upload PXC images to K3D test cluster to speedup tests execution.
	$(info Uploading PXC images to K3D test cluster)
	@{ \
	pxc_engine_image=$$(kubectl -n everest get dbengine/percona-xtradb-cluster-operator -ojson | jq -r '[.status.availableVersions.engine[] | select(.status == "recommended").imagePath][-1] ') ;\
	pxc_proxy_image=$$(kubectl -n everest get dbengine/percona-xtradb-cluster-operator -ojson | jq -r '[.status.availableVersions.proxy.haproxy[] | select(.status == "recommended").imagePath][-1] ') ;\
	pxc_backup_image=$$(kubectl -n everest get dbengine/percona-xtradb-cluster-operator -ojson | jq -r '[.status.availableVersions.backup[] | select(.status == "recommended").imagePath][-1] ') ;\
	pxc_operator_image=$$(kubectl -n everest get deploy/percona-xtradb-cluster-operator -o json | jq -r '.spec.template.spec.containers[] | select (.name == "percona-xtradb-cluster-operator").image') ;\
	pmm_client_image=percona/pmm-client:2 ;\
	for node in $(k3d node list | grep everest-server-test | grep -v registry | awk '{print $1}'); do \
		docker exec $node ctr image pull docker.io/$${pxc_engine_image} ;\
		docker exec $node ctr image pull docker.io/$${pxc_proxy_image} ;\
		docker exec $node ctr image pull docker.io/$${pxc_backup_image} ;\
		docker exec $node ctr image pull docker.io/$${pxc_operator_image} ;\
		docker exec $node ctr image pull docker.io/$${pmm_client_image} ;\
	done ;\
	}

.PHONY: k3d-upload-images
k3d-upload-images: k3d-upload-pg-images k3d-upload-psmdb-images k3d-upload-pxc-images 	## Pre-upload PG,PSMDB,PXC images to K3D test cluster to speedup tests execution.

##@ Development

.PHONY: lint
lint:               ## Run linter typescript checks.
	npm run lint:ts && npm run typecheck && echo 'Lint OK'

.PHONY: lint-fix
lint-fix:           ## Fix linter errors where possible.
	npx eslint --fix .

##@ Test

# Set a particular test, e.g. make test TEST=database-cluster.spec.ts
PROJECT = api-tests
FLAGS =
.PHONY: test
test:               ## Run all tests.
	@{ \
  	set -xe ;\
	export PMM1_IP=$$(kubectl get -n everest-monitoring svc/monitoring-service -o json --request-timeout=2m | jq -r '.spec.clusterIP')  ;\
	if [ -z "$${PMM1_IP}" ]; then echo "Cannot get PMM1_IP" ; exit 1 ; fi ;\
	export PMM2_IP=$${PMM1_IP} ;\
	export PMM_1_LOCAL_URL="https://admin:admin@localhost:8090" ;\
	export PMM_2_LOCAL_URL="https://admin:admin@localhost:8091" ;\
	export CI_USER="everest_ci" ;\
	export CI_PASSWORD="password" ;\
	export TEST_USER="test" ;\
	export TEST_PASSWORD="password" ;\
	projects=$$(echo $(PROJECT) | tr ',' ' ') ;\
	cmd_proj="" ;\
	for project in $${projects} ; do \
	  cmd_proj="$${cmd_proj} --project=$${project}" ;\
	done ;\
	npx playwright test --config=./playwright.config.ts $${cmd_proj} $${FLAGS};\
	}
