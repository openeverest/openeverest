EVEREST_OUT_DIR := ""

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

# This might change in the future if we have more apps, but for now keep it simple
dev:
	pnpm --filter "@percona/everest" dev

preflight:
	$(MAKE) lint
	$(MAKE) test
	$(MAKE) -C .. copyright-headers
	$(MAKE) format