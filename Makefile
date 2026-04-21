REPO_ROOT=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
RELEASE_VERSION ?= v0.0.0-$(shell git rev-parse --short HEAD)
RELEASE_FULLCOMMIT ?= $(shell git rev-parse HEAD)
IMAGE_PREFIX ?= ghcr.io/openeverest
EVEREST_SERVER_DEV_IMAGE_NAME ?= openeverest-dev
EVEREST_OPERATOR_DEV_IMAGE_NAME ?= openeverest-operator-dev
EVEREST_CONTROLLER_DEV_IMAGE_NAME ?= openeverest-controller-dev
EVEREST_CATALOG_DEV_IMAGE_NAME ?= openeverest-catalog-dev
IMAGE_TAG ?= 0.0.0
IMG = $(IMAGE_PREFIX)/$(EVEREST_SERVER_DEV_IMAGE_NAME):$(IMAGE_TAG)
EVEREST_CONTROLLER_IMG = $(IMAGE_PREFIX)/$(EVEREST_CONTROLLER_DEV_IMAGE_NAME):$(IMAGE_TAG)
EVEREST_OPERATOR_IMG = $(IMAGE_PREFIX)/$(EVEREST_OPERATOR_DEV_IMAGE_NAME):$(IMAGE_TAG)
CONTROLLER_TOOLS_VERSION ?= v0.18.0


.PHONY: default
default: help

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## Location to install binaries to
CWD = $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
LOCALBIN := $(CWD)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
KUSTOMIZE ?= $(LOCALBIN)/kustomize
KUBECTL ?= kubectl

## Tool Versions
KUSTOMIZE_VERSION ?= v5.7.0

# ENVTEST_VERSION is the version of controller-runtime release branch to fetch the envtest setup script.
ENVTEST_VERSION ?= $(shell v='$(call gomodver,sigs.k8s.io/controller-runtime)'; \
  [ -n "$$v" ] || { echo "Set ENVTEST_VERSION manually" >&2; exit 1; }; \
  printf '%s\n' "$$v" | sed -E 's/^v?([0-9]+)\.([0-9]+).*/release-\1.\2/')

# ENVTEST_K8S_VERSION is the version of Kubernetes to use for setting up ENVTEST binaries.
ENVTEST_K8S_VERSION ?= $(shell v='$(call gomodver,k8s.io/api)'; \
  [ -n "$$v" ] || { echo "Set ENVTEST_K8S_VERSION manually" >&2; exit 1; }; \
  printf '%s\n' "$$v" | sed -E 's/^v?[0-9]+\.([0-9]+).*/1.\1/')

##@ Development

.PHONY: gen-crd-openapi
gen-crds-openapi: ## Extract OpenAPI schemas from CRD manifests.
	go run hack/gen-crds-openapi/main.go

.PHONY: gen-openapi-ts-types
gen-openapi-ts-types: ## Generate TypeScript types from all OpenAPI YAML files in api/openapi/.
	$(MAKE) -C ui generate-openapi-types

# `make generate` is used by kubebuilder to create new API.
# The presence of generate target is purely for kubebuilder succeed without an error.
.PHONY: generate
generate: gen

.PHONY: gen
gen: gen-crds-deepcopy gen-crds-manifests gen-crds-openapi gen-openapi-ts-types ## Generate code.
	go generate ./...
	python3 hack/add_copyright.py \
		internal/server/api/everest-server.gen.go \
		internal/server/api/crds.gen.go \
		client/everest-client.gen.go \
		client/crds.gen.go \
		internal/server/handlers/mock_handler.go \
		pkg/rbac/mocks/i_enforcer.go \
		ui/api/crds.gen.types.ts \
		ui/api/http-api.types.ts \
		ui/api/index.ts || true
	$(MAKE) format

.PHONY: format
format:                 ## Format source code.
	go tool gofumpt -l -w .
	go tool goimports -local github.com/openeverest/openeverest/v2 -l -w .
	go tool gci write --skip-generated -s standard -s default -s "prefix(github.com/openeverest/openeverest/v2)" .

.PHONY: check
check:                  ## Run checks/linters for the whole project.
	go tool go-consistent -pedantic ./...
	LOG_LEVEL=error go tool golangci-lint run

.PHONY: copyright-check
copyright-check: COPYRIGHT_FLAGS=--check
copyright-check: ## Check changed .go/.ts/.tsx files for missing copyright headers.

.PHONY: copyright-headers
copyright-headers: COPYRIGHT_FLAGS=

.PHONY: copyright-headers copyright-check
copyright-headers: copyright-run ## Add missing copyright headers to changed .go/.ts/.tsx files.
copyright-check: copyright-run

.PHONY: copyright-run
copyright-run:
	@TMP_FILES_LIST=$$(mktemp "$${TMPDIR:-/tmp}/everest_copyright.XXXXXX" 2>/dev/null || mktemp -t everest_copyright.XXXXXX); \
	cleanup() { rm -f "$$TMP_FILES_LIST"; }; \
	trap cleanup EXIT; \
	if [ -n "$(FILES_FILE)" ]; then \
		while IFS= read -r file; do \
			[ -n "$$file" ] && printf '%s\0' "$$file"; \
		done < "$(FILES_FILE)" > "$$TMP_FILES_LIST"; \
	elif [ -n "$(FILES)" ]; then \
		for file in $(FILES); do \
			printf '%s\0' "$$file"; \
		done > "$$TMP_FILES_LIST"; \
	else \
		BASE_BRANCH_LOCAL=$${BASE_BRANCH:-main}; \
		if ! BASE=$$(git merge-base HEAD "$$BASE_BRANCH_LOCAL" 2>/dev/null); then \
			echo "Failed to determine merge base with '$$BASE_BRANCH_LOCAL'. Ensure the branch exists and is fetched, or set BASE_BRANCH explicitly."; \
			exit 1; \
		fi; \
		git diff -z --name-only --diff-filter=ACM "$$BASE" -- '*.go' '*.ts' '*.tsx' > "$$TMP_FILES_LIST"; \
		git ls-files -z --others --exclude-standard -- '*.go' '*.ts' '*.tsx' >> "$$TMP_FILES_LIST"; \
	fi; \
	if [ ! -s "$$TMP_FILES_LIST" ]; then \
		echo "No changed .go/.ts/.tsx files to process."; \
		exit 0; \
	fi; \
	echo "Processing copyright headers for changed files..."; \
	python3 hack/add_copyright.py $(COPYRIGHT_FLAGS) --paths-nul-file "$$TMP_FILES_LIST"

.PHONY: charts
HELM=go tool helm
charts:        ## Install Helm dependency charts for Everest CLI.
	$(HELM) repo add prometheus-community https://prometheus-community.github.io/helm-charts
	$(HELM) repo add openeverest https://openeverest.github.io/helm-charts/
	$(HELM) repo add vm https://victoriametrics.github.io/helm-charts
	$(HELM) repo update

##@ Build
export GOPRIVATE = github.com/percona,github.com/percona-platform,github.com/Percona-Lab
export GOOS = $(shell go env GOHOSTOS)
export CGO_ENABLED = 0
export GOARCH = $(shell go env GOHOSTARCH)

# Everest API server
SERVER_LD_FLAGS = -X 'github.com/openeverest/openeverest/v2/pkg/version.Version=$(RELEASE_VERSION)' \
	-X 'github.com/openeverest/openeverest/v2/pkg/version.FullCommit=$(RELEASE_FULLCOMMIT)' \
	-X 'github.com/openeverest/openeverest/v2/pkg/version.ProjectName=Everest API Server' \
	-X 'github.com/openeverest/openeverest/v2/cmd/config.TelemetryInterval=24h'
SERVER_BUILD_TAGS =
SERVER_GC_FLAGS =

# Helper target to build Everest API server binary.
# CGO_ENABLED, GOOS and GOARCH are set explicitly because Everest API server is running inside a container only.
.PHONY: build-server
build-server-helper: GOOS = linux
build-server-helper: GOARCH = amd64
build-server-helper: $(LOCALBIN)
# We need to ensure that /public/dist/index.html exists before building Everest
# API server because it's embedded into the binary and missing file will cause
# build failure. We avoid touching the file if it already exists to prevent
# unnecessary rebuilds when only the timestamp of the file changes.
	mkdir -p ./public/dist && [ -f ./public/dist/index.html ] || touch ./public/dist/index.html
	$(info Building Everest API server for $(GOOS)/$(GOARCH) with CGO_ENABLED=$(CGO_ENABLED))
	go build -v $(SERVER_BUILD_TAGS) $(SERVER_GC_FLAGS) -ldflags "$(SERVER_LD_FLAGS)" -o $(LOCALBIN)/everest ./cmd

.PHONY: build
build: SERVER_LD_FLAGS += -s -w
build: build-server-helper 	## Build Everest API server binary.

.PHONY: build-debug
build-debug: SERVER_BUILD_TAGS = -tags debug
build-debug: SERVER_GC_FLAGS = -gcflags=all="-N -l"
build-debug: build-server-helper	## Build Everest API server binary with debug symbols.

.PHONY: rc
rc: SERVER_LD_FLAGS += -X 'github.com/openeverest/openeverest/v2/cmd/config.TelemetryURL=https://check-dev.percona.com'
rc: build-server-helper	## Build Everest API server RC version.

.PHONY: release
release: SERVER_LD_FLAGS += -X 'github.com/openeverest/openeverest/v2/cmd/config.TelemetryURL=https://check.percona.com'
release: build-server-helper	## Build Everest API server release version. (Use for building release only!)

# Everest CLI
CLI_LD_FLAGS = -X 'github.com/openeverest/openeverest/v2/pkg/version.Version=$(RELEASE_VERSION)' \
	-X 'github.com/openeverest/openeverest/v2/pkg/version.FullCommit=$(RELEASE_FULLCOMMIT)' \
	-X 'github.com/openeverest/openeverest/v2/pkg/version.ProjectName=everestctl'
CLI_BUILD_TAGS =
CLI_GC_FLAGS =

# Helper target to build Everest CLI binary.
.PHONY: build-cli-helper
build-cli-helper: $(LOCALBIN) charts
	$(info Building Everest CLI for $(GOOS)/$(GOARCH) with CGO_ENABLED=$(CGO_ENABLED))
	go build -v $(CLI_BUILD_TAGS) $(CLI_GC_FLAGS) -ldflags "$(CLI_LD_FLAGS)" -o $(LOCALBIN)/everestctl ./cmd/cli

.PHONY: build-cli
build-cli: CLI_LD_FLAGS += -s -w
build-cli: build-cli-helper	## Build Everest CLI binary.

.PHONY: build-cli-debug
build-cli-debug: CLI_LD_FLAGS += -X 'github.com/openeverest/openeverest/v2/pkg/version.EverestChannelOverride=fast-v0'
build-cli-debug: CLI_BUILD_TAGS = -tags debug
build-cli-debug: CLI_GC_FLAGS = -gcflags=all="-N -l"
build-cli-debug: build-cli-helper	## Build Everest CLI binary with debug symbols and development OLM channel.

UI_DIR = $(CWD)/ui
.PHONY: build-ui
build-ui: ## Build Everest UI and embed it into the Everest API server binary.
	$(info Building Everest UI)
	cd $(UI_DIR) && $(MAKE) init build EVEREST_OUT_DIR=$(CWD)/public/dist

.PHONY: release-cli
release-cli: CLI_LD_FLAGS += -s -w
release-cli: ## Build Everest CLI release versions for different OS and ARCH. (Use for building release only!).
	GOOS=linux GOARCH=amd64 go build -v -ldflags "$(CLI_LD_FLAGS)" -o ./dist/everestctl-linux-amd64 ./cmd/cli
	GOOS=linux GOARCH=arm64 go build -v -ldflags "$(CLI_LD_FLAGS)" -o ./dist/everestctl-linux-arm64 ./cmd/cli
	GOOS=darwin GOARCH=amd64 go build -v -ldflags "$(CLI_LD_FLAGS)" -o ./dist/everestctl-darwin-amd64 ./cmd/cli
	GOOS=darwin GOARCH=arm64 go build -v -ldflags "$(CLI_LD_FLAGS)" -o ./dist/everestctl-darwin-arm64 ./cmd/cli
	GOOS=windows GOARCH=amd64 go build -v -ldflags "$(CLI_LD_FLAGS)" -o ./dist/everestctl.exe ./cmd/cli

# Everest controller manager
CONTROLLER_LD_FLAGS = -X 'github.com/openeverest/openeverest/v2/pkg/version.Version=$(RELEASE_VERSION)' \
	-X 'github.com/openeverest/openeverest/v2/pkg/version.FullCommit=$(RELEASE_FULLCOMMIT)'
CONTROLLER_BUILD_TAGS =
CONTROLLER_GC_FLAGS =

# Helper target to build the Everest controller manager binary.
.PHONY: build-controller-helper
build-controller-helper: GOOS = linux
build-controller-helper: GOARCH = amd64
build-controller-helper: $(LOCALBIN)
	$(info Building Everest controller manager for $(GOOS)/$(GOARCH) with CGO_ENABLED=$(CGO_ENABLED))
	go build -v $(CONTROLLER_BUILD_TAGS) $(CONTROLLER_GC_FLAGS) -ldflags "$(CONTROLLER_LD_FLAGS)" -o $(LOCALBIN)/manager ./cmd/controller

.PHONY: build-controller
build-controller: CONTROLLER_LD_FLAGS += -s -w
build-controller: build-controller-helper	## Build Everest controller manager binary.

.PHONY: build-controller-debug
build-controller-debug: CONTROLLER_BUILD_TAGS = -tags debug
build-controller-debug: CONTROLLER_GC_FLAGS = -gcflags=all="-N -l"
build-controller-debug: build-controller-helper	## Build Everest controller manager binary with debug symbols.

.PHONY: run-controller
run-controller: ## Run the Everest controller manager from your host.
	go run ./cmd/controller

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: docker-build
docker-build: ## Build docker image with Everest API server and controller.
	docker build -f build/package/server/Dockerfile --target openeverest -t ${IMG} .

.PHONY: docker-build-controller
docker-build-controller: build-controller ## Build docker image with Everest controller.
	docker build -f build/package/server/Dockerfile --target controller -t ${EVEREST_CONTROLLER_IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with Everest API server and controller.
	docker push ${IMG}

.PHONY: clean
clean:
	rm -rf $(LOCALBIN)/*
	rm -rf ./dist/*

##@ Test

.PHONY: test
test: setup-envtest ## Run unit tests.
# We need to ensure that /public/dist/index.html exists before running tests
# because it's embedded into the binary and missing file will cause test
# failure. We avoid touching the file if it already exists to prevent
# unnecessary rebuilds when only the timestamp of the file changes.
	mkdir -p ./public/dist && [ -f ./public/dist/index.html ] || touch ./public/dist/index.html
	KUBEBUILDER_ASSETS="$$("$(ENVTEST)" use $(ENVTEST_K8S_VERSION) --bin-dir "$(LOCALBIN)" -p path)" \
	CGO_ENABLED=1 go test -race -timeout=20m ./...

.PHONY: test-cover
test-cover: setup-envtest ## Run unit tests and collect per-package coverage information.
# We need to ensure that /public/dist/index.html exists before running tests
# because it's embedded into the binary and missing file will cause test
# failure. We avoid touching the file if it already exists to prevent
# unnecessary rebuilds when only the timestamp of the file changes.
	mkdir -p ./public/dist && [ -f ./public/dist/index.html ] || touch ./public/dist/index.html
	KUBEBUILDER_ASSETS="$$("$(ENVTEST)" use $(ENVTEST_K8S_VERSION) --bin-dir "$(LOCALBIN)" -p path)" \
	CGO_ENABLED=1 go test -race -timeout=20m -count=1 -coverprofile=cover.out -covermode=atomic ./...

.PHONY: test-crosscover
test-crosscover: setup-envtest ## Run unit tests and collect cross-package coverage information.
# We need to ensure that /public/dist/index.html exists before running tests
# because it's embedded into the binary and missing file will cause test
# failure. We avoid touching the file if it already exists to prevent
# unnecessary rebuilds when only the timestamp of the file changes.
	mkdir -p ./public/dist && [ -f ./public/dist/index.html ] || touch ./public/dist/index.html
	KUBEBUILDER_ASSETS="$$("$(ENVTEST)" use $(ENVTEST_K8S_VERSION) --bin-dir "$(LOCALBIN)" -p path)" \
	CGO_ENABLED=1 go test -race -timeout=20m -count=1 -coverprofile=crosscover.out -covermode=atomic -p=1 -coverpkg=./... ./...

.PHONY: test-integration-monitoring
test-integration-monitoring: docker-build-controller k3d-upload-controller-image
	. ./test/vars.sh && kubectl kuttl test --config test/integration/kuttl-monitoring.yaml

##@ Deployment management

# This target builds the docker image for Everest operator from the commit referenced in go.mod.
# Docker image will be tagged with the same tag as Everest API server image (IMAGE_TAG).
.PHONY: docker-build-operator
docker-build-operator:
	$(info Building Everest Operator Docker image=$(EVEREST_OPERATOR_IMG))
	@{ \
	set -xe ;\
	operator_commit_id=$(word 3, $(subst -,  ,$(word 2, $(shell go list -m github.com/percona/everest-operator)))) ;\
	cd $(shell mktemp -d) ;\
	git clone -q https://github.com/percona/everest-operator.git ;\
	cd ./everest-operator ;\
	git reset --hard $${operator_commit_id} ;\
	make build docker-build IMG=$(EVEREST_OPERATOR_IMG) ;\
	}

DB_NAMESPACES = everest
.PHONY: deploy
deploy:  ## Deploy Everest to K8S cluster using Everest CLI.
	$(info Deploying Everest ($(IMG)) into K8S cluster using everestctl)
	$(LOCALBIN)/everestctl install -v \
	--disable-telemetry \
	--version=$(IMAGE_TAG) \
	--version-metadata-url=https://check-dev.percona.com \
	--operator.mongodb=true \
	--operator.postgresql=true \
	--operator.mysql=true \
	--skip-wizard \
	--namespaces $(DB_NAMESPACES) \
	--helm.set server.image=$(IMAGE_PREFIX)/$(EVEREST_SERVER_DEV_IMAGE_NAME) \
	--helm.set server.apiRequestsRateLimit=500 \
	--helm.set server.sessionRequestsRateLimit=200 \
	--helm.set versionMetadataURL=https://check-dev.percona.com \
	--helm.set server.initialAdminPassword=admin \
	--helm.set operator.init=false \
	--helm.set operator.image=$(IMAGE_PREFIX)/$(EVEREST_OPERATOR_DEV_IMAGE_NAME) \
	--helm.set olm.catalogSourceImage=$(IMAGE_PREFIX)/$(EVEREST_CATALOG_DEV_IMAGE_NAME)
	$(MAKE) expose

DEPLOY_ALL_DEPS := build-ui build-debug docker-build k3d-upload-server-image
DEPLOY_ALL_DEPS += docker-build-operator k3d-upload-operator-image
DEPLOY_ALL_DEPS += k3d-upload-server-image deploy
.PHONY: deploy-all
deploy-all: $(DEPLOY_ALL_DEPS) ## Helper to build Everest and its dependencies and deploy to K3D test cluster.

.PHONY: undeploy-cli
undeploy: build-cli-debug ## Undeploy Everest from K8S cluster using Everest CLI.
	$(info Uninstalling Everest from K8S cluster using everestctl)
	$(LOCALBIN)/everestctl uninstall -y -f -v

.PHONY: add-pg-namespaces
add-pg-namespaces: ## Add PostgreSQL namespace to Everest using Everest CLI(usage: DB_NAMESPACES=ns-1,ns-2 make add-pg-namespaces).
	$(info Adding PostgreSQL namespaces=${DB_NAMESPACE} to Everest using everestctl)
	$(LOCALBIN)/everestctl namespaces add $(DB_NAMESPACES) -v \
	--operator.mongodb=false \
	--operator.postgresql=true \
	--operator.mysql=false \
	--skip-wizard

.PHONY: add-psmdb-namespaces
add-psmdb-namespaces: ## Add PSMDB namespace to Everest using Everest CLI(usage: DB_NAMESPACES=ns-1,ns-2 make add-psmdb-namespaces).
	$(info Adding PSMDB namespaces=${DB_NAMESPACE} to Everest using everestctl)
	$(LOCALBIN)/everestctl namespaces add $(DB_NAMESPACES) -v \
	--operator.mongodb=true \
	--operator.postgresql=false \
	--operator.mysql=false \
	--skip-wizard

.PHONY: add-pxc-namespaces
add-pxc-namespaces: ## Add PXC namespace to Everest using Everest CLI(usage: DB_NAMESPACES=ns-1,ns-2 make add-pxc-namespaces).
	$(info Adding PXC namespaces=${DB_NAMESPACE} to Everest using everestctl)
	$(LOCALBIN)/everestctl namespaces add $(DB_NAMESPACES) -v \
	--operator.mongodb=false \
	--operator.postgresql=false \
	--operator.mysql=true \
	--skip-wizard

.PHONY: expose
expose:
	kubectl patch svc -n everest-system everest --type=merge \
	-p '{"spec": {"type": "NodePort", "ports": [{"name": "http", "port": 8080, "protocol": "TCP", "targetPort": 8080, "nodePort": 30080}]}}'

.PHONY: k3d-cluster-up
k3d-cluster-up: ## Create a K8S cluster for testing.
	$(info Creating K3D cluster for testing)
	k3d cluster create --config ./dev/k3d_config.yaml

.PHONY: k3d-cluster-up
k3d-cluster-down: ## Create a K8S cluster for testing.
	$(info Destroying K3D test cluster)
	k3d cluster delete --config ./dev/k3d_config.yaml

.PHONY: k3d-cluster-reset
k3d-cluster-reset: k3d-cluster-down k3d-cluster-up ## Reset the K8S cluster for testing.

.PHONY: k3d-upload-server-image
k3d-upload-server-image: ## Upload the Everest API server image to the testing k3d cluster.
	$(info Uploading Everest API server image=$(IMG) to K3D testing cluster)
	k3d image import -c everest-server-test $(IMG)

.PHONY: k3d-upload-controller-image
k3d-upload-controller-image: ## Upload the Everest controller image to the testing k3d cluster.
	$(info Uploading Everest controller image=$(EVEREST_CONTROLLER_IMG) to K3D testing cluster)
	k3d image import -c everest-server-test $(EVEREST_CONTROLLER_IMG)

.PHONY: k3d-upload-operator-image
k3d-upload-operator-image: ## Upload the Everest operator image to the testing k3d cluster.
	$(info Uploading Everest operator image=$(EVEREST_OPERATOR_IMG) to K3D testing cluster)
	k3d image import -c everest-server-test $(EVEREST_OPERATOR_IMG)

.PHONY: cert
cert:                   ## Create dev TLS certificates.
	mkcert -install
	mkcert -cert-file=dev-cert.pem -key-file=dev-key.pem everest everest.localhost 127.0.0.1

##@ Development with Tilt

.PHONY: k3d-cluster-up-dev
k3d-cluster-up-dev:     ## Create a K8S cluster for Tilt development (no port conflicts).
	@if ! k3d cluster list | grep -q "everest-dev"; then \
		echo "Creating K3D cluster for Tilt development"; \
		k3d cluster create --config ./dev/k3d_config.dev.yaml; \
	else \
		echo "K3D cluster everest-dev already exists"; \
	fi

.PHONY: k3d-cluster-down-dev
k3d-cluster-down-dev:   ## Destroy the K8S cluster for Tilt development.
	@if k3d cluster list | grep -q "everest-dev"; then \
		echo "Destroying K3D dev cluster"; \
		k3d cluster delete --config ./dev/k3d_config.dev.yaml; \
	else \
		echo "K3D cluster everest-dev does not exist"; \
	fi

.PHONY: dev-up
dev-up: k3d-cluster-up-dev  ## Create k3d cluster for Tilt development and start Tilt.
	tilt up -f dev/Tiltfile

.PHONY: dev-down
dev-down:               ## Stop Tilt (keeps cluster running for later reuse).
	tilt down -f dev/Tiltfile

.PHONY: dev-destroy
dev-destroy: k3d-cluster-down-dev ## Destroy the k3d cluster.

##@ GitHub PR

CHART_BRANCH ?= v2
.PHONY: update-dev-chart
update-dev-chart: ## Update dependency to Everest Helm chart to the latest version from the specified branch (default v2).
	COMMIT=$$(git ls-remote https://github.com/openeverest/helm-charts refs/heads/$(CHART_BRANCH) | cut -f1) && \
	GOPROXY=direct go get -u -v github.com/openeverest/helm-charts/charts/everest@$$COMMIT
	go mod tidy

EVEREST_OPERATOR_BRANCH ?= main
.PHONY: update-dev-everest-operator
update-dev-everest-operator: ## Update dependency to Everest operator to the latest version from the specified branch (default main).
	GOPROXY=direct go get -u -v github.com/percona/everest-operator@${EVEREST_OPERATOR_BRANCH}
	go mod tidy

.PHONY: prepare-pr
prepare-pr: gen ## Prepare code for pushing to GitHub PR (includes 'update-dev-chart' and 'update-dev-everest-operator' targets).
	CHART_BRANCH=${CHART_BRANCH} $(MAKE) update-dev-chart
	EVEREST_OPERATOR_BRANCH=${EVEREST_OPERATOR_BRANCH} $(MAKE) update-dev-everest-operator

.PHONY: gen-crds-deepcopy
gen-crds-deepcopy: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: gen-crds-manifests
gen-crds-manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd:allowDangerousTypes=true webhook paths="./..." output:crd:artifacts:config=config/crd/bases

##@ Kustomize Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: gen-crds-manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	@out="$$( "$(KUSTOMIZE)" build config/crd 2>/dev/null || true )"; \
	if [ -n "$$out" ]; then echo "$$out" | "$(KUBECTL)" apply -f -; else echo "No CRDs to install; skipping."; fi

.PHONY: uninstall
uninstall: gen-crds-manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	@out="$$( "$(KUSTOMIZE)" build config/crd 2>/dev/null || true )"; \
	if [ -n "$$out" ]; then echo "$$out" | "$(KUBECTL)" delete --ignore-not-found=$(ignore-not-found) -f -; else echo "No CRDs to delete; skipping."; fi

.PHONY: deploy-controller
deploy-controller: gen-crds-manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && "$(KUSTOMIZE)" edit set image controller=${EVEREST_CONTROLLER_IMG}
	"$(KUSTOMIZE)" build config/default | "$(KUBECTL)" apply -f -

.PHONY: undeploy-controller
undeploy-controller: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	"$(KUSTOMIZE)" build config/default | "$(KUBECTL)" delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: build-installer
build-installer: gen-crds-manifests kustomize ## Generate a consolidated YAML with CRDs and deployment.
	mkdir -p dist
	cd config/manager && "$(KUSTOMIZE)" edit set image controller=${IMG}
	"$(KUSTOMIZE)" build config/default > dist/install.yaml

.PHONY: deploy-test-controller
deploy-test-controller: gen-crds-manifests kustomize deploy-cert-manager
	cd config/test && "$(KUSTOMIZE)" edit set image controller=${EVEREST_CONTROLLER_IMG}
	$(KUSTOMIZE) build config/test | kubectl apply -f -

##@ Dependencies

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: setup-envtest
setup-envtest: envtest ## Download the binaries required for ENVTEST.
	@echo "Setting up envtest binaries for Kubernetes version $(ENVTEST_K8S_VERSION)..."
	@"$(ENVTEST)" use $(ENVTEST_K8S_VERSION) --bin-dir "$(LOCALBIN)" -p path || { \
		echo "Error: Failed to set up envtest binaries for version $(ENVTEST_K8S_VERSION)."; \
		exit 1; \
	}

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: deploy-cert-manager
deploy-cert-manager: # Install cert-manager used by controller webhook.
	kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
	kubectl wait --for=condition=available --timeout=120s deployment/cert-manager -n cert-manager
	kubectl wait --for=condition=available --timeout=120s deployment/cert-manager-webhook -n cert-manager
	kubectl wait --for=condition=available --timeout=120s deployment/cert-manager-cainjector -n cert-manager

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist.
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] && [ "$$(readlink -- "$(1)" 2>/dev/null)" = "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f "$(1)" ;\
GOBIN="$(LOCALBIN)" go install $${package} ;\
mv "$(LOCALBIN)/$$(basename "$(1)")" "$(1)-$(3)" ;\
} ;\
ln -sf "$$(realpath "$(1)-$(3)")" "$(1)"
endef

define gomodver
$(shell go list -m -f '{{if .Replace}}{{.Replace.Version}}{{else}}{{.Version}}{{end}}' $(1) 2>/dev/null)
endef
