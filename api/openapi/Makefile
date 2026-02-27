OPENAPI_BIN=./node_modules/.bin/openapi-typescript

##@ General

.PHONY: default
default: help

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ TypeScript Types Generation

.PHONY: init
init: ## Install dependencies for TypeScript types generation.
	pnpm install

.PHONY: generate-all
generate-all: ## Generate TypeScript types from all OpenAPI YAML files in this directory.
	mkdir -p generated
	@for f in *.yaml; do \
		case "$$f" in pnpm-lock.yaml) continue ;; esac; \
		echo "Generating $$f -> generated/$${f%.yaml}.types.ts"; \
		$(OPENAPI_BIN) "$$f" --output "./generated/$${f%.yaml}.types.ts"; \
	done
	$(MAKE) generate-barrel

.PHONY: generate
generate: ## Generate TypeScript types from a specific YAML file. Usage: make generate FILE=crds.gen.yaml
ifndef FILE
	$(error FILE is not set. Usage: make generate FILE=crds.gen.yaml)
endif
	mkdir -p generated
	$(OPENAPI_BIN) "$(FILE)" --output "./generated/$(subst .yaml,.types.ts,$(FILE))"
	$(MAKE) generate-barrel

.PHONY: generate-barrel
generate-barrel: ## Regenerate generated/index.ts barrel that re-exports all *.types.ts files.
	node generate-barrel.mjs

.PHONY: clean
clean: ## Remove generated files and node_modules.
	rm -rf generated/ node_modules pnpm-lock.yaml
