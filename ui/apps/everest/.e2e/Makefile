REPO_ROOT=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/../../../../
REPO_ROOT_MAKEFILE=$(REPO_ROOT)/Makefile

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
	cd $(REPO_ROOT)/ui && pnpm install
	$(info Install Chromium for Playwright)
	npx playwright install --with-deps chromium

CI_INIT_DEPS := init deploy-minio create-db-namespaces deploy-monitoring
CI_INIT_DEPS += k3d-upload-images
.PHONY: ci-init
ci-init: $(CI_INIT_DEPS) 	## Prepare environment for running tests.

MINIO_RESOURCE=$(REPO_ROOT)/.github/minio.conf.yaml
.PHONY: install-minio
deploy-minio:
	$(info Installing MinIO instance to K3D test cluster)
	kubectl apply -f $(MINIO_RESOURCE)

PMM_HELM_CHART_VER=1.3.21
HELM=go tool -modfile=$(REPO_ROOT)/go.mod helm
.PHONY: deploy-monitoring
deploy-monitoring:     ## Deploy monitoring instances to K3D test cluster.
	$(info Deploying PMM instances to K3D test cluster)
	$(HELM) repo add percona https://percona.github.io/percona-helm-charts/
	$(HELM) install pmm --set secret.pmm_password='admin',service.type=ClusterIP percona/pmm --version $(PMM_HELM_CHART_VER) --timeout 10m --wait

#.PHONY: port-forward-monitoring
#port-forward-monitoring:
#	$(info Port-forwarding PMM instances to localhost)
#	kubectl port-forward --namespace default svc/monitoring-service 8070:443 &

.PHONY: create-db-namespaces
create-db-namespaces:
	$(info Creating DB namespaces in Everest)
	$(MAKE) -f $(REPO_ROOT_MAKEFILE) add-psmdb-namespaces DB_NAMESPACES=psmdb-only
	$(MAKE) -f $(REPO_ROOT_MAKEFILE) add-pxc-namespaces DB_NAMESPACES=pxc-only
	$(MAKE) -f $(REPO_ROOT_MAKEFILE) add-pg-namespaces DB_NAMESPACES=pg-only

#EVERESTCTL=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))/../bin/everestctl
#.PHONY: create-test-users
#create-test-users: ## Create test users and add everest_ci user to admin role.
#	$(info Creating test users in Everest)
#	$(EVERESTCTL) accounts create -u test -p password
#	$(EVERESTCTL) accounts create -u everest_ci -p password
#	kubectl patch configmap everest-rbac -n everest-system --patch "$$(kubectl get configmap everest-rbac -n everest-system -o json | jq '.data["policy.csv"] += "\ng, everest_ci, role:admin"' | jq '{data: { "policy.csv": .data["policy.csv"] } }')"
#	kubectl get configmap everest-rbac -n everest-system -ojsonpath='{.data.policy\.csv}'

.PHONY: k3d-upload-pg-images
k3d-upload-pg-images: ## Pre-upload Postgresql images to K3D test  cluster to speedup tests execution.
	$(info Uploading PG images to K3D test cluster)
	@{ \
	pg_engine_image=$$(kubectl -n pg-only get dbengine/percona-postgresql-operator -ojson | jq -r '[.status.availableVersions.engine[] | select(.status == "recommended").imagePath][-1] ') ;\
	pg_proxy_image=$$(kubectl -n pg-only get dbengine/percona-postgresql-operator -ojson | jq -r '[.status.availableVersions.proxy.pgbouncer[] | select(.status == "recommended").imagePath][-1] ') ;\
	pg_backup_image=$$(kubectl -n pg-only get dbengine/percona-postgresql-operator -ojson | jq -r '[.status.availableVersions.backup[] | select(.status == "recommended").imagePath][-1] ') ;\
	pg_operator_image=$$(kubectl -n pg-only get deploy/percona-postgresql-operator -o json | jq -r '.spec.template.spec.containers[] | select (.name == "operator").image') ;\
	docker pull -q $${pg_engine_image} ;\
	docker pull -q $${pg_proxy_image} ;\
	docker pull -q $${pg_backup_image} ;\
	docker pull -q $${pg_operator_image} ;\
	k3d image import -c everest-server-test -m direct $${pg_engine_image} $${pg_proxy_image} $${pg_backup_image} $${pg_operator_image} ;\
	}

.PHONY: k3d-upload-psmdb-images
k3d-upload-psmdb-images: ## Pre-upload PSMDB images to K3D test  cluster to speedup tests execution.
	$(info Uploading PSMDB images to K3D test cluster)
	@{ \
	psmdb_engine_image=$$(kubectl -n psmdb-only get dbengine/percona-server-mongodb-operator -ojson | jq -r '[.status.availableVersions.engine[] | select(.status == "recommended").imagePath][-1] ') ;\
	psmdb_backup_image=$$(kubectl -n psmdb-only get dbengine/percona-server-mongodb-operator -ojson | jq -r '[.status.availableVersions.backup[] | select(.status == "recommended").imagePath][-1] ') ;\
	psmdb_operator_image=$$(kubectl -n psmdb-only get deploy/percona-server-mongodb-operator -o json | jq -r '.spec.template.spec.containers[] | select (.name == "percona-server-mongodb-operator").image') ;\
	docker pull -q $${psmdb_engine_image} ;\
	docker pull -q $${psmdb_backup_image}  ;\
	docker pull -q $${psmdb_operator_image}  ;\
	k3d image import -c everest-server-test -m direct $${psmdb_engine_image} $${psmdb_backup_image} $${psmdb_operator_image}  ;\
	}

.PHONY: k3d-upload-pxc-images
k3d-upload-pxc-images: ## Pre-upload PXC images to K3D test cluster to speedup tests execution.
	$(info Uploading PXC images to K3D test cluster)
	@{ \
	pxc_engine_image=$$(kubectl -n pxc-only get dbengine/percona-xtradb-cluster-operator -ojson | jq -r '[.status.availableVersions.engine[] | select(.status == "recommended").imagePath][-1] ') ;\
	pxc_proxy_image=$$(kubectl -n pxc-only get dbengine/percona-xtradb-cluster-operator -ojson | jq -r '[.status.availableVersions.proxy.haproxy[] | select(.status == "recommended").imagePath][-1] ') ;\
	pxc_backup_image=$$(kubectl -n pxc-only get dbengine/percona-xtradb-cluster-operator -ojson | jq -r '[.status.availableVersions.backup[] | select(.status == "recommended").imagePath][-1] ') ;\
	pxc_operator_image=$$(kubectl -n pxc-only get deploy/percona-xtradb-cluster-operator -o json | jq -r '.spec.template.spec.containers[] | select (.name == "percona-xtradb-cluster-operator").image') ;\
	pmm_client_image=percona/pmm-client:2 ;\
	docker pull -q $${pxc_engine_image} ;\
	docker pull -q $${pxc_proxy_image} ;\
	docker pull -q $${pxc_backup_image} ;\
	docker pull -q $${pxc_operator_image} ;\
	docker pull -q  $${pmm_client_image};\
	k3d image import -c everest-server-test -m direct $${pxc_engine_image} $${pxc_proxy_image} $${pxc_backup_image} $${pxc_operator_image} $${pmm_client_image};\
	}

.PHONY: k3d-upload-images
k3d-upload-images: k3d-upload-pg-images k3d-upload-psmdb-images k3d-upload-pxc-images 	## Pre-upload PG,PSMDB,PXC images to K3D test cluster to speedup tests execution.

##@ Test

# Set a particular test, e.g. make test PROJECT=pr,release:session
PROJECT = pr
# FLAGS =
.PHONY: test
## Run all tests.
test:              
	@{ \
  	set -xe ;\
	RAND=$$(echo $${RANDOM} | md5sum | head -c 5 ) ;\
	PMM_IP=$$(kubectl get svc/monitoring-service -o json --request-timeout=2m | jq -r '.spec.clusterIP')  ;\
	if [ -z "$${PMM_IP}" ]; then echo "Cannot get PMM_IP" ; exit 1 ; fi ;\
	export MONITORING_URL="http://$${PMM_IP}" ;\
	export MONITORING_USER="admin" ;\
	export MONITORING_PASSWORD="admin" ;\
	export CI_USER="admin" ;\
	export CI_PASSWORD="admin" ;\
	export RBAC_USER="rbac_user" ;\
  	export RBAC_PASSWORD="rbac-e2e-test" ;\
    export SESSION_USER="session" ;\
    export SESSION_PASS="session1234" ;\
	export EVEREST_BUCKETS_NAMESPACES_MAP='[["bucket-1","everest"],["bucket-2","psmdb-only"],["bucket-3","pxc-only"],["bucket-4","pg-only"],["bucket-5","everest"]]' ;\
  	export EVEREST_LOCATION_ACCESS_KEY="minioadmin" ;\
	export EVEREST_LOCATION_SECRET_KEY="minioadmin" ;\
	export EVEREST_LOCATION_REGION="us-east-1" ;\
	export EVEREST_LOCATION_URL="https://minio.minio.svc.cluster.local" ;\
	projects=$$(echo $(PROJECT) | tr ',' ' ') ;\
	cmd_proj="" ;\
	for project in $${projects} ; do \
	  cmd_proj="$${cmd_proj} --project=$${project}" ;\
	done ;\
	npx playwright test --config=./playwright.config.ts $${cmd_proj} $${FLAGS};\
	}
