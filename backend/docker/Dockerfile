FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /logpulse ./cmd/server

# Runtime image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /logpulse /app/logpulse

# Create data directory
RUN mkdir -p /app/data/logs

# Copy configs if they exist
RUN if [ -d configs ]; then cp -r configs/* /app/configs/ 2>/dev/null || true; else mkdir -p /app/configs; fi

EXPOSE 8080

CMD ["/app/logpulse"]
