# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 ensures a statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -o momail .

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
# tzdata is crucial for correct logging timestamps and cron schedules
RUN apk --no-cache add ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/momail .

# Create directory structure matching config defaults
RUN mkdir -p run/config run/logs run/inbound run/outbound run/temp_inbound run/secure_inbound run/filebox run/nodelist

# Expose ports: 24554 (BinkP) and 8080 (API)
EXPOSE 24554 8080

# Define volume for persistent data (config, logs, mail)
VOLUME ["/app/run"]

# Run the mailer
ENTRYPOINT ["./momail"]
CMD ["-c", "./run/config/config.yaml"]