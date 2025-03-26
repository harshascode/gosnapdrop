FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o gosnapdrop

# Create final minimal image
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/gosnapdrop .

EXPOSE 8080

CMD ["./gosnapdrop"]
