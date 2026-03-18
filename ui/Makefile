EVEREST_OUT_DIR := ""
OPENAPI_BIN=./node_modules/.bin/openapi-typescript
OPENAPI_SRC=../api/openapi
OPENAPI_OUT=api

.PHONY: init test build lint format dev preflight

init:
	pnpm install

test:
	pnpm test

build:
	EVEREST_OUT_DIR=$(EVEREST_OUT_DIR) pnpm build

lint:
	pnpm lint

format:
	pnpm format

preflight:
	$(MAKE) format
	$(MAKE) lint
	pnpm --filter "@percona/everest" test
	pnpm --filter "@percona/everest" build

# This might change in the future if we have more apps, but for now keep it simple
dev:
	pnpm --filter "@percona/everest" dev

preflight:
	$(MAKE) lint
	$(MAKE) test
	$(MAKE) -C .. copyright-headers
	$(MAKE) format

##@ OpenAPI TypeScript Types Generation

.PHONY: generate-openapi-types
generate-openapi-types: ## Generate TypeScript types from all OpenAPI YAML files in api/openapi/.
	mkdir -p $(OPENAPI_OUT)
	@for f in $(OPENAPI_SRC)/*.yaml; do \
		b=$$(basename "$$f"); \
		echo "Generating $$b -> ui/api/$${b%.yaml}.types.ts"; \
		$(OPENAPI_BIN) "$$f" --output "$(OPENAPI_OUT)/$${b%.yaml}.types.ts"; \
	done
	$(MAKE) generate-openapi-barrel

.PHONY: generate-openapi-type
generate-openapi-type: ## Generate TypeScript types from a specific YAML file. Usage: make generate-openapi-type FILE=crds.gen.yaml
ifndef FILE
	$(error FILE is not set. Usage: make generate-openapi-type FILE=crds.gen.yaml)
endif
	mkdir -p $(OPENAPI_OUT)
	$(OPENAPI_BIN) "$(OPENAPI_SRC)/$(FILE)" --output "$(OPENAPI_OUT)/$(subst .yaml,.types.ts,$(FILE))"
	$(MAKE) generate-openapi-barrel

.PHONY: generate-openapi-barrel
generate-openapi-barrel: ## Regenerate ui/api/index.ts barrel re-exporting all *.types.ts files.
	node generate-barrel.mjs