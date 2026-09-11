FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod ./
COPY . .
RUN go build -o /bin/asana-extractor ./cmd

FROM alpine:3.22

WORKDIR /app
COPY --from=builder /bin/asana-extractor /usr/local/bin/asana-extractor
COPY config/configExample.yaml /app/config/configExample.yaml
COPY infra/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

VOLUME ["/app/output"]

ENTRYPOINT ["sh", "/usr/local/bin/docker-entrypoint.sh"]
CMD ["--config", "/app/config/config.yaml"]
