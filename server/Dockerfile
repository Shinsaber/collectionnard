# Contexte de build : la racine du dépôt (voir docker-compose.yml), pour embarquer db/.
# Multi-architecture (CI : linux/amd64 et linux/arm64) : le binaire est compilé sur la plateforme
# de build pour la cible (GOOS/GOARCH), sans émulation ; seule l'image finale est par architecture.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder
ARG TARGETOS TARGETARCH

WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /colec-server .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 colec \
    && mkdir -p /data/uploads \
    && chown colec:colec /data/uploads
COPY --from=builder /colec-server /usr/local/bin/colec-server
COPY db/migrations /app/db/migrations
COPY db/seed.sql db/seed_bd_manga.sql db/seed_music.sql db/seed_video.sql db/seed_games.sql /app/db/
ENV MIGRATIONS_DIR=/app/db/migrations \
    SEED_FILE=/app/db/seed.sql,/app/db/seed_bd_manga.sql,/app/db/seed_music.sql,/app/db/seed_video.sql,/app/db/seed_games.sql \
    UPLOAD_DIR=/data/uploads \
    TZ=UTC
USER colec
EXPOSE 8080
CMD ["colec-server"]
