# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY app/go.mod app/go.sum ./
RUN go mod download

# Copy source code
COPY app/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o wellness-nutrition .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /build/wellness-nutrition .

# Expose port
EXPOSE 3000

# Run the application
CMD ["./wellness-nutrition", "-listen-addr=:3000"]
