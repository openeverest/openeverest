# Scanner image for Anthropic OSS Scanner (https://github.com/anthropics/oss-scanner).
# Builds OpenEverest from the checkout at /src and keeps everything needed to rebuild and test it offline:
# the Go module and build caches (API server, controller, CLI and all test binaries) and the UI's node_modules.
# If go.mod asks for a newer Go than this image has, the toolchain is downloaded here and stays in the module cache.
FROM docker.io/library/golang:1.27-trixie

ENV DEBIAN_FRONTEND=noninteractive \
    CGO_ENABLED=0

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates curl git make jq procps \
    && rm -rf /var/lib/apt/lists/*

# Node.js for the UI (ui/package.json requires Node 22).
COPY --from=docker.io/library/node:22-trixie /usr/local/bin/node /usr/local/bin/node
COPY --from=docker.io/library/node:22-trixie /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s ../lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
    && ln -s ../lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx \
    && node --version

COPY . /src
WORKDIR /src

RUN go mod download

# pnpm at the version in ui/package.json's packageManager field. npm is used instead of corepack because npm
# retries failed downloads.
RUN npm install -g "$(node -p "require('./ui/package.json').packageManager")" && pnpm --version

# UI dependencies and production bundle. The bundle lands in public/dist, which the API server embeds and serves.
RUN cd ui && pnpm install --frozen-lockfile
RUN make build-ui

# The three shipped binaries, built like `make build`, `make build-controller` and `make build-cli`, but unstripped
# so stack traces and debuggers work. The `debug` build tag is not used because it changes runtime behaviour.
RUN go build -o /usr/local/bin/everest ./cmd \
    && go build -o /usr/local/bin/manager ./cmd/controller \
    && go build -o /usr/local/bin/everestctl ./cmd/cli \
    && everestctl version --client-only

# Compile every package and every test binary once, so tests and reproducers rebuild quickly on the small, offline
# scanning machine. `-run '^$'` compiles the tests without running any.
RUN go build ./... && go test -count=1 -run '^$' ./... >/dev/null

# Baseline tests for the security-sensitive areas: accounts, sessions, OIDC, RBAC, plugin tokens, the API server,
# the admission webhooks and the provider runtime. A failing test must not abort the build.
RUN go test -count=1 -timeout 20m \
      ./pkg/accounts/... ./pkg/session/... ./pkg/oidc/... ./pkg/rbac/... ./pkg/plugintoken/... ./pkg/kubernetes/... \
      ./internal/tokenregistry/... ./internal/server/... ./internal/webhook/... ./provider-runtime/... \
    || echo 'WARNING: baseline Go tests reported failures'

# UI unit tests (jsdom). The browser suite needs a downloaded Chromium and is not run here.
RUN cd ui/apps/everest && (pnpm test:unit || echo 'WARNING: UI unit tests reported failures')

# Reproducers and patches must build with the network disabled.
ENV GOPROXY=off \
    GOSUMDB=off

CMD ["bash"]
