# This image is used to run everestctl in Helm chart hooks (e.g. pre-upgrade checks).
# Stage 1: Build the everestctl binary inside Docker
FROM golang:1.26-alpine@sha256:f85330846cde1e57ca9ec309382da3b8e6ae3ab943d2739500e08c86393a21b1 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=0.0.0
ARG RELEASE_FULLCOMMIT=unknown

WORKDIR /everest

# Copy all source needed for CLI build (go mod download uses network, not build context)
COPY go.mod go.sum ./
COPY cmd/ cmd/
COPY commands/ commands/
COPY internal/ internal/
COPY pkg/ pkg/
COPY api/ api/
COPY client/ client/
COPY data/ data/

# Create the required stub for embedded UI asset
RUN mkdir -p ./public/dist && touch ./public/dist/index.html

RUN go mod download && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags "-s -w \
      -X 'github.com/percona/everest/pkg/version.Version=${VERSION}' \
      -X 'github.com/percona/everest/pkg/version.FullCommit=${RELEASE_FULLCOMMIT}' \
      -X 'github.com/percona/everest/pkg/version.ProjectName=everestctl'" \
    -o /usr/local/bin/everestctl \
    ./cmd/cli

# Stage 2: Minimal runtime image
FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

COPY --from=builder /usr/local/bin/everestctl /usr/local/bin/everestctl

ENTRYPOINT ["everestctl"]
