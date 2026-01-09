# FaaS Platform Rebuild Architecture

## Executive Summary

This document outlines a production-ready rebuild of the FaaS platform, addressing current limitations while maintaining the proven controller-worker pattern. The new architecture emphasizes **security**, **scalability**, **observability**, and **multi-runtime support**.

## Core Design Principles

1. **Security First**: Sandboxed execution, authentication, and authorization
2. **Horizontal Scalability**: Stateless components with distributed storage
3. **Observability**: Comprehensive logging, metrics, and tracing
4. **Reliability**: Error handling, retries, and graceful degradation
5. **Extensibility**: Plugin-based runtimes and middleware architecture

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Load Balancer                              │
└─────────────────────┬───────────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────────┐
│                    API Gateway Layer                                │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Authentication │ Rate Limiting │ Request Validation         │   │
│  │ Authorization  │ Metrics       │ Request/Response Logging   │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────┬───────────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────────┐
│                   Control Plane                                     │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ Function Management API │ Invocation API │ Admin API        │  │
│  │ - CRUD operations       │ - Sync/Async   │ - System health  │  │
│  │ - Versioning           │ - Result fetch  │ - Metrics        │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────┬───────────────────────────────────────────────┘
                      │
                      ├─────────────────┬─────────────────────────────┐
                      │                 │                             │
                      ▼                 ▼                             ▼
        ┌─────────────────────┐ ┌─────────────────┐    ┌─────────────────────┐
        │   Message Queue     │ │   Metadata      │    │   Function Storage  │
        │   (Redis Cluster)   │ │   Database      │    │   (Object Store)    │
        │                     │ │   (PostgreSQL)  │    │   (S3/MinIO)        │
        │ - Execution Queue   │ │ - Functions     │    │ - Function Code     │
        │ - Result Queue      │ │ - Executions    │    │ - Dependencies      │
        │ - Dead Letter Queue │ │ - Users/Auth    │    │ - Runtime Images    │
        └─────────────────────┘ └─────────────────┘    └─────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Data Plane                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    Scheduler                                 │  │
│  │ - Queue Management    │ - Resource Allocation               │  │
│  │ - Load Balancing      │ - Execution Planning                │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                │                                    │
│  ┌─────────────────────────────▼────────────────────────────────┐  │
│  │                  Worker Pool                                 │  │
│  │                                                              │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │  │
│  │  │   Worker 1  │  │   Worker 2  │  │   Worker N  │   ...   │  │
│  │  │             │  │             │  │             │         │  │
│  │  │ ┌─────────┐ │  │ ┌─────────┐ │  │ ┌─────────┐ │         │  │
│  │  │ │Runtime 1│ │  │ │Runtime 2│ │  │ │Runtime N│ │         │  │
│  │  │ │Container│ │  │ │Container│ │  │ │Container│ │         │  │
│  │  │ └─────────┘ │  │ └─────────┘ │  │ └─────────┘ │         │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘         │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                                │
                                ▼
        ┌─────────────────────────────────────────────────────────┐
        │                Runtime Isolation                        │
        │  ┌─────────────────────────────────────────────────┐   │
        │  │ Container Runtime (Docker/Podman)               │   │
        │  │ - Sandboxed execution                           │   │
        │  │ - Resource limits (CPU, Memory, Network)        │   │
        │  │ - Security policies                             │   │
        │  └─────────────────────────────────────────────────┘   │
        └─────────────────────────────────────────────────────────┘
```

## Layer Definitions

### 1. API Gateway Layer

**Responsibilities:**
- Request authentication and authorization
- Rate limiting and throttling
- Request/response validation
- Metrics collection and logging
- SSL termination

**Public Interfaces:**
```go
type Gateway interface {
    // Middleware chain
    Authenticate(next http.Handler) http.Handler
    Authorize(next http.Handler) http.Handler
    RateLimit(next http.Handler) http.Handler
    ValidateRequest(next http.Handler) http.Handler
    LogRequest(next http.Handler) http.Handler
}
```

**Key Data Structures:**
```go
type AuthContext struct {
    UserID    string
    Roles     []string
    Scopes    []string
    ExpiresAt time.Time
}

type RateLimitConfig struct {
    RequestsPerSecond int
    BurstSize         int
    WindowSize        time.Duration
}
```

**Dependencies:**
- Authentication service (JWT/OAuth2)
- Rate limiting store (Redis)
- Metrics collector (Prometheus)

### 2. Control Plane

**Responsibilities:**
- Function lifecycle management (CRUD)
- Invocation request handling
- Result retrieval
- System administration
- API versioning

**Public Interfaces:**
```go
type FunctionService interface {
    CreateFunction(ctx context.Context, req CreateFunctionRequest) (*Function, error)
    GetFunction(ctx context.Context, id string) (*Function, error)
    UpdateFunction(ctx context.Context, id string, req UpdateFunctionRequest) (*Function, error)
    DeleteFunction(ctx context.Context, id string) error
    ListFunctions(ctx context.Context, filter FunctionFilter) ([]*Function, error)
}

type InvocationService interface {
    InvokeSync(ctx context.Context, req InvocationRequest) (*InvocationResult, error)
    InvokeAsync(ctx context.Context, req InvocationRequest) (*InvocationHandle, error)
    GetResult(ctx context.Context, invocationID string) (*InvocationResult, error)
    ListInvocations(ctx context.Context, filter InvocationFilter) ([]*Invocation, error)
}
```

**Key Data Structures:**
```go
type Function struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Version     string            `json:"version"`
    Runtime     Runtime           `json:"runtime"`
    Handler     string            `json:"handler"`
    Code        FunctionCode      `json:"code"`
    Config      FunctionConfig    `json:"config"`
    Metadata    map[string]string `json:"metadata"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}

type FunctionCode struct {
    Source      string `json:"source"`       // Base64 encoded or S3 URL
    SourceType  string `json:"source_type"`  // "inline", "s3", "git"
    Checksum    string `json:"checksum"`     // SHA256 hash
    Size        int64  `json:"size"`         // Bytes
}

type FunctionConfig struct {
    Timeout     time.Duration     `json:"timeout"`
    Memory      int               `json:"memory_mb"`
    Environment map[string]string `json:"environment"`
    Concurrency int               `json:"max_concurrency"`
}

type InvocationRequest struct {
    FunctionID string                 `json:"function_id"`
    Payload    json.RawMessage        `json:"payload"`
    Headers    map[string]string      `json:"headers"`
    Async      bool                   `json:"async"`
    Timeout    *time.Duration         `json:"timeout,omitempty"`
}

type InvocationResult struct {
    ID          string          `json:"id"`
    FunctionID  string          `json:"function_id"`
    Status      ExecutionStatus `json:"status"`
    Result      json.RawMessage `json:"result,omitempty"`
    Error       *ExecutionError `json:"error,omitempty"`
    Metrics     ExecutionMetrics `json:"metrics"`
    StartedAt   time.Time       `json:"started_at"`
    CompletedAt *time.Time      `json:"completed_at,omitempty"`
}
```

**Dependencies:**
- Metadata database (PostgreSQL)
- Message queue (Redis)
- Function storage (S3/MinIO)
- Metrics collector

### 3. Data Plane - Scheduler

**Responsibilities:**
- Queue management and prioritization
- Worker load balancing
- Resource allocation
- Execution planning
- Dead letter queue handling

**Public Interfaces:**
```go
type Scheduler interface {
    ScheduleExecution(ctx context.Context, req ExecutionRequest) error
    GetWorkerStats(ctx context.Context) ([]WorkerStats, error)
    RebalanceLoad(ctx context.Context) error
}

type QueueManager interface {
    Enqueue(ctx context.Context, queue string, message Message) error
    Dequeue(ctx context.Context, queue string, timeout time.Duration) (*Message, error)
    DeadLetter(ctx context.Context, message Message, reason string) error
    GetQueueStats(ctx context.Context, queue string) (*QueueStats, error)
}
```

**Key Data Structures:**
```go
type ExecutionRequest struct {
    ID         string                 `json:"id"`
    FunctionID string                 `json:"function_id"`
    Payload    json.RawMessage        `json:"payload"`
    Headers    map[string]string      `json:"headers"`
    Priority   int                    `json:"priority"`
    Timeout    time.Duration          `json:"timeout"`
    RetryCount int                    `json:"retry_count"`
    CreatedAt  time.Time              `json:"created_at"`
}

type WorkerStats struct {
    WorkerID        string    `json:"worker_id"`
    ActiveTasks     int       `json:"active_tasks"`
    CompletedTasks  int64     `json:"completed_tasks"`
    FailedTasks     int64     `json:"failed_tasks"`
    CPUUsage        float64   `json:"cpu_usage"`
    MemoryUsage     int64     `json:"memory_usage"`
    LastHeartbeat   time.Time `json:"last_heartbeat"`
}
```

**Dependencies:**
- Message queue (Redis Cluster)
- Metrics collector
- Worker pool

### 4. Data Plane - Worker Pool

**Responsibilities:**
- Function execution
- Runtime management
- Resource monitoring
- Result reporting
- Health checking

**Public Interfaces:**
```go
type Worker interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Execute(ctx context.Context, req ExecutionRequest) (*ExecutionResult, error)
    GetStats(ctx context.Context) (*WorkerStats, error)
}

type RuntimeManager interface {
    GetRuntime(runtimeType RuntimeType) (Runtime, error)
    CreateContainer(ctx context.Context, spec ContainerSpec) (*Container, error)
    DestroyContainer(ctx context.Context, containerID string) error
    ListContainers(ctx context.Context) ([]*Container, error)
}
```

**Key Data Structures:**
```go
type ExecutionResult struct {
    ID          string           `json:"id"`
    Status      ExecutionStatus  `json:"status"`
    Result      json.RawMessage  `json:"result,omitempty"`
    Error       *ExecutionError  `json:"error,omitempty"`
    Metrics     ExecutionMetrics `json:"metrics"`
    StartedAt   time.Time        `json:"started_at"`
    CompletedAt time.Time        `json:"completed_at"`
}

type ExecutionMetrics struct {
    Duration     time.Duration `json:"duration"`
    CPUTime      time.Duration `json:"cpu_time"`
    MemoryPeak   int64         `json:"memory_peak"`
    NetworkIn    int64         `json:"network_in"`
    NetworkOut   int64         `json:"network_out"`
}

type ContainerSpec struct {
    Image       string            `json:"image"`
    Command     []string          `json:"command"`
    Environment map[string]string `json:"environment"`
    Limits      ResourceLimits    `json:"limits"`
    Timeout     time.Duration     `json:"timeout"`
}

type ResourceLimits struct {
    CPUShares    int64 `json:"cpu_shares"`
    MemoryBytes  int64 `json:"memory_bytes"`
    NetworkBps   int64 `json:"network_bps"`
    DiskIOBps    int64 `json:"disk_io_bps"`
}
```

**Dependencies:**
- Container runtime (Docker/Podman)
- Function storage
- Metrics collector
- Scheduler

### 5. Runtime Isolation Layer

**Responsibilities:**
- Sandboxed function execution
- Resource enforcement
- Security policy application
- Container lifecycle management

**Public Interfaces:**
```go
type Runtime interface {
    Execute(ctx context.Context, spec ExecutionSpec) (*ExecutionResult, error)
    Validate(code FunctionCode) error
    GetCapabilities() RuntimeCapabilities
}

type ContainerRuntime interface {
    CreateContainer(ctx context.Context, spec ContainerSpec) (*Container, error)
    StartContainer(ctx context.Context, containerID string) error
    StopContainer(ctx context.Context, containerID string, timeout time.Duration) error
    RemoveContainer(ctx context.Context, containerID string) error
    GetLogs(ctx context.Context, containerID string) (io.ReadCloser, error)
}
```

**Key Data Structures:**
```go
type ExecutionSpec struct {
    FunctionCode FunctionCode      `json:"function_code"`
    Handler      string            `json:"handler"`
    Payload      json.RawMessage   `json:"payload"`
    Environment  map[string]string `json:"environment"`
    Timeout      time.Duration     `json:"timeout"`
    Limits       ResourceLimits    `json:"limits"`
}

type RuntimeCapabilities struct {
    Language     string   `json:"language"`
    Version      string   `json:"version"`
    Extensions   []string `json:"extensions"`
    MaxTimeout   time.Duration `json:"max_timeout"`
    MaxMemory    int64    `json:"max_memory"`
}
```

**Dependencies:**
- Container runtime engine
- Security policies
- Resource monitoring

### 6. Storage Layer

**Responsibilities:**
- Function code storage and retrieval
- Metadata persistence
- Execution history
- Audit logging

**Public Interfaces:**
```go
type FunctionStorage interface {
    Store(ctx context.Context, functionID string, code FunctionCode) (*StorageLocation, error)
    Retrieve(ctx context.Context, location StorageLocation) (*FunctionCode, error)
    Delete(ctx context.Context, location StorageLocation) error
    List(ctx context.Context, prefix string) ([]*StorageLocation, error)
}

type MetadataStore interface {
    CreateFunction(ctx context.Context, fn *Function) error
    GetFunction(ctx context.Context, id string) (*Function, error)
    UpdateFunction(ctx context.Context, fn *Function) error
    DeleteFunction(ctx context.Context, id string) error
    ListFunctions(ctx context.Context, filter FunctionFilter) ([]*Function, error)
    
    CreateInvocation(ctx context.Context, inv *Invocation) error
    GetInvocation(ctx context.Context, id string) (*Invocation, error)
    UpdateInvocation(ctx context.Context, inv *Invocation) error
    ListInvocations(ctx context.Context, filter InvocationFilter) ([]*Invocation, error)
}
```

**Dependencies:**
- Object storage (S3/MinIO)
- Relational database (PostgreSQL)
- Caching layer (Redis)

### 7. Observability Layer

**Responsibilities:**
- Metrics collection and aggregation
- Distributed tracing
- Structured logging
- Alerting and monitoring

**Public Interfaces:**
```go
type MetricsCollector interface {
    Counter(name string, tags map[string]string) Counter
    Gauge(name string, tags map[string]string) Gauge
    Histogram(name string, tags map[string]string) Histogram
    Timer(name string, tags map[string]string) Timer
}

type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    WithFields(fields ...Field) Logger
}

type Tracer interface {
    StartSpan(ctx context.Context, operationName string) (Span, context.Context)
    InjectHeaders(span Span, headers map[string]string)
    ExtractHeaders(headers map[string]string) (SpanContext, error)
}
```

**Dependencies:**
- Metrics backend (Prometheus)
- Logging backend (ELK/Loki)
- Tracing backend (Jaeger/Zipkin)

## Security Considerations

### Authentication & Authorization
- JWT-based authentication with refresh tokens
- Role-based access control (RBAC)
- Function-level permissions
- API key management for service-to-service communication

### Runtime Security
- Container-based isolation
- Resource limits enforcement
- Network policies
- Read-only filesystem for function execution
- Secrets management integration

### Data Security
- Encryption at rest for function code
- Encryption in transit (TLS)
- Audit logging for all operations
- PII detection and handling

## Scalability Design

### Horizontal Scaling
- Stateless API servers behind load balancer
- Worker pool auto-scaling based on queue depth
- Database read replicas
- Redis cluster for high availability

### Performance Optimization
- Function code caching
- Container image pre-warming
- Connection pooling
- Async processing with result caching

### Resource Management
- CPU and memory limits per function
- Execution timeout enforcement
- Queue prioritization
- Dead letter queue for failed executions

## Deployment Architecture

### Container Orchestration
```yaml
# Kubernetes deployment structure
apiVersion: v1
kind: Namespace
metadata:
  name: faas-platform
---
# API Gateway Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-gateway
  namespace: faas-platform
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api-gateway
  template:
    metadata:
      labels:
        app: api-gateway
    spec:
      containers:
      - name: gateway
        image: faas/api-gateway:latest
        ports:
        - containerPort: 8080
        env:
        - name: REDIS_URL
          value: "redis://redis-cluster:6379"
        - name: DB_URL
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: url
---
# Worker Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: workers
  namespace: faas-platform
spec:
  replicas: 5
  selector:
    matchLabels:
      app: worker
  template:
    metadata:
      labels:
        app: worker
    spec:
      containers:
      - name: worker
        image: faas/worker:latest
        volumeMounts:
        - name: docker-sock
          mountPath: /var/run/docker.sock
        env:
        - name: REDIS_URL
          value: "redis://redis-cluster:6379"
      volumes:
      - name: docker-sock
        hostPath:
          path: /var/run/docker.sock
```

### Infrastructure Components
- **Load Balancer**: NGINX/HAProxy for API gateway
- **Container Registry**: Harbor/ECR for runtime images
- **Monitoring**: Prometheus + Grafana stack
- **Logging**: ELK stack or Loki
- **Service Mesh**: Istio for advanced networking (optional)

## Migration Strategy

### Phase 1: Core Infrastructure
1. Set up PostgreSQL database with schema
2. Deploy Redis cluster
3. Set up S3-compatible object storage
4. Implement basic authentication

### Phase 2: Control Plane
1. Implement function management APIs
2. Add invocation APIs with result handling
3. Set up basic observability

### Phase 3: Data Plane
1. Implement scheduler with queue management
2. Deploy worker pool with container runtime
3. Add multi-runtime support

### Phase 4: Production Hardening
1. Add comprehensive security measures
2. Implement auto-scaling
3. Add advanced monitoring and alerting
4. Performance optimization

This architecture provides a solid foundation for a production-ready FaaS platform while maintaining the simplicity and clarity of the original design.