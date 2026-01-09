# FaaS Platform Architecture Diagrams

## Complete System Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              Client Layer                                │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐       │
│  │   Web UI   │  │  CLI Tool  │  │   SDK      │  │  curl/API  │       │
│  └──────┬─────┘  └──────┬─────┘  └──────┬─────┘  └──────┬─────┘       │
└─────────┼────────────────┼────────────────┼────────────────┼────────────┘
          │                │                │                │
          └────────────────┴────────────────┴────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         API Gateway (Port 8080)                          │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │  Middleware: Logging │ Validation │ Error Handling              │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │                    HTTP Handlers                                 │  │
│  │  ┌────────────────┐  ┌────────────────┐  ┌──────────────────┐  │  │
│  │  │   Functions    │  │  Invocations   │  │     Health       │  │  │
│  │  │   - Create     │  │  - Invoke      │  │   - Check        │  │  │
│  │  │   - Read       │  │  - GetResult   │  │                  │  │  │
│  │  │   - Update     │  │  - List        │  │                  │  │  │
│  │  │   - Delete     │  │                │  │                  │  │  │
│  │  │   - List       │  │                │  │                  │  │  │
│  │  └────────────────┘  └────────────────┘  └──────────────────┘  │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└────────────────┬────────────────────────────────┬────────────────────────┘
                 │                                │
                 ▼                                ▼
┌─────────────────────────────┐    ┌─────────────────────────────────────┐
│     Function Service        │    │    Invocation Service               │
│  ┌──────────────────────┐   │    │  ┌──────────────────────────────┐  │
│  │ - Validation         │   │    │  │ - Queue Management           │  │
│  │ - Code Storage       │   │    │  │ - Status Tracking            │  │
│  │ - Metadata Mgmt      │   │    │  │ - Result Retrieval           │  │
│  └──────────────────────┘   │    │  └──────────────────────────────┘  │
└────────────┬────────────────┘    └────────────┬────────────────────────┘
             │                                   │
             ▼                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          Storage Layer                                   │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────────┐  │
│  │   PostgreSQL     │  │   Redis Queue    │  │  Function Storage    │  │
│  │                  │  │                  │  │   (Filesystem/S3)    │  │
│  │ - Functions      │  │ - Exec Queue     │  │                      │  │
│  │ - Invocations    │  │ - Processing     │  │ - Function Code      │  │
│  │ - Users          │  │ - Dead Letter    │  │ - Dependencies       │  │
│  │ - Permissions    │  │                  │  │                      │  │
│  └──────────────────┘  └──────────────────┘  └──────────────────────┘  │
└─────────────────────────────────┬───────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          Worker Pool                                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                  │
│  │   Worker 1   │  │   Worker 2   │  │   Worker N   │   ...            │
│  │              │  │              │  │              │                  │
│  │ ┌──────────┐ │  │ ┌──────────┐ │  │ ┌──────────┐ │                  │
│  │ │ Dequeue  │ │  │ │ Dequeue  │ │  │ │ Dequeue  │ │                  │
│  │ │ Execute  │ │  │ │ Execute  │ │  │ │ Execute  │ │                  │
│  │ │ Report   │ │  │ │ Report   │ │  │ │ Report   │ │                  │
│  │ └──────────┘ │  │ └──────────┘ │  │ └──────────┘ │                  │
│  └──────────────┘  └──────────────┘  └──────────────┘                  │
└────────────────────────────┬────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      Runtime Execution Layer                             │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │                    Runtime Interface                           │    │
│  │         Execute(ctx, spec) -> (result, error)                  │    │
│  └────────────────────┬───────────────────────────────────────────┘    │
│                       │                                                 │
│         ┌─────────────┴─────────────┐                                  │
│         │                           │                                  │
│         ▼                           ▼                                  │
│  ┌──────────────┐          ┌──────────────────────┐                   │
│  │   Simple     │          │   Container Runtime  │                   │
│  │   Runtime    │          │   (Docker-based)     │                   │
│  │              │          │                      │                   │
│  │ - Direct     │          │ - Image Selection    │                   │
│  │   Process    │          │ - Container Create   │                   │
│  │ - Fast       │          │ - Volume Mount       │                   │
│  │ - Dev Only   │          │ - Resource Limits    │                   │
│  └──────────────┘          │ - Execution          │                   │
│                            │ - Cleanup            │                   │
│                            └──────────┬───────────┘                   │
└───────────────────────────────────────┼───────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Docker Engine                                    │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │                    Container Execution                           │  │
│  │                                                                  │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐                │  │
│  │  │ Go Runtime │  │ Py Runtime │  │ JS Runtime │                │  │
│  │  │ Container  │  │ Container  │  │ Container  │                │  │
│  │  │            │  │            │  │            │                │  │
│  │  │ ┌────────┐ │  │ ┌────────┐ │  │ ┌────────┐ │                │  │
│  │  │ │Function│ │  │ │Function│ │  │ │Function│ │                │  │
│  │  │ │  Code  │ │  │ │  Code  │ │  │ │  Code  │ │                │  │
│  │  │ └────────┘ │  │ └────────┘ │  │ └────────┘ │                │  │
│  │  │            │  │            │  │            │                │  │
│  │  │ Memory:    │  │ Memory:    │  │ Memory:    │                │  │
│  │  │ 128MB      │  │ 256MB      │  │ 128MB      │                │  │
│  │  │ CPU: 1.0   │  │ CPU: 2.0   │  │ CPU: 1.0   │                │  │
│  │  └────────────┘  └────────────┘  └────────────┘                │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

## Function Execution Flow (Container-Based)

```
┌─────────────────────────────────────────────────────────────────────┐
│ 1. Client Invokes Function                                          │
│    POST /invoke {"function_id": "abc", "payload": {...}}            │
└────────────────────────┬────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 2. Controller Creates Invocation Record                             │
│    - Generate invocation ID                                         │
│    - Store in PostgreSQL (status: pending)                          │
│    - Return invocation handle to client                             │
└────────────────────────┬────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 3. Controller Enqueues Execution Request                            │
│    Redis LPUSH faas:queue:faas_executions                           │
│    {                                                                │
│      "invocation_id": "inv-123",                                    │
│      "function_id": "func-abc",                                     │
│      "payload": {...},                                              │
│      "timeout": "30s"                                               │
│    }                                                                │
└────────────────────────┬────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 4. Worker Dequeues Message                                          │
│    Redis BRPOPLPUSH (reliable dequeue)                              │
│    - Message moved to processing queue                              │
│    - Worker begins processing                                       │
└────────────────────────┬────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 5. Worker Updates Status                                            │
│    UPDATE invocations SET status='running', started_at=NOW()        │
└────────────────────────┬────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 6. Worker Retrieves Function Metadata                               │
│    SELECT * FROM functions WHERE id='func-abc'                      │
│    - Get runtime, handler, timeout, memory limits                   │
└────────────────────────┬────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 7. Worker Retrieves Function Code                                   │
│    Read from storage: ./storage/functions/func-abc/code             │
└────────────────────────┬────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 8. Container Runtime Prepares Execution                             │
│    - Create temp directory: /tmp/work/func-abc/1234567890           │
│    - Write function code to: main.go / main.py / main.js            │
│    - Select runtime image: faas-runtime-go:latest                   │
└────────────────────────┬────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 9. Docker Creates Container                                         │
│    docker create \                                                  │
│      --memory=128m \                                                │
│      --cpus=1.0 \                                                   │
│      --mount type=bind,src=/tmp/work/func-abc/...,dst=/app/function│
│      -e FUNCTION_PAYLOAD='{"key":"value"}' \                        │
│      -e FUNCTION_HANDLER='main' \                                   │
│      faas-runtime-go:latest                                         │
│                                                                     │
│    Container ID: abc123def456                                       │
└────────────────────────┬────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 10. Docker Starts Container                                         │
│     docker start abc123def456                                       │
│                                                                     │
│     Inside Container:                                               │
│     ┌─────────────────────────────────────────────────────────┐   │
│     │ /app/wrapper.sh                                          │   │
│     │   ├─ cd /app/function                                    │   │
│     │   ├─ go run main.go                                      │   │
│     │   └─ Function executes, writes to stdout                 │   │
│     └─────────────────────────────────────────────────────────┘   │
└────────────────────────┬────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 11. Worker Waits for Completion                                     │
│     docker wait abc123def456                                        │
│     - With timeout (context cancellation)                           │
│     - Monitors for timeout/errors                                   │
└────────────────────────┬────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 12. Worker Collects Results                                         │
│     - Exit code: 0 (success) or non-zero (failure)                  │
│     - Stdout/stderr: docker logs abc123def456                       │
│     - Metrics: duration, memory usage, etc.                         │
└────────────────────────┬────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 13. Worker Updates Invocation                                       │
│     UPDATE invocations SET                                          │
│       status='completed',                                           │
│       result='{"message":"Hello!"}',                                │
│       metrics='{"duration":1500000000}',                            │
│       completed_at=NOW()                                            │
│     WHERE id='inv-123'                                              │
└────────────────────────┬────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 14. Worker Cleans Up                                                │
│     - docker rm -f abc123def456                                     │
│     - rm -rf /tmp/work/func-abc/1234567890                          │
│     - Redis LREM (acknowledge message)                              │
└────────────────────────┬────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 15. Client Retrieves Result                                         │
│     GET /invocations/inv-123                                        │
│     {                                                               │
│       "status": "completed",                                        │
│       "result": {"message": "Hello!"},                              │
│       "metrics": {"duration": 1500000000}                           │
│     }                                                               │
└─────────────────────────────────────────────────────────────────────┘
```

## Container Isolation Model

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Host Operating System                        │
│                                                                     │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │                      Docker Engine                            │ │
│  │                                                               │ │
│  │  ┌─────────────────────────────────────────────────────────┐ │ │
│  │  │              Container Namespace                        │ │ │
│  │  │                                                         │ │ │
│  │  │  ┌───────────────────────────────────────────────────┐ │ │ │
│  │  │  │           Process Namespace (PID)                 │ │ │ │
│  │  │  │  - Isolated process tree                          │ │ │ │
│  │  │  │  - Cannot see host processes                      │ │ │ │
│  │  │  └───────────────────────────────────────────────────┘ │ │ │
│  │  │                                                         │ │ │
│  │  │  ┌───────────────────────────────────────────────────┐ │ │ │
│  │  │  │           Network Namespace (NET)                 │ │ │ │
│  │  │  │  - Isolated network stack                         │ │ │ │
│  │  │  │  - Bridge network (default)                       │ │ │ │
│  │  │  │  - No direct host access                          │ │ │ │
│  │  │  └───────────────────────────────────────────────────┘ │ │ │
│  │  │                                                         │ │ │
│  │  │  ┌───────────────────────────────────────────────────┐ │ │ │
│  │  │  │           Mount Namespace (MNT)                   │ │ │ │
│  │  │  │  - Isolated filesystem                            │ │ │ │
│  │  │  │  - Read-only function code mount                  │ │ │ │
│  │  │  │  - No access to host filesystem                   │ │ │ │
│  │  │  └───────────────────────────────────────────────────┘ │ │ │
│  │  │                                                         │ │ │
│  │  │  ┌───────────────────────────────────────────────────┐ │ │ │
│  │  │  │           Resource Limits (cgroups)               │ │ │ │
│  │  │  │  - Memory: 128MB (hard limit)                     │ │ │ │
│  │  │  │  - CPU: 1.0 cores (proportional)                  │ │ │ │
│  │  │  │  - Network: Unlimited (configurable)              │ │ │ │
│  │  │  └───────────────────────────────────────────────────┘ │ │ │
│  │  │                                                         │ │ │
│  │  │  ┌───────────────────────────────────────────────────┐ │ │ │
│  │  │  │           Function Execution                      │ │ │ │
│  │  │  │                                                   │ │ │ │
│  │  │  │  /app/wrapper.sh                                  │ │ │ │
│  │  │  │    └─ go run /app/function/main.go                │ │ │ │
│  │  │  │         └─ User Function Code                     │ │ │ │
│  │  │  │              └─ Reads FUNCTION_PAYLOAD            │ │ │ │
│  │  │  │              └─ Writes to stdout                  │ │ │ │
│  │  │  └───────────────────────────────────────────────────┘ │ │ │
│  │  └─────────────────────────────────────────────────────────┘ │ │
│  └───────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

## Data Flow Diagram

```
┌──────────┐
│  Client  │
└────┬─────┘
     │ 1. POST /functions (create)
     │    {name, runtime, code}
     ▼
┌─────────────────┐
│   Controller    │
│   (API Server)  │
└────┬────────────┘
     │ 2. Validate & Store
     │
     ├─────────────────────────┐
     │                         │
     ▼                         ▼
┌──────────────┐      ┌─────────────────┐
│  PostgreSQL  │      │ Function Storage│
│              │      │  (Filesystem)   │
│ - Metadata   │      │  - Code Files   │
└──────────────┘      └─────────────────┘
     │
     │ 3. POST /invoke
     │    {function_id, payload}
     ▼
┌─────────────────┐
│   Controller    │
└────┬────────────┘
     │ 4. Create Invocation
     │    & Enqueue
     │
     ├─────────────────────────┐
     │                         │
     ▼                         ▼
┌──────────────┐      ┌─────────────────┐
│  PostgreSQL  │      │  Redis Queue    │
│              │      │                 │
│ - Invocation │      │ - Exec Request  │
│   Record     │      │                 │
└──────────────┘      └────┬────────────┘
                           │ 5. Dequeue
                           ▼
                    ┌─────────────────┐
                    │     Worker      │
                    └────┬────────────┘
                         │ 6. Retrieve
                         │    Function
                         │
                         ├─────────────────────────┐
                         │                         │
                         ▼                         ▼
                    ┌──────────────┐      ┌─────────────────┐
                    │  PostgreSQL  │      │ Function Storage│
                    │              │      │                 │
                    │ - Function   │      │ - Code          │
                    │   Metadata   │      │                 │
                    └──────────────┘      └─────────────────┘
                         │
                         │ 7. Execute in Container
                         ▼
                    ┌─────────────────┐
                    │ Docker Engine   │
                    │                 │
                    │ ┌─────────────┐ │
                    │ │  Container  │ │
                    │ │  - Isolated │ │
                    │ │  - Limited  │ │
                    │ │  - Secure   │ │
                    │ └─────────────┘ │
                    └────┬────────────┘
                         │ 8. Result
                         ▼
                    ┌─────────────────┐
                    │     Worker      │
                    └────┬────────────┘
                         │ 9. Update
                         │    Invocation
                         ▼
                    ┌──────────────┐
                    │  PostgreSQL  │
                    │              │
                    │ - Result     │
                    │ - Metrics    │
                    │ - Status     │
                    └──────────────┘
                         │
                         │ 10. GET /invocations/{id}
                         ▼
                    ┌─────────────────┐
                    │   Controller    │
                    └────┬────────────┘
                         │ 11. Return Result
                         ▼
                    ┌──────────┐
                    │  Client  │
                    └──────────┘
```

## Component Interaction Matrix

```
┌──────────────┬──────────┬──────────┬──────────┬──────────┬──────────┐
│              │Controller│  Worker  │PostgreSQL│  Redis   │  Docker  │
├──────────────┼──────────┼──────────┼──────────┼──────────┼──────────┤
│ Controller   │    -     │    No    │   Yes    │   Yes    │    No    │
│              │          │          │ (R/W)    │ (Write)  │          │
├──────────────┼──────────┼──────────┼──────────┼──────────┼──────────┤
│ Worker       │    No    │    -     │   Yes    │   Yes    │   Yes    │
│              │          │          │ (R/W)    │ (R/W)    │ (R/W)    │
├──────────────┼──────────┼──────────┼──────────┼──────────┼──────────┤
│ PostgreSQL   │   Yes    │   Yes    │    -     │    No    │    No    │
│              │ (Serve)  │ (Serve)  │          │          │          │
├──────────────┼──────────┼──────────┼──────────┼──────────┼──────────┤
│ Redis        │   Yes    │   Yes    │    No    │    -     │    No    │
│              │ (Serve)  │ (Serve)  │          │          │          │
├──────────────┼──────────┼──────────┼──────────┼──────────┼──────────┤
│ Docker       │    No    │   Yes    │    No    │    No    │    -     │
│              │          │ (Serve)  │          │          │          │
└──────────────┴──────────┴──────────┴──────────┴──────────┴──────────┘

Legend:
- Yes (R/W): Read and Write access
- Yes (Write): Write-only access
- Yes (Serve): Serves requests
- No: No direct interaction
```

This architecture provides:
- ✅ Clear separation of concerns
- ✅ Scalable worker pool
- ✅ Reliable async processing
- ✅ Container-based isolation
- ✅ Resource management
- ✅ Production-ready design
