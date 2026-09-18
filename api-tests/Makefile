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
	# Tests only use the page fixture on chromium; skip firefox/webkit/ffmpeg.
	npx playwright install chromium-headless-shell

.PHONY: gen-types
gen-types: ## Generate TypeScript types from the OpenAPI spec (api/openapi/http-api.yaml).
	pnpm install
	pnpm generate-types

CI_INIT_DEPS := deploy-monitoring init create-test-users
CI_INIT_DEPS += deploy-minio create-test-manifests wait-monitoring
.PHONY: ci-init
ci-init: $(CI_INIT_DEPS) 	## Prepare environment for running tests.

PMM_HELM_CHART_VER=1.4.13
HELM=go tool -modfile=$(TEST_ROOT)/../go.mod helm
.PHONY: deploy-monitoring
deploy-monitoring:   ## Start deploying PMM to the test cluster; wait-monitoring blocks on readiness.
	$(info Deploying PMM instances to test cluster)
	$(HELM) repo add percona https://percona.github.io/percona-helm-charts/
	$(HELM) install pmm --set secret.pmm_password='admin',service.type=ClusterIP percona/pmm --version $(PMM_HELM_CHART_VER) -n everest-monitoring

.PHONY: wait-monitoring
wait-monitoring:   ## Wait for PMM readiness and port-forward it to localhost.
	# PMM 3 starts slowly; deploy-monitoring runs first so the boot overlaps the rest of ci-init.
	kubectl rollout status statefulset/pmm -n everest-monitoring --timeout=15m
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

TEST_PROVIDER_RESOURCE=$(TEST_ROOT)/manifests/test-provider.yaml
TEST_PRESET_RESOURCE=$(TEST_ROOT)/manifests/test-preset.yaml
TEST_SECRET_RESOURCE=$(TEST_ROOT)/manifests/test-secret.yaml
.PHONY: create-test-manifests
create-test-manifests:
	$(info Deploying test provider and preset)
	kubectl apply -f $(TEST_PROVIDER_RESOURCE)
	kubectl apply -f $(TEST_PRESET_RESOURCE)
	kubectl apply -f $(TEST_SECRET_RESOURCE)

.PHONY: cleanup-test-manifests
cleanup-test-manifests:
	$(info Cleaning up test provider and preset)
	kubectl delete -f $(TEST_PRESET_RESOURCE) --ignore-not-found=true
	kubectl delete -f $(TEST_PROVIDER_RESOURCE) --ignore-not-found=true
	kubectl delete -f $(TEST_SECRET_RESOURCE) --ignore-not-found=true

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
