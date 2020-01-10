FROM golang as builder
ADD . /go/provisionize/
WORKDIR /go/provisionize/cmd/provisionize
RUN CGOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o /go/bin/provisionize

FROM alpine:latest
ENV ZipkinEndpoint ""
RUN apk --no-cache add ca-certificates bash && \
    mkdir /app
WORKDIR /app
COPY --from=builder /go/bin/provisionize .
CMD ./provisionize --config=/config/config.yml --zipkin-endpoint=$ZipkinEndpoint
VOLUME /config
EXPOSE 1337
EXPOSE 9500
