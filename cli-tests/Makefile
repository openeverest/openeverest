IMAGE_PREFIX ?= ghcr.io/openeverest
EVEREST_SERVER_DEV_IMAGE_NAME ?= openeverest-dev
EVEREST_OPERATOR_DEV_IMAGE_NAME ?= openeverest-operator-dev
EVEREST_CATALOG_DEV_IMAGE_NAME ?= openeverest-catalog-dev

init:               ## Install dependencies
	npm ci
	npx playwright install

install-operators:  ## Install operators to k8s
	../bin/everestctl install -v \
		--disable-telemetry \
		--version 0.0.0 \
		--version-metadata-url https://check-dev.percona.com \
		--namespaces percona-everest-operators \
		--skip-wizard \
		--helm.set versionMetadataURL=https://check-dev.percona.com \
		--helm.set operator.init=false \
		--helm.set server.image=$(IMAGE_PREFIX)/$(EVEREST_SERVER_DEV_IMAGE_NAME) \
		--helm.set operator.image=$(IMAGE_PREFIX)/$(EVEREST_OPERATOR_DEV_IMAGE_NAME) \
		--helm.set olm.catalogSourceImage=$(IMAGE_PREFIX)/$(EVEREST_CATALOG_DEV_IMAGE_NAME)

test-cli:           ## Run all tests
	npx playwright test --project=cli

test-all-operators:       ## Run mongo operator flow tests. Requires minikube installed and /etc/hosts updated
	npx playwright test tests/flow/all-operators.spec.ts

test-mongo-operator:       ## Run mongo operator flow tests. Requires minikube installed and /etc/hosts updated
	npx playwright test tests/flow/mongodb-operator.spec.ts

test-pg-operator:       ## Run pg operator flow tests. Requires minikube installed and /etc/hosts updated
	npx playwright test tests/flow/pg-operator.spec.ts

test-pxc-operator:       ## Run pxc operator flow tests. Requires minikube installed and /etc/hosts updated
	npx playwright test tests/flow/pxc-operator.spec.ts

test-namespaces:
	npx playwright test tests/flow/namespaces.spec.ts

lint:               ## Run linter typescript checks
	npm run lint:ts && npm run typecheck && echo 'Lint OK'

lint-fix:           ## Fix linter errors where possible
	npx eslint --fix .
