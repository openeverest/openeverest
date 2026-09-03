FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b AS dev
WORKDIR /home/everest
RUN adduser -D everest
COPY --chown=everest:everest ./bin/everest  /home/everest/everest-api
COPY --chown=everest:everest ./bin/manager  /home/everest/everest-controller
USER 1000:1000

EXPOSE 8080
ENTRYPOINT ["/home/everest/everest-api"]

# Build the Delve debuger
FROM golang:1.27-alpine@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc AS delve
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
