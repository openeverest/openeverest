# This image is used to run Everest Helm chart hooks.
FROM alpine:3.22
ARG TARGETOS
ARG TARGETARCH

ARG KUBECTL_VERSION=v1.33.5

# Install kubectl
RUN apk add --no-cache curl \
    && curl -LO "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/${TARGETOS}/${TARGETARCH}/kubectl" \
    && install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl \
    && rm kubectl \
    && apk del curl
