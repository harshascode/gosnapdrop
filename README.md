# Gosnapdrop

A Go implementation of Snapdrop, allowing instant file sharing between devices on the same network.

## Local Development Setup

1. Clone the repository
```bash
git clone https://github.com/harshascode/gosnapdrop.git
cd gosnapdrop
```

2. Install dependencies and run
```bash
go mod tidy
go run .
```

The server will start on `http://localhost:3000`

## Production Deployment

### Option 1: Using Docker (Recommended)

1. Build the Docker image:
```bash
docker build -t gosnapdrop .
```

2. Run the container:
```bash
docker run -p 3000:3000 -d gosnapdrop
```

### Option 2: Native Go Binary

1. Build the optimized binary:
```bash
go build -ldflags="-w -s" -o gosnapdrop
```

2. Run the binary:
```bash
./gosnapdrop
```

For production deployments, consider:
- Using a process manager (like systemd) to manage the service
- Setting up a reverse proxy (nginx/caddy) for HTTPS
- Configuring proper logging
- Setting the `PORT` environment variable if needed

## Why Docker for Production?

Docker deployment is recommended because:
- Consistent environment across deployments
- Isolated dependencies
- Easy scaling and updates
- Built-in process management
- Resource containment

## System Requirements

- Go 1.16 or higher
- For Docker deployment: Docker 19.03 or higher