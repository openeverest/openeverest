FROM golang:1.27-alpine@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc AS build

WORKDIR /everest

COPY . .

RUN apk add -U --no-cache ca-certificates

FROM scratch

WORKDIR /

COPY ./bin/everest  /everest-api
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080

ENTRYPOINT ["/everest-api"]
