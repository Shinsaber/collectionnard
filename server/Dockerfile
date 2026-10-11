FROM golang:1.27-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /colec-server .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
RUN mkdir -p /data/uploads
COPY --from=builder /colec-server /usr/local/bin/colec-server
EXPOSE 8080
CMD ["colec-server"]
