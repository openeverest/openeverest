# Self-contained Makefile for the PSMDB backup-import discovery job image.
#
# Kept separate from the top-level openeverest Makefile so this whole folder
# (cmd/import/psmdb) can later be lifted out into a dedicated plugin repository
# without untangling shared build infrastructure.
#
# Targets must be run from the openeverest module root so the Docker build
# context includes the packages the binary imports (api/, pkg/, jobspec).
#   make -f cmd/import/psmdb/Makefile docker-build

TARGETOS     ?= linux
TARGETARCH   ?= amd64

.PHONY: build
build: ## Build the import-psmdb binary locally (compile check only; the binary must run in-cluster).
	CGO_ENABLED=0 GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build \
		-ldflags "-s -w" \
		-o ./bin/import-psmdb \
		./cmd/import/psmdb

.PHONY: docker-build
docker-build: ## Build the import-psmdb container image.
	docker build \
		--build-arg TARGETOS=$(TARGETOS) \
		--build-arg TARGETARCH=$(TARGETARCH) \
		-f cmd/import/psmdb/Dockerfile \
		-t openeverest-import-psmdb:dev \
		.
