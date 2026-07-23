# Step 1: Build the Go binary using Alpine Linux
FROM golang:1.22-alpine AS builder

# Install gcc and musl-dev (required for SQLite CGo/ModernC support)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the static Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/main.go

# Step 2: Create tiny production image
FROM alpine:latest  
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy compiled binary from builder stage
COPY --from=builder /app/main .
COPY --from=builder /app/knowledge ./knowledge

# Expose port 8080
EXPOSE 8080

# Run the app
CMD ["./main"]

