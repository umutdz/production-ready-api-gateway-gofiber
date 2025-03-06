.PHONY: build run test clean docker-build docker-run

# Variables
APP_NAME = api-gateway
MAIN_PATH = ./cmd/gateway
DOCKER_IMAGE = $(APP_NAME):latest

# Build the application
build:
	go build -o $(APP_NAME) $(MAIN_PATH)

# Run the application
run:
	go run $(MAIN_PATH)

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f $(APP_NAME)

# Build Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE) .

# Run Docker container
docker-run:
	docker run -p 8080:8080 $(DOCKER_IMAGE)

# Generate mocks for testing
mocks:
	go generate ./...

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Check for security issues
security:
	gosec ./...

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o $(APP_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 go build -o $(APP_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 go build -o $(APP_NAME)-windows-amd64.exe $(MAIN_PATH)

# Default target
all: clean build
