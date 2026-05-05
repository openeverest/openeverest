EVEREST_OUT_DIR := ""

.PHONY: init test build lint lint-check format format-check dev preflight

init:
	pnpm install

test:
	pnpm test

build:
	EVEREST_OUT_DIR=$(EVEREST_OUT_DIR) pnpm build

lint:
	pnpm lint

lint-check:
	pnpm lint-check

format:
	pnpm format

format-check:
	pnpm format-check

# This might change in the future if we have more apps, but for now keep it simple
dev:
	pnpm --filter "@percona/everest" dev

preflight:
	$(MAKE) lint
	$(MAKE) test
	$(MAKE) -C .. copyright-headers
	$(MAKE) format