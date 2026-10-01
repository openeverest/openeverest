# Self-contained image for the PSMDB backup-import discovery job.
# This binary scans an S3 storage location for PBM backup metadata and creates
# external Backup CRs. It is intentionally packaged on its own (own Dockerfile +
# Makefile) so it can later be lifted out into a dedicated plugin repository.
#
# NOTE: build context must be the openeverest module root, because the binary
# imports openeverest packages (api/, pkg/common, jobspec).
#   docker build -f cmd/import/psmdb/Dockerfile .

# Stage 1: build the discovery binary.
FROM golang:1.26-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src

RUN apk add -U --no-cache ca-certificates

# Copy only what the binary needs to compile.
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY api/ api/
COPY pkg/ pkg/

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags "-s -w" \
    -o /usr/local/bin/import-psmdb \
    ./cmd/import/psmdb

# Stage 2: minimal runtime image.
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/local/bin/import-psmdb /import-psmdb

ENTRYPOINT ["/import-psmdb"]
