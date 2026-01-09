# FaaS Platform Folder Structure

## Project Layout

```
faas-platform/
├── cmd/                           # Application entry points
│   ├── api-gateway/              # API Gateway service
│   │   └── main.go
│   ├── controller/               # Control plane service
│   │   └── main.go
│   ├── worker/                   # Worker service
│   │   └── main.go
│   ├── scheduler/                # Scheduler service
│   │   └── main.go
│   └── cli/                      # CLI tool for administration
│       └── main.go
├── internal/                     # Private application code
│   ├── api/                      # API layer
│   │   ├── gateway/              # API Gateway implementation
│   │   │   ├── middleware/       # Authentication, rate limiting, etc.
│   │   │   │   ├── auth.go
│   │   │   │   ├── ratelimit.go
│   │   │   │   ├── logging.go
│   │   │   │   └── validation.go
│   │   │   ├── handler.go        # Gateway HTTP handlers
│   │   │   └── server.go         # HTTP server setup
│   │   ├── controller/           # Control plane API
│   │   │   ├── function.go       # Function management handlers
│   │   │   ├── invocation.go     # Invocation handlers
│   │   │   ├── admin.go          # Admin API handlers
│   │   │   └── server.go         # HTTP server setup
│   │   └── common/               # Shared API utilities
│   │       ├── response.go       # Standard response formats
│   │       ├── validation.go     # Request validation
│   │       └── errors.go         # Error handling
│   ├── core/                     # Core business logic
│   │   ├── function/             # Function domain
│   │   │   ├── service.go        # Function service implementation
│   │   │   ├── repository.go     # Function repository interface
│   │   │   ├── models.go         # Function domain models
│   │   │   └── validation.go     # Function validation logic
│   │   ├── invocation/           # Invocation domain
│   │   │   ├── service.go        # Invocation service implementation
│   │   │   ├── repository.go     # Invocation repository interface
│   │   │   ├── models.go         # Invocation domain models
│   │   │   └── executor.go       # Execution orchestration
│   │   └── auth/                 # Authentication domain
│   │       ├── service.go        # Auth service implementation
│   │       ├── models.go         # User/auth models
│   │       └── jwt.go            # JWT handling
│   ├── scheduler/                # Scheduling logic
│   │   ├── scheduler.go          # Main scheduler implementation
│   │   ├── queue.go              # Queue management
│   │   ├── balancer.go           # Load balancing logic
│   │   └── models.go             # Scheduler models
│   ├── worker/                   # Worker implementation
│   │   ├── worker.go             # Main worker implementation
│   │   ├── executor.go           # Function execution logic
│   │   ├── runtime/              # Runtime implementations
│   │   │   ├── runtime.go        # Runtime interface
│   │   │   ├── go.go             # Go runtime
│   │   │   ├── python.go         # Python runtime
│   │   │   ├── nodejs.go         # Node.js runtime
│   │   │   └── container.go      # Container management
│   │   └── models.go             # Worker models
│   ├── storage/                  # Storage implementations
│   │   ├── function/             # Function storage
│   │   │   ├── s3.go             # S3 implementation
│   │   │   ├── local.go          # Local filesystem implementation
│   │   │   └── interface.go      # Storage interface
│   │   ├── metadata/             # Metadata storage
│   │   │   ├── postgres.go       # PostgreSQL implementation
│   │   │   ├── memory.go         # In-memory implementation (testing)
│   │   │   └── interface.go      # Repository interfaces
│   │   └── cache/                # Caching layer
│   │       ├── redis.go          # Redis implementation
│   │       ├── memory.go         # In-memory cache
│   │       └── interface.go      # Cache interface
│   ├── messaging/                # Message queue implementations
│   │   ├── redis.go              # Redis queue implementation
│   │   ├── memory.go             # In-memory queue (testing)
│   │   └── interface.go          # Queue interface
│   ├── observability/            # Observability components
│   │   ├── metrics/              # Metrics collection
│   │   │   ├── prometheus.go     # Prometheus implementation
│   │   │   ├── noop.go           # No-op implementation
│   │   │   └── interface.go      # Metrics interface
│   │   ├── logging/              # Structured logging
│   │   │   ├── zap.go            # Zap logger implementation
│   │   │   ├── logrus.go         # Logrus implementation
│   │   │   └── interface.go      # Logger interface
│   │   └── tracing/              # Distributed tracing
│   │       ├── jaeger.go         # Jaeger implementation
│   │       ├── noop.go           # No-op implementation
│   │       └── interface.go      # Tracer interface
│   └── config/                   # Configuration management
│       ├── config.go             # Configuration structures
│       ├── loader.go             # Configuration loading
│       └── validation.go         # Configuration validation
├── pkg/                          # Public library code
│   ├── client/                   # Client SDK
│   │   ├── client.go             # Main client implementation
│   │   ├── function.go           # Function management client
│   │   ├── invocation.go         # Invocation client
│   │   └── models.go             # Client models
│   ├── errors/                   # Error definitions
│   │   ├── errors.go             # Common error types
│   │   └── codes.go              # Error codes
│   ├── types/                    # Shared types
│   │   ├── function.go           # Function types
│   │   ├── invocation.go         # Invocation types
│   │   ├── runtime.go            # Runtime types
│   │   └── common.go             # Common types
│   └── utils/                    # Utility functions
│       ├── crypto.go             # Cryptographic utilities
│       ├── validation.go         # Validation utilities
│       ├── http.go               # HTTP utilities
│       └── time.go               # Time utilities
├── deployments/                  # Deployment configurations
│   ├── docker/                   # Docker configurations
│   │   ├── Dockerfile.gateway    # API Gateway Dockerfile
│   │   ├── Dockerfile.controller # Controller Dockerfile
│   │   ├── Dockerfile.worker     # Worker Dockerfile
│   │   ├── Dockerfile.scheduler  # Scheduler Dockerfile
│   │   └── docker-compose.yml    # Local development setup
│   ├── kubernetes/               # Kubernetes manifests
│   │   ├── namespace.yaml
│   │   ├── configmap.yaml
│   │   ├── secrets.yaml
│   │   ├── api-gateway.yaml
│   │   ├── controller.yaml
│   │   ├── worker.yaml
│   │   ├── scheduler.yaml
│   │   ├── redis.yaml
│   │   ├── postgres.yaml
│   │   └── ingress.yaml
│   └── helm/                     # Helm charts
│       └── faas-platform/
│           ├── Chart.yaml
│           ├── values.yaml
│           └── templates/
├── scripts/                      # Build and deployment scripts
│   ├── build.sh                  # Build script
│   ├── test.sh                   # Test script
│   ├── deploy.sh                 # Deployment script
│   └── migrate.sh                # Database migration script
├── migrations/                   # Database migrations
│   ├── 001_initial_schema.up.sql
│   ├── 001_initial_schema.down.sql
│   ├── 002_add_function_versions.up.sql
│   └── 002_add_function_versions.down.sql
├── docs/                         # Documentation
│   ├── api/                      # API documentation
│   │   ├── openapi.yaml          # OpenAPI specification
│   │   └── README.md
│   ├── deployment/               # Deployment guides
│   │   ├── kubernetes.md
│   │   ├── docker.md
│   │   └── production.md
│   └── development/              # Development guides
│       ├── getting-started.md
│       ├── architecture.md
│       └── contributing.md
├── test/                         # Test files
│   ├── integration/              # Integration tests
│   │   ├── api_test.go
│   │   ├── worker_test.go
│   │   └── e2e_test.go
│   ├── fixtures/                 # Test fixtures
│   │   ├── functions/            # Sample functions
│   │   │   ├── hello.go
│   │   │   ├── hello.py
│   │   │   └── hello.js
│   │   └── data/                 # Test data
│   └── mocks/                    # Mock implementations
│       ├── storage.go
│       ├── queue.go
│       └── runtime.go
├── examples/                     # Example code and tutorials
│   ├── functions/                # Example functions
│   │   ├── go/
│   │   ├── python/
│   │   └── nodejs/
│   ├── clients/                  # Example client usage
│   │   ├── go/
│   │   ├── python/
│   │   └── curl/
│   └── deployments/              # Example deployments
├── tools/                        # Development tools
│   ├── codegen/                  # Code generation tools
│   ├── linter/                   # Custom linters
│   └── benchmarks/               # Performance benchmarks
├── .github/                      # GitHub workflows
│   └── workflows/
│       ├── ci.yml
│       ├── cd.yml
│       └── security.yml
├── go.mod                        # Go module definition
├── go.sum                        # Go module checksums
├── Makefile                      # Build automation
├── README.md                     # Project documentation
├── LICENSE                       # License file
└── .gitignore                    # Git ignore rules
```

## Key Design Decisions

### 1. Clean Architecture Principles
- **Separation of Concerns**: Each layer has a single responsibility
- **Dependency Inversion**: Core business logic doesn't depend on external frameworks
- **Interface Segregation**: Small, focused interfaces for better testability

### 2. Domain-Driven Design
- **Core Domains**: Function, Invocation, Auth as separate bounded contexts
- **Repository Pattern**: Abstract data access behind interfaces
- **Service Layer**: Encapsulates business logic and orchestration

### 3. Hexagonal Architecture
- **Ports and Adapters**: Clear boundaries between business logic and external systems
- **Pluggable Components**: Easy to swap implementations (storage, messaging, etc.)
- **Testability**: Mock external dependencies for unit testing

### 4. Microservices Patterns
- **Service per Concern**: Gateway, Controller, Worker, Scheduler as separate services
- **Shared Libraries**: Common types and utilities in `pkg/`
- **Configuration Management**: Centralized configuration with environment overrides

## Naming Conventions

### Packages
- Use lowercase, single words when possible
- Use descriptive names that indicate purpose
- Avoid generic names like `utils`, `common` unless necessary

### Files
- Use snake_case for file names
- Group related functionality in the same file
- Use `_test.go` suffix for test files
- Use `interface.go` for interface definitions

### Types and Functions
- Use PascalCase for exported types and functions
- Use camelCase for unexported types and functions
- Use descriptive names that indicate purpose
- Prefix interfaces with the domain name when needed

### Constants and Variables
- Use UPPER_SNAKE_CASE for constants
- Use camelCase for variables
- Group related constants in const blocks

## Module Dependencies

### External Dependencies
```go
// Core dependencies
github.com/gorilla/mux           // HTTP routing
github.com/go-redis/redis/v8     // Redis client
github.com/lib/pq                // PostgreSQL driver
github.com/aws/aws-sdk-go-v2     // AWS SDK for S3

// Observability
github.com/prometheus/client_golang  // Prometheus metrics
go.uber.org/zap                     // Structured logging
github.com/opentracing/opentracing-go // Distributed tracing

// Authentication
github.com/golang-jwt/jwt/v4        // JWT handling
golang.org/x/crypto                 // Cryptographic functions

// Container runtime
github.com/docker/docker            // Docker client
github.com/containers/podman/v4     // Podman client (alternative)

// Configuration
github.com/spf13/viper             // Configuration management
github.com/spf13/cobra             // CLI framework

// Testing
github.com/stretchr/testify        // Testing utilities
github.com/golang/mock             // Mock generation
```

### Internal Dependencies
- `internal/` packages should not import from `cmd/`
- `pkg/` packages should not import from `internal/`
- Core business logic should not depend on external frameworks
- Use dependency injection for external dependencies

## Build and Development

### Makefile Targets
```makefile
.PHONY: build test lint clean docker deploy

# Build all services
build:
	go build -o bin/api-gateway cmd/api-gateway/main.go
	go build -o bin/controller cmd/controller/main.go
	go build -o bin/worker cmd/worker/main.go
	go build -o bin/scheduler cmd/scheduler/main.go

# Run tests
test:
	go test -v ./...

# Run integration tests
test-integration:
	go test -v -tags=integration ./test/integration/...

# Lint code
lint:
	golangci-lint run

# Build Docker images
docker:
	docker build -f deployments/docker/Dockerfile.gateway -t faas/api-gateway .
	docker build -f deployments/docker/Dockerfile.controller -t faas/controller .
	docker build -f deployments/docker/Dockerfile.worker -t faas/worker .
	docker build -f deployments/docker/Dockerfile.scheduler -t faas/scheduler .

# Deploy to Kubernetes
deploy:
	kubectl apply -f deployments/kubernetes/

# Clean build artifacts
clean:
	rm -rf bin/
	docker system prune -f
```

This folder structure provides a solid foundation for building a production-ready FaaS platform with clear separation of concerns, testability, and maintainability.