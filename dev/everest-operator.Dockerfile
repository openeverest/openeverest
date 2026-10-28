FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS dev
WORKDIR /home/everest
RUN adduser -D everest
USER 1000:1000
COPY ./bin/manager  ./manager
ENTRYPOINT ["./manager"]

# Build the Delve debuger
FROM golang:1.27-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS delve
RUN go install github.com/go-delve/delve/cmd/dlv@v1.27.2
RUN chmod +x /go/bin/dlv

# Build the image with debuger
FROM dev AS debug
COPY --from=delve /go/bin/dlv /dlv
WORKDIR /
USER root

# Expose Delve port
EXPOSE 40001
ENTRYPOINT [ "/dlv", \
    "--listen=:40001", \
    "--headless=true", \
    "--api-version=2", \
    "--continue=true", \
    "--accept-multiclient=true", \
    "exec", \
    "/home/everest/manager", \
    "--"]
