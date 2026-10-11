# Contexte de build : la racine du dépôt (voir docker-compose.yml), pour embarquer db/.
FROM golang:1.27-alpine AS builder

WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /colec-server .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 colec \
    && mkdir -p /data/uploads \
    && chown colec:colec /data/uploads
COPY --from=builder /colec-server /usr/local/bin/colec-server
COPY db/migrations /app/db/migrations
COPY db/seed.sql db/seed_bd_manga.sql /app/db/
ENV MIGRATIONS_DIR=/app/db/migrations \
    SEED_FILE=/app/db/seed.sql,/app/db/seed_bd_manga.sql \
    UPLOAD_DIR=/data/uploads \
    TZ=UTC
USER colec
EXPOSE 8080
CMD ["colec-server"]
