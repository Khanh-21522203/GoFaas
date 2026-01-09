# FaaS Platform Implementation Guide

## PHASE 3 — Implementation Guidance

This guide provides step-by-step instructions for implementing the FaaS platform from scratch, building layer by layer starting from core abstractions.

## Implementation Strategy

### Build Order (Bottom-Up Approach)
1. **Foundation Layer**: Core types, interfaces, and utilities
2. **Storage Layer**: Database schemas and storage implementations
3. **Messaging Layer**: Queue implementations and message handling
4. **Core Business Logic**: Domain services and repositories
5. **Runtime Layer**: Function execution and container management
6. **API Layer**: HTTP handlers and middleware
7. **Orchestration Layer**: Scheduler and worker coordination
8. **Observability Layer**: Metrics, logging, and tracing
9. **Deployment Layer**: Containerization and orchestration

## Step 1: Foundation Layer

### Core Types and Interfaces

**Rationale**: Start with the fundamental data structures and interfaces that define the system's contracts. This provides a stable foundation for all other components.

```go
// pkg/types/function.go
package types

import (
    "encoding/json"
    "time"
)

// RuntimeType represents supported function runtimes
type RuntimeType string

const (
    RuntimeGo     RuntimeType = "go"
    RuntimePython RuntimeType = "python"
    RuntimeNodeJS RuntimeType = "nodejs"
)

// ExecutionStatus represents the status of function execution
type ExecutionStatus string

const (
    StatusPending   ExecutionStatus = "pending"
    StatusRunning   ExecutionStatus = "running"
    StatusCompleted ExecutionStatus = "completed"
    StatusFailed    ExecutionStatus = "failed"
    StatusTimeout   ExecutionStatus = "timeout"
)

// Function represents a serverless function
type Function struct {
    ID          string            `json:"id" db:"id"`
    Name        string            `json:"name" db:"name"`
    Version     string            `json:"version" db:"version"`
    Runtime     RuntimeType       `json:"runtime" db:"runtime"`
    Handler     string            `json:"handler" db:"handler"`
    Code        FunctionCode      `json:"code"`
    Config      FunctionConfig    `json:"config"`
    Metadata    map[string]string `json:"metadata" db:"metadata"`
    CreatedAt   time.Time         `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
}

// FunctionCode represents function source code
type FunctionCode struct {
    Source     string `json:"source"`      // Base64 encoded or storage URL
    SourceType string `json:"source_type"` // "inline", "s3", "git"
    Checksum   string `json:"checksum"`    // SHA256 hash
    Size       int64  `json:"size"`        // Size in bytes
}

// FunctionConfig represents function configuration
type FunctionConfig struct {
    Timeout     time.Duration     `json:"timeout" db:"timeout"`
    Memory      int               `json:"memory_mb" db:"memory_mb"`
    Environment map[string]string `json:"environment" db:"environment"`
    Concurrency int               `json:"max_concurrency" db:"max_concurrency"`
}

// Invocation represents a function invocation request
type Invocation struct {
    ID          string                 `json:"id" db:"id"`
    FunctionID  string                 `json:"function_id" db:"function_id"`
    Payload     json.RawMessage        `json:"payload" db:"payload"`
    Headers     map[string]string      `json:"headers" db:"headers"`
    Status      ExecutionStatus        `json:"status" db:"status"`
    Result      json.RawMessage        `json:"result,omitempty" db:"result"`
    Error       *ExecutionError        `json:"error,omitempty" db:"error"`
    Metrics     *ExecutionMetrics      `json:"metrics,omitempty" db:"metrics"`
    CreatedAt   time.Time              `json:"created_at" db:"created_at"`
    StartedAt   *time.Time             `json:"started_at,omitempty" db:"started_at"`
    CompletedAt *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
}

// ExecutionError represents an execution error
type ExecutionError struct {
    Type    string `json:"type"`
    Message string `json:"message"`
    Stack   string `json:"stack,omitempty"`
}

// ExecutionMetrics represents execution metrics
type ExecutionMetrics struct {
    Duration   time.Duration `json:"duration"`
    CPUTime    time.Duration `json:"cpu_time"`
    MemoryPeak int64         `json:"memory_peak"`
    NetworkIn  int64         `json:"network_in"`
    NetworkOut int64         `json:"network_out"`
}
```

**Why this approach**: 
- Defines clear contracts upfront
- Enables parallel development of different components
- Provides type safety across the entire system
- Makes testing easier with well-defined interfaces

**Alternatives considered**:
- **Code-first approach**: Start with implementation and extract interfaces later (rejected: leads to tight coupling)
- **Database-first approach**: Start with schema design (rejected: doesn't capture business logic well)

## Step 2: Storage Layer

### Database Schema Design

**Rationale**: Establish persistent storage foundation before building business logic. This ensures data consistency and provides a stable base for repositories.

```sql
-- migrations/001_initial_schema.up.sql

-- Functions table
CREATE TABLE functions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    runtime VARCHAR(50) NOT NULL,
    handler VARCHAR(255) NOT NULL,
    code_source TEXT NOT NULL,
    code_source_type VARCHAR(20) NOT NULL DEFAULT 'inline',
    code_checksum VARCHAR(64) NOT NULL,
    code_size BIGINT NOT NULL,
    timeout_seconds INTEGER NOT NULL DEFAULT 30,
    memory_mb INTEGER NOT NULL DEFAULT 128,
    max_concurrency INTEGER NOT NULL DEFAULT 10,
    environment JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(name, version)
);

-- Invocations table
CREATE TABLE invocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID NOT NULL REFERENCES functions(id) ON DELETE CASCADE,
    payload JSONB,
    headers JSONB DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    result JSONB,
    error JSONB,
    metrics JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    INDEX idx_invocations_function_id (function_id),
    INDEX idx_invocations_status (status),
    INDEX idx_invocations_created_at (created_at)
);

-- Users table (for authentication)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    roles TEXT[] DEFAULT ARRAY['user'],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Function permissions table
CREATE TABLE function_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID NOT NULL REFERENCES functions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission VARCHAR(20) NOT NULL, -- 'read', 'write', 'execute', 'admin'
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(function_id, user_id, permission)
);
```

### Repository Implementation

```go
// internal/storage/metadata/interface.go
package metadata

import (
    "context"
    "github.com/your-org/faas-platform/pkg/types"
)

// FunctionRepository defines function storage operations
type FunctionRepository interface {
    Create(ctx context.Context, fn *types.Function) error
    GetByID(ctx context.Context, id string) (*types.Function, error)
    GetByName(ctx context.Context, name, version string) (*types.Function, error)
    Update(ctx context.Context, fn *types.Function) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, filter FunctionFilter) ([]*types.Function, error)
}

// InvocationRepository defines invocation storage operations
type InvocationRepository interface {
    Create(ctx context.Context, inv *types.Invocation) error
    GetByID(ctx context.Context, id string) (*types.Invocation, error)
    Update(ctx context.Context, inv *types.Invocation) error
    List(ctx context.Context, filter InvocationFilter) ([]*types.Invocation, error)
}

// FunctionFilter represents function query filters
type FunctionFilter struct {
    Runtime    *types.RuntimeType
    CreatedBy  *string
    Limit      int
    Offset     int
}

// InvocationFilter represents invocation query filters
type InvocationFilter struct {
    FunctionID *string
    Status     *types.ExecutionStatus
    Limit      int
    Offset     int
}
```

```go
// internal/storage/metadata/postgres.go
package metadata

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    
    "github.com/lib/pq"
    _ "github.com/lib/pq"
    "github.com/your-org/faas-platform/pkg/types"
)

// PostgresRepository implements metadata repositories using PostgreSQL
type PostgresRepository struct {
    db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
    return &PostgresRepository{db: db}
}

// Create implements FunctionRepository.Create
func (r *PostgresRepository) CreateFunction(ctx context.Context, fn *types.Function) error {
    query := `
        INSERT INTO functions (
            id, name, version, runtime, handler, code_source, code_source_type,
            code_checksum, code_size, timeout_seconds, memory_mb, max_concurrency,
            environment, metadata
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`
    
    envJSON, _ := json.Marshal(fn.Config.Environment)
    metaJSON, _ := json.Marshal(fn.Metadata)
    
    _, err := r.db.ExecContext(ctx, query,
        fn.ID, fn.Name, fn.Version, fn.Runtime, fn.Handler,
        fn.Code.Source, fn.Code.SourceType, fn.Code.Checksum, fn.Code.Size,
        int(fn.Config.Timeout.Seconds()), fn.Config.Memory, fn.Config.Concurrency,
        envJSON, metaJSON,
    )
    
    if err != nil {
        if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
            return fmt.Errorf("function %s version %s already exists", fn.Name, fn.Version)
        }
        return fmt.Errorf("failed to create function: %w", err)
    }
    
    return nil
}

// GetByID implements FunctionRepository.GetByID
func (r *PostgresRepository) GetFunctionByID(ctx context.Context, id string) (*types.Function, error) {
    query := `
        SELECT id, name, version, runtime, handler, code_source, code_source_type,
               code_checksum, code_size, timeout_seconds, memory_mb, max_concurrency,
               environment, metadata, created_at, updated_at
        FROM functions WHERE id = $1`
    
    var fn types.Function
    var envJSON, metaJSON []byte
    var timeoutSeconds int
    
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &fn.ID, &fn.Name, &fn.Version, &fn.Runtime, &fn.Handler,
        &fn.Code.Source, &fn.Code.SourceType, &fn.Code.Checksum, &fn.Code.Size,
        &timeoutSeconds, &fn.Config.Memory, &fn.Config.Concurrency,
        &envJSON, &metaJSON, &fn.CreatedAt, &fn.UpdatedAt,
    )
    
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("function not found: %s", id)
        }
        return nil, fmt.Errorf("failed to get function: %w", err)
    }
    
    fn.Config.Timeout = time.Duration(timeoutSeconds) * time.Second
    json.Unmarshal(envJSON, &fn.Config.Environment)
    json.Unmarshal(metaJSON, &fn.Metadata)
    
    return &fn, nil
}
```

**Why this approach**:
- **PostgreSQL**: ACID compliance, JSON support, mature ecosystem
- **Repository pattern**: Abstracts data access, enables testing with mocks
- **Migration-based schema**: Version-controlled database changes

**Alternatives considered**:
- **NoSQL (MongoDB)**: Better for document storage but lacks ACID guarantees
- **Active Record pattern**: Simpler but couples business logic to database

## Step 3: Messaging Layer

### Queue Implementation

**Rationale**: Implement reliable async communication before building orchestration logic. This enables decoupling between API and execution layers.

```go
// internal/messaging/interface.go
package messaging

import (
    "context"
    "time"
)

// Message represents a queue message
type Message struct {
    ID       string            `json:"id"`
    Queue    string            `json:"queue"`
    Payload  []byte            `json:"payload"`
    Headers  map[string]string `json:"headers"`
    Attempts int               `json:"attempts"`
    EnqueuedAt time.Time       `json:"enqueued_at"`
}

// Queue defines message queue operations
type Queue interface {
    Enqueue(ctx context.Context, queue string, payload []byte, headers map[string]string) error
    Dequeue(ctx context.Context, queue string, timeout time.Duration) (*Message, error)
    Ack(ctx context.Context, message *Message) error
    Nack(ctx context.Context, message *Message) error
    DeadLetter(ctx context.Context, message *Message, reason string) error
    GetStats(ctx context.Context, queue string) (*QueueStats, error)
}

// QueueStats represents queue statistics
type QueueStats struct {
    Name         string `json:"name"`
    Size         int64  `json:"size"`
    Consumers    int    `json:"consumers"`
    EnqueueRate  float64 `json:"enqueue_rate"`
    DequeueRate  float64 `json:"dequeue_rate"`
}
```

```go
// internal/messaging/redis.go
package messaging

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/go-redis/redis/v8"
    "github.com/google/uuid"
)

// RedisQueue implements Queue using Redis
type RedisQueue struct {
    client *redis.Client
    prefix string
}

// NewRedisQueue creates a new Redis queue
func NewRedisQueue(client *redis.Client, prefix string) *RedisQueue {
    return &RedisQueue{
        client: client,
        prefix: prefix,
    }
}

// Enqueue adds a message to the queue
func (q *RedisQueue) Enqueue(ctx context.Context, queue string, payload []byte, headers map[string]string) error {
    message := Message{
        ID:         uuid.New().String(),
        Queue:      queue,
        Payload:    payload,
        Headers:    headers,
        Attempts:   0,
        EnqueuedAt: time.Now(),
    }
    
    data, err := json.Marshal(message)
    if err != nil {
        return fmt.Errorf("failed to marshal message: %w", err)
    }
    
    queueKey := q.queueKey(queue)
    return q.client.LPush(ctx, queueKey, data).Err()
}

// Dequeue removes and returns a message from the queue
func (q *RedisQueue) Dequeue(ctx context.Context, queue string, timeout time.Duration) (*Message, error) {
    queueKey := q.queueKey(queue)
    processingKey := q.processingKey(queue)
    
    // Use BRPOPLPUSH for reliable message processing
    result, err := q.client.BRPopLPush(ctx, queueKey, processingKey, timeout).Result()
    if err != nil {
        if err == redis.Nil {
            return nil, nil // No message available
        }
        return nil, fmt.Errorf("failed to dequeue message: %w", err)
    }
    
    var message Message
    if err := json.Unmarshal([]byte(result), &message); err != nil {
        return nil, fmt.Errorf("failed to unmarshal message: %w", err)
    }
    
    message.Attempts++
    return &message, nil
}

// Ack acknowledges successful message processing
func (q *RedisQueue) Ack(ctx context.Context, message *Message) error {
    processingKey := q.processingKey(message.Queue)
    data, _ := json.Marshal(message)
    
    return q.client.LRem(ctx, processingKey, 1, string(data)).Err()
}

// Nack rejects a message and requeues it
func (q *RedisQueue) Nack(ctx context.Context, message *Message) error {
    processingKey := q.processingKey(message.Queue)
    queueKey := q.queueKey(message.Queue)
    
    data, _ := json.Marshal(message)
    
    // Remove from processing queue and add back to main queue
    pipe := q.client.Pipeline()
    pipe.LRem(ctx, processingKey, 1, string(data))
    pipe.LPush(ctx, queueKey, data)
    
    _, err := pipe.Exec(ctx)
    return err
}

// DeadLetter moves a message to the dead letter queue
func (q *RedisQueue) DeadLetter(ctx context.Context, message *Message, reason string) error {
    processingKey := q.processingKey(message.Queue)
    deadLetterKey := q.deadLetterKey(message.Queue)
    
    // Add reason to message headers
    if message.Headers == nil {
        message.Headers = make(map[string]string)
    }
    message.Headers["dead_letter_reason"] = reason
    message.Headers["dead_lettered_at"] = time.Now().Format(time.RFC3339)
    
    data, _ := json.Marshal(message)
    
    pipe := q.client.Pipeline()
    pipe.LRem(ctx, processingKey, 1, string(data))
    pipe.LPush(ctx, deadLetterKey, data)
    
    _, err := pipe.Exec(ctx)
    return err
}

func (q *RedisQueue) queueKey(queue string) string {
    return fmt.Sprintf("%s:queue:%s", q.prefix, queue)
}

func (q *RedisQueue) processingKey(queue string) string {
    return fmt.Sprintf("%s:processing:%s", q.prefix, queue)
}

func (q *RedisQueue) deadLetterKey(queue string) string {
    return fmt.Sprintf("%s:dead_letter:%s", q.prefix, queue)
}
```

**Why this approach**:
- **Redis BRPOPLPUSH**: Provides reliable message processing with automatic retry
- **Dead letter queue**: Handles permanently failed messages
- **Message acknowledgment**: Ensures at-least-once delivery semantics

**Alternatives considered**:
- **RabbitMQ**: More features but adds operational complexity
- **Apache Kafka**: Better for high-throughput but overkill for this use case
- **Simple Redis BLPOP**: Simpler but no reliability guarantees

## Step 4: Core Business Logic

### Function Service Implementation

**Rationale**: Implement core business logic after establishing storage and messaging foundations. This layer orchestrates operations and enforces business rules.

```go
// internal/core/function/service.go
package function

import (
    "context"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "time"
    
    "github.com/google/uuid"
    "github.com/your-org/faas-platform/internal/storage/metadata"
    "github.com/your-org/faas-platform/pkg/types"
)

// Service implements function management business logic
type Service struct {
    repo    metadata.FunctionRepository
    storage FunctionStorage
    logger  Logger
}

// FunctionStorage defines function code storage operations
type FunctionStorage interface {
    Store(ctx context.Context, functionID string, code []byte) (string, error)
    Retrieve(ctx context.Context, location string) ([]byte, error)
    Delete(ctx context.Context, location string) error
}

// Logger defines logging interface
type Logger interface {
    Info(msg string, fields ...interface{})
    Error(msg string, fields ...interface{})
    Debug(msg string, fields ...interface{})
}

// NewService creates a new function service
func NewService(repo metadata.FunctionRepository, storage FunctionStorage, logger Logger) *Service {
    return &Service{
        repo:    repo,
        storage: storage,
        logger:  logger,
    }
}

// CreateFunction creates a new function
func (s *Service) CreateFunction(ctx context.Context, req CreateFunctionRequest) (*types.Function, error) {
    // Validate request
    if err := s.validateCreateRequest(req); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    
    // Decode function code
    codeBytes, err := base64.StdEncoding.DecodeString(req.Code)
    if err != nil {
        return nil, fmt.Errorf("invalid base64 code: %w", err)
    }
    
    // Generate function ID
    functionID := uuid.New().String()
    
    // Calculate checksum
    checksum := fmt.Sprintf("%x", sha256.Sum256(codeBytes))
    
    // Store function code
    codeLocation, err := s.storage.Store(ctx, functionID, codeBytes)
    if err != nil {
        return nil, fmt.Errorf("failed to store function code: %w", err)
    }
    
    // Create function entity
    function := &types.Function{
        ID:      functionID,
        Name:    req.Name,
        Version: req.Version,
        Runtime: req.Runtime,
        Handler: req.Handler,
        Code: types.FunctionCode{
            Source:     codeLocation,
            SourceType: "s3",
            Checksum:   checksum,
            Size:       int64(len(codeBytes)),
        },
        Config: types.FunctionConfig{
            Timeout:     req.Timeout,
            Memory:      req.Memory,
            Environment: req.Environment,
            Concurrency: req.Concurrency,
        },
        Metadata:  req.Metadata,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    
    // Save to database
    if err := s.repo.Create(ctx, function); err != nil {
        // Cleanup stored code on failure
        s.storage.Delete(ctx, codeLocation)
        return nil, fmt.Errorf("failed to save function: %w", err)
    }
    
    s.logger.Info("Function created successfully",
        "function_id", functionID,
        "name", req.Name,
        "version", req.Version,
        "runtime", req.Runtime,
    )
    
    return function, nil
}

// GetFunction retrieves a function by ID
func (s *Service) GetFunction(ctx context.Context, id string) (*types.Function, error) {
    function, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("failed to get function: %w", err)
    }
    
    return function, nil
}

// UpdateFunction updates an existing function
func (s *Service) UpdateFunction(ctx context.Context, id string, req UpdateFunctionRequest) (*types.Function, error) {
    // Get existing function
    function, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("function not found: %w", err)
    }
    
    // Update fields
    if req.Handler != nil {
        function.Handler = *req.Handler
    }
    if req.Timeout != nil {
        function.Config.Timeout = *req.Timeout
    }
    if req.Memory != nil {
        function.Config.Memory = *req.Memory
    }
    if req.Environment != nil {
        function.Config.Environment = req.Environment
    }
    if req.Concurrency != nil {
        function.Config.Concurrency = *req.Concurrency
    }
    
    // Update code if provided
    if req.Code != nil {
        codeBytes, err := base64.StdEncoding.DecodeString(*req.Code)
        if err != nil {
            return nil, fmt.Errorf("invalid base64 code: %w", err)
        }
        
        // Store new code
        codeLocation, err := s.storage.Store(ctx, id, codeBytes)
        if err != nil {
            return nil, fmt.Errorf("failed to store function code: %w", err)
        }
        
        // Update code metadata
        function.Code.Source = codeLocation
        function.Code.Checksum = fmt.Sprintf("%x", sha256.Sum256(codeBytes))
        function.Code.Size = int64(len(codeBytes))
    }
    
    function.UpdatedAt = time.Now()
    
    // Save changes
    if err := s.repo.Update(ctx, function); err != nil {
        return nil, fmt.Errorf("failed to update function: %w", err)
    }
    
    s.logger.Info("Function updated successfully",
        "function_id", id,
        "name", function.Name,
    )
    
    return function, nil
}

// DeleteFunction deletes a function
func (s *Service) DeleteFunction(ctx context.Context, id string) error {
    // Get function to retrieve code location
    function, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return fmt.Errorf("function not found: %w", err)
    }
    
    // Delete from database first
    if err := s.repo.Delete(ctx, id); err != nil {
        return fmt.Errorf("failed to delete function: %w", err)
    }
    
    // Delete stored code (best effort)
    if err := s.storage.Delete(ctx, function.Code.Source); err != nil {
        s.logger.Error("Failed to delete function code",
            "function_id", id,
            "code_location", function.Code.Source,
            "error", err,
        )
    }
    
    s.logger.Info("Function deleted successfully",
        "function_id", id,
        "name", function.Name,
    )
    
    return nil
}

// validateCreateRequest validates function creation request
func (s *Service) validateCreateRequest(req CreateFunctionRequest) error {
    if req.Name == "" {
        return fmt.Errorf("function name is required")
    }
    if req.Runtime == "" {
        return fmt.Errorf("runtime is required")
    }
    if req.Handler == "" {
        return fmt.Errorf("handler is required")
    }
    if req.Code == "" {
        return fmt.Errorf("function code is required")
    }
    if req.Timeout <= 0 {
        return fmt.Errorf("timeout must be positive")
    }
    if req.Memory <= 0 {
        return fmt.Errorf("memory must be positive")
    }
    
    // Validate runtime
    switch req.Runtime {
    case types.RuntimeGo, types.RuntimePython, types.RuntimeNodeJS:
        // Valid runtime
    default:
        return fmt.Errorf("unsupported runtime: %s", req.Runtime)
    }
    
    return nil
}
```

**Why this approach**:
- **Service layer pattern**: Encapsulates business logic and orchestration
- **Dependency injection**: Makes testing easier and reduces coupling
- **Validation**: Ensures data integrity at the business logic level
- **Error handling**: Provides meaningful error messages with context

**Alternatives considered**:
- **Anemic domain model**: Simpler but pushes business logic to controllers
- **Rich domain model**: More OOP but can become complex for simple CRUD operations

## Step 5: Runtime Layer

### Container-Based Function Execution

**Rationale**: Implement secure, isolated function execution using containers. This provides security, resource control, and multi-runtime support.

```go
// internal/worker/runtime/interface.go
package runtime

import (
    "context"
    "io"
    "time"
    
    "github.com/your-org/faas-platform/pkg/types"
)

// Runtime defines function execution interface
type Runtime interface {
    Execute(ctx context.Context, spec ExecutionSpec) (*ExecutionResult, error)
    Validate(code types.FunctionCode) error
    GetCapabilities() RuntimeCapabilities
}

// ExecutionSpec defines function execution parameters
type ExecutionSpec struct {
    FunctionID  string                 `json:"function_id"`
    Code        types.FunctionCode     `json:"code"`
    Handler     string                 `json:"handler"`
    Payload     []byte                 `json:"payload"`
    Environment map[string]string      `json:"environment"`
    Timeout     time.Duration          `json:"timeout"`
    Limits      ResourceLimits         `json:"limits"`
}

// ExecutionResult represents function execution result
type ExecutionResult struct {
    Status    types.ExecutionStatus `json:"status"`
    Result    []byte                `json:"result,omitempty"`
    Error     *types.ExecutionError `json:"error,omitempty"`
    Metrics   types.ExecutionMetrics `json:"metrics"`
    Logs      []LogEntry            `json:"logs,omitempty"`
}

// ResourceLimits defines resource constraints
type ResourceLimits struct {
    CPUShares   int64         `json:"cpu_shares"`
    MemoryBytes int64         `json:"memory_bytes"`
    NetworkBps  int64         `json:"network_bps"`
    DiskIOBps   int64         `json:"disk_io_bps"`
    Timeout     time.Duration `json:"timeout"`
}

// RuntimeCapabilities describes runtime capabilities
type RuntimeCapabilities struct {
    Language    string        `json:"language"`
    Version     string        `json:"version"`
    Extensions  []string      `json:"extensions"`
    MaxTimeout  time.Duration `json:"max_timeout"`
    MaxMemory   int64         `json:"max_memory"`
}

// LogEntry represents a log entry
type LogEntry struct {
    Timestamp time.Time `json:"timestamp"`
    Level     string    `json:"level"`
    Message   string    `json:"message"`
}
```

```go
// internal/worker/runtime/container.go
package runtime

import (
    "context"
    "fmt"
    "io"
    "strings"
    "time"
    
    "github.com/docker/docker/api/types"
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/network"
    "github.com/docker/docker/client"
    "github.com/docker/go-connections/nat"
)

// ContainerRuntime implements Runtime using Docker containers
type ContainerRuntime struct {
    client       *client.Client
    imagePrefix  string
    networkName  string
    logger       Logger
}

// NewContainerRuntime creates a new container runtime
func NewContainerRuntime(dockerClient *client.Client, imagePrefix, networkName string, logger Logger) *ContainerRuntime {
    return &ContainerRuntime{
        client:      dockerClient,
        imagePrefix: imagePrefix,
        networkName: networkName,
        logger:      logger,
    }
}

// Execute runs a function in a container
func (r *ContainerRuntime) Execute(ctx context.Context, spec ExecutionSpec) (*ExecutionResult, error) {
    startTime := time.Now()
    
    // Create execution context with timeout
    execCtx, cancel := context.WithTimeout(ctx, spec.Timeout)
    defer cancel()
    
    // Determine runtime image
    image, err := r.getRuntimeImage(spec.Code.Runtime)
    if err != nil {
        return nil, fmt.Errorf("unsupported runtime: %w", err)
    }
    
    // Create container
    containerID, err := r.createContainer(execCtx, image, spec)
    if err != nil {
        return nil, fmt.Errorf("failed to create container: %w", err)
    }
    defer r.cleanupContainer(context.Background(), containerID)
    
    // Start container
    if err := r.client.ContainerStart(execCtx, containerID, types.ContainerStartOptions{}); err != nil {
        return nil, fmt.Errorf("failed to start container: %w", err)
    }
    
    // Wait for completion
    statusCh, errCh := r.client.ContainerWait(execCtx, containerID, container.WaitConditionNotRunning)
    
    var exitCode int64
    select {
    case err := <-errCh:
        if err != nil {
            return &ExecutionResult{
                Status: types.ExecutionStatusFailed,
                Error: &types.ExecutionError{
                    Type:    "ContainerError",
                    Message: err.Error(),
                },
                Metrics: r.calculateMetrics(startTime, time.Now(), 0),
            }, nil
        }
    case status := <-statusCh:
        exitCode = status.StatusCode
    case <-execCtx.Done():
        // Timeout occurred
        r.client.ContainerKill(context.Background(), containerID, "SIGKILL")
        return &ExecutionResult{
            Status: types.ExecutionStatusTimeout,
            Error: &types.ExecutionError{
                Type:    "TimeoutError",
                Message: "Function execution timed out",
            },
            Metrics: r.calculateMetrics(startTime, time.Now(), 0),
        }, nil
    }
    
    endTime := time.Now()
    
    // Get container logs
    logs, err := r.getContainerLogs(ctx, containerID)
    if err != nil {
        r.logger.Error("Failed to get container logs", "container_id", containerID, "error", err)
    }
    
    // Get container stats
    stats, err := r.getContainerStats(ctx, containerID)
    if err != nil {
        r.logger.Error("Failed to get container stats", "container_id", containerID, "error", err)
    }
    
    // Parse execution result
    result := &ExecutionResult{
        Metrics: r.calculateMetrics(startTime, endTime, stats),
        Logs:    logs,
    }
    
    if exitCode == 0 {
        result.Status = types.ExecutionStatusCompleted
        // Extract function result from logs (last line typically contains result)
        if len(logs) > 0 {
            result.Result = []byte(logs[len(logs)-1].Message)
        }
    } else {
        result.Status = types.ExecutionStatusFailed
        result.Error = &types.ExecutionError{
            Type:    "RuntimeError",
            Message: fmt.Sprintf("Function exited with code %d", exitCode),
        }
    }
    
    return result, nil
}

// createContainer creates a new container for function execution
func (r *ContainerRuntime) createContainer(ctx context.Context, image string, spec ExecutionSpec) (string, error) {
    // Prepare environment variables
    env := make([]string, 0, len(spec.Environment)+2)
    env = append(env, fmt.Sprintf("FUNCTION_HANDLER=%s", spec.Handler))
    env = append(env, fmt.Sprintf("FUNCTION_PAYLOAD=%s", string(spec.Payload)))
    
    for key, value := range spec.Environment {
        env = append(env, fmt.Sprintf("%s=%s", key, value))
    }
    
    // Container configuration
    config := &container.Config{
        Image:        image,
        Env:          env,
        WorkingDir:   "/app",
        AttachStdout: true,
        AttachStderr: true,
    }
    
    // Host configuration with resource limits
    hostConfig := &container.HostConfig{
        Resources: container.Resources{
            Memory:    spec.Limits.MemoryBytes,
            CPUShares: spec.Limits.CPUShares,
        },
        NetworkMode: container.NetworkMode(r.networkName),
        ReadonlyRootfs: true, // Security: read-only filesystem
        AutoRemove:     false, // We'll remove manually after getting logs
    }
    
    // Network configuration
    networkConfig := &network.NetworkingConfig{}
    
    // Create container
    resp, err := r.client.ContainerCreate(ctx, config, hostConfig, networkConfig, nil, "")
    if err != nil {
        return "", err
    }
    
    return resp.ID, nil
}

// getRuntimeImage returns the Docker image for the specified runtime
func (r *ContainerRuntime) getRuntimeImage(runtime types.RuntimeType) (string, error) {
    switch runtime {
    case types.RuntimeGo:
        return fmt.Sprintf("%s/go-runtime:latest", r.imagePrefix), nil
    case types.RuntimePython:
        return fmt.Sprintf("%s/python-runtime:latest", r.imagePrefix), nil
    case types.RuntimeNodeJS:
        return fmt.Sprintf("%s/nodejs-runtime:latest", r.imagePrefix), nil
    default:
        return "", fmt.Errorf("unsupported runtime: %s", runtime)
    }
}

// getContainerLogs retrieves container logs
func (r *ContainerRuntime) getContainerLogs(ctx context.Context, containerID string) ([]LogEntry, error) {
    options := types.ContainerLogsOptions{
        ShowStdout: true,
        ShowStderr: true,
        Timestamps: true,
    }
    
    reader, err := r.client.ContainerLogs(ctx, containerID, options)
    if err != nil {
        return nil, err
    }
    defer reader.Close()
    
    // Parse logs (simplified - in production, use proper log parsing)
    logs := make([]LogEntry, 0)
    // Implementation would parse Docker log format and extract log entries
    
    return logs, nil
}

// calculateMetrics calculates execution metrics
func (r *ContainerRuntime) calculateMetrics(startTime, endTime time.Time, stats *ContainerStats) types.ExecutionMetrics {
    duration := endTime.Sub(startTime)
    
    metrics := types.ExecutionMetrics{
        Duration: duration,
    }
    
    if stats != nil {
        metrics.CPUTime = stats.CPUTime
        metrics.MemoryPeak = stats.MemoryPeak
        metrics.NetworkIn = stats.NetworkIn
        metrics.NetworkOut = stats.NetworkOut
    }
    
    return metrics
}

// cleanupContainer removes the container
func (r *ContainerRuntime) cleanupContainer(ctx context.Context, containerID string) {
    if err := r.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true}); err != nil {
        r.logger.Error("Failed to remove container", "container_id", containerID, "error", err)
    }
}
```

**Why this approach**:
- **Container isolation**: Provides security and resource control
- **Multi-runtime support**: Easy to add new runtimes by creating new images
- **Resource limits**: Prevents resource exhaustion
- **Timeout handling**: Prevents runaway functions

**Alternatives considered**:
- **Process isolation**: Simpler but less secure
- **VM-based isolation**: More secure but higher overhead
- **WebAssembly**: Faster startup but limited language support

## Next Steps

This implementation guide provides the foundation for building a production-ready FaaS platform. The remaining steps would be:

1. **API Layer**: Implement HTTP handlers and middleware
2. **Orchestration**: Build scheduler and worker coordination
3. **Observability**: Add comprehensive logging, metrics, and tracing
4. **Security**: Implement authentication, authorization, and security policies
5. **Deployment**: Create Docker images and Kubernetes manifests
6. **Testing**: Add comprehensive unit, integration, and end-to-end tests

Each step builds upon the previous ones, creating a robust and scalable serverless platform.