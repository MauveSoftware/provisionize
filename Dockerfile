FROM --platform=$BUILDPLATFORM golang:1.27.4-alpine3.24@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf AS builder
ADD . /go/provisionize/
WORKDIR /go/provisionize/cmd/provisionize
RUN CGOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o /go/bin/provisionize

FROM alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf
ENV ZipkinEndpoint=""
RUN apk --no-cache add ca-certificates bash && \
    mkdir /app
WORKDIR /app
COPY --from=builder /go/bin/provisionize .
CMD ./provisionize --config=/config/config.yml --zipkin-endpoint=$ZipkinEndpoint
VOLUME /config
EXPOSE 1337
EXPOSE 9500
