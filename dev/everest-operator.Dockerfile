FROM alpine AS dev
WORKDIR /home/everest
RUN adduser -D everest
USER 1000:1000
COPY ./bin/manager  ./manager
ENTRYPOINT ["./manager"]

# Build the Delve debuger
FROM golang:1.26-alpine AS delve
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
