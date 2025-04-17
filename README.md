# API Gateway

Production-ready API Gateway built with Go Fiber, designed for high performance and reliability.

## Features

- **Routing & Service Discovery**: Dynamic routing and service discovery for microservices
- **Proxy Support**: HTTP and WebSocket proxy with efficient connection handling
- **Security**: JWT/API key authentication, TLS/SSL, XSS & CSRF protection
- **Resilience**: Circuit breaker patterns using Sony GoBreaker, retry mechanisms with Eapache Resiliency, timeout management
- **Performance**: Response caching, Gzip compression, connection pooling
- **Monitoring**: Prometheus metrics, structured logging with Zap, OpenTelemetry distributed tracing
- **Kubernetes Integration**: Ready-to-use K8s configurations for production and local development

## Project Structure

```
.
├── cmd/                    # Application entry points
│   └── gateway/            # Main API Gateway application
│       └── main.go         # Application bootstrap and initialization
├── internal/               # Private application code
│   ├── config/             # Configuration management using Viper
│   ├── middleware/         # Custom middleware for request processing
│   ├── proxy/              # Proxy implementations (HTTP, WebSocket)
│   ├── resilience/         # Resilience patterns (circuit breaker, retry)
│   ├── router/             # Dynamic routing and service discovery
│   ├── security/           # Security implementations (JWT, API keys)
│   └── server/             # Server setup and initialization
├── pkg/                    # Public libraries that can be used by external applications
│   ├── cache/              # Caching mechanisms
│   ├── http/               # HTTP utilities
│   ├── logging/            # Logging utilities with Zap
│   ├── metrics/            # Metrics collection with Prometheus
│   └── tracing/            # Distributed tracing with OpenTelemetry
├── api/                    # API definitions and documentation
├── deployments/            # Deployment configurations
│   ├── kubernetes/         # Kubernetes manifests
│   │   └── local/          # Local development configurations
│   ├── prometheus/         # Prometheus configuration
│   └── grafana/            # Grafana dashboards and configuration
├── scripts/                # Utility scripts
├── test/                   # Additional test applications and test data
├── config.yaml             # Default configuration file
├── go.mod                  # Go module definition
├── go.sum                  # Go module checksums
├── Makefile                # Build automation
├── Dockerfile              # Container definition
└── README.md               # Project documentation
```

## Technology Stack

- **Web Framework**: Go Fiber v2.52.6
- **Configuration**: Viper v1.19.0
- **Logging**: Zap v1.27.0
- **Metrics**: Prometheus Client v1.21.0
- **Tracing**: OpenTelemetry v1.35.0
- **Resilience**: Sony GoBreaker v1.0.0, Eapache Resiliency v1.7.0
- **JWT Authentication**: golang-jwt/jwt v5.2.1
- **WebSocket Support**: fasthttp/websocket v1.5.10, gofiber/websocket v2.2.1

## Getting Started

### Prerequisites

- Go 1.23+
- Docker and Docker Compose (for containerization)
- Kubernetes and Kind (for local cluster deployment)

### Local Development Setup

#### Running with Go

1. Clone the repository
2. Install dependencies: `go mod download`
3. Build the application: `make build`
4. Run the application: `make run`

#### Running with Docker

```bash
# Build the Docker image
make docker-build

# Run the container
make docker-run
```

### Local Kubernetes Setup

We provide scripts for setting up a local Kubernetes environment using Kind:

```bash
# Create a Kind cluster
kind create cluster --name crash-game --config kind-config.yaml

# Install NGINX Ingress Controller
kubectl apply -f https://kind.sigs.k8s.io/examples/ingress/deploy-ingress-nginx.yaml

# Wait for Ingress controller to be ready
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s

# Create local namespace
kubectl create namespace crash-game-backend-local

# Create dependencies directory if it doesn't exist
mkdir -p .gitlab/kubelets/local/dependencies

# Deploy dependencies
kubectl apply -f .gitlab/kubelets/local/dependencies/

# Apply configMap
kubectl apply -f .gitlab/kubelets/local/configmap.yaml

# Build admin service (if needed for your setup)
docker build -t crash-game-backend-admin:latest -f admin_service/compose/Dockerfile .

# Load image to Kind cluster
kind load docker-image crash-game-backend-admin:latest --name crash-game
```

### Accessing Pods and Services

```bash
# Execute commands in a pod
kubectl exec -it $(kubectl get pod -n crash-game-backend-local -l app=admin-service -o jsonpath='{.items[0].metadata.name}') -n crash-game-backend-local -- bash

# Configure local hostname for testing
echo "127.0.0.1 local-api.crash-game" | sudo tee -a /etc/hosts
cat /etc/hosts | grep local-admin.crash-game

# Disable validation webhook (for sticky sessions)
kubectl delete validatingwebhookconfiguration ingress-nginx-admission
```

### Setting Up Monitoring

```bash
# Configure Grafana
kubectl create configmap grafana-provisioning \
  --from-file=datasources.prometheus.yaml=./deployments/grafana/provisioning/datasources/prometheus.yml \
  --from-file=dashboards.dashboard.yaml=./deployments/grafana/provisioning/dashboards/default.yaml \
  --from-file=dashboards.api-gateway.json=./deployments/grafana/provisioning/dashboards/go-metrics-dashboard.json \
  -n monitoring --dry-run=client -o yaml | kubectl apply -f -

# Configure Prometheus
kubectl create configmap prometheus-config \
  --from-file=prometheus.yml=./deployments/prometheus/prometheus.yml \
  -n monitoring --dry-run=client -o yaml | kubectl apply -f -
```

## Configuration

Configuration is managed through environment variables and/or YAML configuration files. The default configuration file is `config.yaml` in the project root.

Key configuration sections include:

- Server settings (port, timeouts, etc.)
- Proxy settings (upstream services, timeouts)
- Security settings (authentication, TLS)
- Resilience settings (circuit breaker, retries)
- Observability settings (logging, metrics, tracing)

See `internal/config` for details on all available options.

## Development

### Available Make Commands

```bash
# Build the application
make build

# Run the application
make run

# Run tests
make test

# Format code
make fmt

# Lint code
make lint

# Check for security issues
make security

# Generate mocks for testing
make mocks

# Build for multiple platforms
make build-all

# Clean build artifacts
make clean
```

## Deployment

### Docker Deployment

```bash
# Build Docker image
make docker-build

# Run Docker container
make docker-run
```

### Kubernetes Deployment

```bash
# Deploy to Kubernetes
kubectl apply -f deployments/kubernetes/
```

## License

MIT
