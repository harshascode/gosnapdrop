# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy source code
COPY . .

# Build with optimization flags
RUN go build -o gosnapdrop

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binary and static files from builder
COPY --from=builder /app/gosnapdrop .
COPY public/ public/

# Set environment variables
ENV GIN_MODE=release
ENV PORT=3000

# Expose port
EXPOSE 3000

# Run the binary
CMD ["./gosnapdrop"]
