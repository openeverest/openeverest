FROM golang:1.27-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS build

WORKDIR /everest

COPY . .

RUN apk add -U --no-cache ca-certificates

FROM scratch

WORKDIR /

COPY ./bin/everest  /everest-api
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080

ENTRYPOINT ["/everest-api"]
