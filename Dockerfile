FROM golang as builder
RUN go get github.com/MauveSoftware/provisionize/cmd/provisionize

FROM alpine:latest
ENV ZipkinEndpoint ""
RUN apk --no-cache add ca-certificates bash && \
    mkdir /app
WORKDIR /app
COPY --from=builder /go/bin/provisionize .
CMD ./provisionize -config-file=/config/config.yml -zipkin-endpoint=$ZipkinEndpoint
VOLUME /config
EXPOSE 1337
EXPOSE 9500