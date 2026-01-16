FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o worker ./cmd/worker

FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/worker .

CMD ["./worker"]
