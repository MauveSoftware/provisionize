FROM --platform=$BUILDPLATFORM golang:1.27.2-alpine3.24@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS builder
ADD . /go/provisionize/
WORKDIR /go/provisionize/cmd/provisionize
RUN CGOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o /go/bin/provisionize

FROM alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a
ENV ZipkinEndpoint=""
RUN apk --no-cache add ca-certificates bash && \
    mkdir /app
WORKDIR /app
COPY --from=builder /go/bin/provisionize .
CMD ./provisionize --config=/config/config.yml --zipkin-endpoint=$ZipkinEndpoint
VOLUME /config
EXPOSE 1337
EXPOSE 9500
