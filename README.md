# Running in Local Machine

cd /workspaces/gosnapdrop
go mod tidy
go install
go run .

# Running in Production

Docker Deployment
Create a Docker Image:

```
docker build -t gosnapdrop .
docker run -p 3000:3000 -d gosnapdrop
```

Dockerfile
Then build and run the Docker container:

Using go run . is fine for development but not recommended for production because:

It compiles the code every time
It's less efficient than running a compiled binary
It's harder to manage as a service
It doesn't include production optimizations
The best practice is to use the compiled binary with a process manager like systemd or containerize it with Docker.