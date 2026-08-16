# Build context is the repo root (this plugin's backend is part of the
# single-module monorepo, github.com/openeverest/openeverest/v2), so this
# is built with:
#   docker build -f plugins/performance-testing/Dockerfile -t performance-testing-backend .
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/backend ./plugins/performance-testing/backend

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/backend /backend
USER nonroot:nonroot
ENTRYPOINT ["/backend"]
