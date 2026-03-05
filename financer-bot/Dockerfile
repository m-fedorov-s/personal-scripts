# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o financer.bin .

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app

# Create user AND the data directory with correct permissions
RUN adduser -D appuser && \
    mkdir -p /app/data && \
    chown -R appuser:appuser /app

USER appuser
COPY --from=builder /app/financer.bin .

USER root
RUN apk add --no-cache su-exec
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
