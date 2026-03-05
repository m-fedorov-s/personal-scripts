# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o financer-bot .

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
# Create a non-root user
RUN adduser -D appuser && chown -R appuser /app
USER appuser
COPY --from=builder /app/financer-bot .
CMD ["./financer-bot"]
