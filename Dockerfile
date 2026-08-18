FROM golang:1.26-alpine@sha256:3889b425f035be855a72fb4755265311293b6d414521f0a519d819df32222d83 AS build

WORKDIR /everest

COPY . .

RUN apk add -U --no-cache ca-certificates

FROM scratch

WORKDIR /

COPY ./bin/everest  /everest-api
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080

ENTRYPOINT ["/everest-api"]
