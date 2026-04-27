FROM golang:1.26-alpine AS build

WORKDIR /everest

COPY . .

RUN apk add -U --no-cache ca-certificates

FROM scratch AS openeverest

WORKDIR /

COPY ./bin/everest  /everest-api
COPY ./bin/manager  /everest-controller
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080

ENTRYPOINT ["/everest-api"]

FROM scratch AS controller

WORKDIR /

COPY ./bin/manager  /everest-controller
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/everest-controller"]
