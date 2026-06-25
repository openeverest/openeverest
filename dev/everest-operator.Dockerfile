FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b AS dev
WORKDIR /home/everest
RUN adduser -D everest
USER 1000:1000
COPY ./bin/manager  ./manager
ENTRYPOINT ["./manager"]

# Build the Delve debuger
FROM golang:1.26-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648 AS delve
RUN go install github.com/go-delve/delve/cmd/dlv@v1.25.2
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
