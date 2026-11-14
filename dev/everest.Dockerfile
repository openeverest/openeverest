FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS dev
WORKDIR /home/everest
RUN adduser -D everest
USER 1000:1000
COPY ./bin/everest  ./everest-api

EXPOSE 8080
ENTRYPOINT ["./everest-api"]

# Build the Delve debuger
FROM golang:1.27-alpine@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS delve
RUN go install github.com/go-delve/delve/cmd/dlv@v1.27.2
RUN chmod +x /go/bin/dlv

# Build the image with debuger
FROM dev AS debug
COPY --from=delve /go/bin/dlv /dlv
WORKDIR /
USER root

# Expose Delve port
EXPOSE 40000
ENTRYPOINT [ "/dlv", \
    "--listen=:40000", \
    "--headless=true", \
    "--api-version=2", \
    "--continue=true", \
    "--accept-multiclient=true", \
    "exec", \
    "/home/everest/everest-api", \
    "--"]
