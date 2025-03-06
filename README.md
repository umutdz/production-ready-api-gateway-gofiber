# API Gateway

Production-ready API Gateway built with Go Fiber.

## Features

- **Routing & Service Discovery**: Dynamic routing and service discovery
- **Proxy Support**: HTTP and WebSocket proxy
- **Security**: JWT/API key authentication, TLS/SSL, XSS & CSRF protection
- **Resilience**: Circuit breaker, retry mechanism, timeout management
- **Performance**: Response caching, Gzip compression, connection pooling
- **Monitoring**: Prometheus metrics, structured logging, distributed tracing
- **Kubernetes Integration**: Ready-to-use K8s configurations

## Project Structure

```
.
├── cmd/                    # Application entry points
│   └── gateway/            # Main API Gateway application
├── internal/               # Private application code
│   ├── config/             # Configuration management
│   ├── middleware/         # Custom middleware
│   ├── proxy/              # Proxy implementations (HTTP, WebSocket)
│   ├── resilience/         # Resilience patterns (circuit breaker, retry)
│   ├── router/             # Dynamic routing and service discovery
│   ├── security/           # Security implementations
│   └── server/             # Server setup and initialization
├── pkg/                    # Public libraries that can be used by external applications
│   ├── cache/              # Caching mechanisms
│   ├── logging/            # Logging utilities
│   ├── metrics/            # Metrics collection
│   └── tracing/            # Distributed tracing
├── api/                    # API definitions and documentation
├── deployments/            # Deployment configurations
│   └── kubernetes/         # Kubernetes manifests
├── scripts/                # Utility scripts
├── test/                   # Additional test applications and test data
├── go.mod                  # Go module definition
├── go.sum                  # Go module checksums
├── Makefile                # Build automation
├── Dockerfile              # Container definition
└── README.md               # Project documentation
```

## Getting Started

### Prerequisites

- Go 1.21+
- Docker (for containerization)
- Kubernetes (for deployment)

### Installation

1. Clone the repository
2. Install dependencies: `go mod download`
3. Build the application: `make build`
4. Run the application: `make run`

### Configuration

Configuration is managed through environment variables and/or configuration files. See `internal/config` for details.

## Deployment

### Docker

```bash
make docker-build
make docker-run
```

### Kubernetes

```bash
kubectl apply -f deployments/kubernetes/
```

## License

MIT
