FROM alpine:3.22@sha256:310c62b5e7ca5b08167e4384c68db0fd2905dd9c7493756d356e893909057601 AS dev
WORKDIR /home/everest
RUN adduser -D everest
COPY --chown=everest:everest ./bin/everest  /home/everest/everest-api
COPY --chown=everest:everest ./bin/manager  /home/everest/everest-controller
USER 1000:1000

EXPOSE 8080
ENTRYPOINT ["/home/everest/everest-api"]

# Build the Delve debuger
FROM golang:1.26-alpine@sha256:f85330846cde1e57ca9ec309382da3b8e6ae3ab943d2739500e08c86393a21b1 AS delve
RUN go install github.com/go-delve/delve/cmd/dlv@v1.25.2
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
