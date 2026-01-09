# FaaS Platform

A production-ready Function-as-a-Service (FaaS) platform built with Go, implementing a distributed controller-worker architecture with async message queuing.

## Architecture

The system consists of three main components:

1. **Controller**: HTTP API server for function management and invocation requests
2. **Worker**: Background processes that execute functions from the queue
3. **Infrastructure**: PostgreSQL (metadata), Redis (message queue), Local/S3 storage (function code)

## Features

- ✅ Function lifecycle management (CRUD operations)
- ✅ Async function invocation with result tracking
- ✅ Multi-runtime support (Go, Python, Node.js)
- ✅ **Container-based execution with Docker** (NEW)
- ✅ Resource limits (memory, CPU, timeout)
- ✅ Reliable message queuing with Redis
- ✅ PostgreSQL-backed metadata storage
- ✅ Configurable timeouts and resource limits
- ✅ Structured logging
- ✅ Graceful shutdown

## Prerequisites

- Go 1.22.5 or higher
- PostgreSQL 15+
- Redis 7+
- **Docker Engine** (for container-based execution)
- Docker & Docker Compose (for local development)

## Quick Start

### 1. Start Infrastructure

```bash
# Start PostgreSQL and Redis
make docker-up

# Run database migrations
make migrate-up

# Build runtime images for container execution
make build-runtime-images
```

### 2. Start Services

```bash
# Terminal 1: Start controller
make run-controller

# Terminal 2: Start worker
make run-worker
```

### 3. Create a Function

```bash
# Create a simple Go function
curl -X POST http://localhost:8080/functions \
  -H "Content-Type: application/json" \
  -d '{
    "name": "hello",
    "version": "1.0.0",
    "runtime": "go",
    "handler": "main",
    "code": "cGFja2FnZSBtYWluCgppbXBvcnQgImZtdCIKCmZ1bmMgbWFpbigpIHsKCWZtdC5QcmludGxuKCJIZWxsbywgRmFhUyEiKQp9",
    "timeout": "30s",
    "memory_mb": 128,
    "max_concurrency": 10,
    "environment": {},
    "metadata": {}
  }'
```

### 4. Invoke the Function

```bash
# Invoke function asynchronously
curl -X POST http://localhost:8080/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "function_id": "<function-id-from-step-3>",
    "payload": {},
    "headers": {}
  }'

# Get invocation result
curl http://localhost:8080/invocations/<invocation-id>
```

## Configuration

Configuration is done via environment variables:

### Server Configuration
- `SERVER_ADDR`: HTTP server address (default: `:8080`)

### Database Configuration
- `DB_HOST`: PostgreSQL host (default: `localhost`)
- `DB_PORT`: PostgreSQL port (default: `5432`)
- `DB_USER`: Database user (default: `postgres`)
- `DB_PASSWORD`: Database password (default: `postgres`)
- `DB_NAME`: Database name (default: `faas`)
- `DB_SSLMODE`: SSL mode (default: `disable`)

### Redis Configuration
- `REDIS_ADDR`: Redis address (default: `localhost:6379`)
- `REDIS_PASSWORD`: Redis password (default: empty)
- `REDIS_DB`: Redis database number (default: `0`)

### Storage Configuration
- `STORAGE_TYPE`: Storage type (default: `local`)
- `STORAGE_BASE_DIR`: Base directory for function storage (default: `./storage/functions`)

### Worker Configuration
- `WORKER_ID`: Worker identifier (default: auto-generated)
- `WORKER_WORK_DIR`: Worker work directory (default: `./storage/work`)
- `WORKER_USE_CONTAINER`: Enable container execution (default: `true`)
- `WORKER_RUNTIME_TYPE`: Runtime type - "simple" or "container" (default: `container`)

## API Endpoints

### Function Management

- `POST /functions` - Create a new function
- `GET /functions` - List all functions
- `GET /functions/{id}` - Get function by ID
- `PUT /functions/{id}` - Update function
- `DELETE /functions/{id}` - Delete function

### Function Invocation

- `POST /invoke` - Invoke a function asynchronously
- `GET /invocations/{id}` - Get invocation result
- `GET /invocations` - List invocations

### Health Check

- `GET /health` - Health check endpoint

## Development

### Build

```bash
make build
```

### Run Tests

```bash
make test
```

### Format Code

```bash
make fmt
```

### Clean

```bash
make clean
```

## Project Structure

```
faas-platform/
├── cmd/                      # Application entry points
│   ├── controller/          # Controller service
│   └── worker/              # Worker service
├── internal/                # Private application code
│   ├── api/                 # API layer
│   ├── config/              # Configuration
│   ├── core/                # Core business logic
│   ├── messaging/           # Message queue
│   ├── observability/       # Logging, metrics
│   ├── storage/             # Storage implementations
│   └── worker/              # Worker and runtime
├── pkg/                     # Public library code
│   ├── errors/              # Error definitions
│   ├── types/               # Shared types
│   └── utils/               # Utility functions
├── migrations/              # Database migrations
├── docker-compose.yml       # Local development setup
├── Makefile                 # Build automation
└── README.md               # This file
```

## Design Documents

See the following documents for detailed architecture and implementation guidance:

- `REBUILD_ARCHITECTURE.md` - Complete system architecture
- `FOLDER_STRUCTURE.md` - Project structure and conventions
- `IMPLEMENTATION_GUIDE.md` - Step-by-step implementation guide
- `CONTAINER_EXECUTION.md` - **Container-based execution guide** (NEW)

## License

MIT License
