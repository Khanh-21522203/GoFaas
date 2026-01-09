# Getting Started with FaaS Platform

This guide will walk you through setting up and running the FaaS platform from scratch.

## Prerequisites

Ensure you have the following installed:

- **Go 1.22.5+**: [Download](https://golang.org/dl/)
- **PostgreSQL 15+**: [Download](https://www.postgresql.org/download/)
- **Redis 7+**: [Download](https://redis.io/download)
- **Docker & Docker Compose**: [Download](https://www.docker.com/products/docker-desktop)

For function execution, you'll also need:
- **Python 3.x** (for Python functions)
- **Node.js 18+** (for Node.js functions)

## Step 1: Clone and Setup

```bash
# Navigate to project directory
cd GoFaas

# Install Go dependencies
make deps

# Verify dependencies
go mod verify
```

## Step 2: Start Infrastructure

### Option A: Using Docker Compose (Recommended)

```bash
# Start PostgreSQL and Redis
make docker-up

# Verify services are running
docker-compose ps
```

### Option B: Manual Setup

If you prefer to run PostgreSQL and Redis manually:

```bash
# Start PostgreSQL (example using Homebrew on macOS)
brew services start postgresql@15

# Start Redis
brew services start redis

# Or use your system's service manager
```

## Step 3: Database Setup

```bash
# Run database migrations
make migrate-up

# Verify tables were created
psql -U postgres -d faas -c "\dt"
```

You should see tables: `functions`, `invocations`, `users`, `function_permissions`

## Step 4: Start Services

### Terminal 1: Start Controller

```bash
# Start the controller (API server)
make run-controller
```

You should see:
```
[2024-01-09T10:00:00Z] INFO: Starting FaaS Controller
[2024-01-09T10:00:00Z] INFO: Database connection established
[2024-01-09T10:00:00Z] INFO: Redis connection established
[2024-01-09T10:00:00Z] INFO: Starting API server addr=:8080
```

### Terminal 2: Start Worker

```bash
# Start the worker (function executor)
make run-worker
```

You should see:
```
[2024-01-09T10:00:00Z] INFO: Starting FaaS Worker worker_id=worker-1234567890
[2024-01-09T10:00:00Z] INFO: Database connection established
[2024-01-09T10:00:00Z] INFO: Redis connection established
[2024-01-09T10:00:00Z] INFO: Worker starting
```

## Step 5: Test the Platform

### Health Check

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{"status":"healthy"}
```

### Create a Function

Let's create a simple Go function:

```bash
# First, encode the function code
cat > /tmp/hello.go << 'EOF'
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Request struct {
	Name string `json:"name"`
}

type Response struct {
	Message string `json:"message"`
}

func main() {
	payloadStr := os.Getenv("FUNCTION_PAYLOAD")
	
	var req Request
	if payloadStr != "" {
		json.Unmarshal([]byte(payloadStr), &req)
	}
	
	name := req.Name
	if name == "" {
		name = "World"
	}
	
	response := Response{
		Message: fmt.Sprintf("Hello, %s!", name),
	}
	
	output, _ := json.Marshal(response)
	fmt.Println(string(output))
}
EOF

# Encode to base64
CODE=$(base64 -w 0 /tmp/hello.go)

# Create the function
curl -X POST http://localhost:8080/functions \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"hello\",
    \"version\": \"1.0.0\",
    \"runtime\": \"go\",
    \"handler\": \"main\",
    \"code\": \"$CODE\",
    \"timeout\": \"30s\",
    \"memory_mb\": 128,
    \"max_concurrency\": 10,
    \"environment\": {},
    \"metadata\": {}
  }"
```

Expected response:
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "hello",
    "version": "1.0.0",
    "runtime": "go",
    ...
  }
}
```

Save the `id` from the response - you'll need it for invocation.

### List Functions

```bash
curl http://localhost:8080/functions
```

### Invoke the Function

```bash
# Replace FUNCTION_ID with the ID from the create response
FUNCTION_ID="550e8400-e29b-41d4-a716-446655440000"

curl -X POST http://localhost:8080/invoke \
  -H "Content-Type: application/json" \
  -d "{
    \"function_id\": \"$FUNCTION_ID\",
    \"payload\": {\"name\": \"FaaS Platform\"},
    \"headers\": {}
  }"
```

Expected response:
```json
{
  "success": true,
  "data": {
    "invocation_id": "660e8400-e29b-41d4-a716-446655440001",
    "function_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "pending",
    "created_at": "2024-01-09T10:05:00Z"
  }
}
```

### Check Invocation Result

```bash
# Replace INVOCATION_ID with the ID from the invoke response
INVOCATION_ID="660e8400-e29b-41d4-a716-446655440001"

# Wait a few seconds for execution, then check result
sleep 3

curl http://localhost:8080/invocations/$INVOCATION_ID
```

Expected response:
```json
{
  "success": true,
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "function_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "completed",
    "result": "{\"message\":\"Hello, FaaS Platform!\"}",
    "metrics": {
      "duration": 1500000000,
      "cpu_time": 0,
      "memory_peak": 0,
      "network_in": 0,
      "network_out": 0
    },
    "created_at": "2024-01-09T10:05:00Z",
    "started_at": "2024-01-09T10:05:01Z",
    "completed_at": "2024-01-09T10:05:02Z"
  }
}
```

## Step 6: Try Different Runtimes

### Python Function

```bash
# Create Python function
cat > /tmp/hello.py << 'EOF'
#!/usr/bin/env python3
import json
import os

def main():
    payload_str = os.environ.get('FUNCTION_PAYLOAD', '{}')
    
    try:
        payload = json.loads(payload_str)
    except:
        payload = {}
    
    name = payload.get('name', 'World')
    
    response = {
        'message': f'Hello from Python, {name}!'
    }
    
    print(json.dumps(response))

if __name__ == '__main__':
    main()
EOF

CODE=$(base64 -w 0 /tmp/hello.py)

curl -X POST http://localhost:8080/functions \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"hello-python\",
    \"version\": \"1.0.0\",
    \"runtime\": \"python\",
    \"handler\": \"main\",
    \"code\": \"$CODE\",
    \"timeout\": \"30s\",
    \"memory_mb\": 128,
    \"max_concurrency\": 10
  }"
```

### Node.js Function

```bash
# Create Node.js function
cat > /tmp/hello.js << 'EOF'
#!/usr/bin/env node

function main() {
    const payloadStr = process.env.FUNCTION_PAYLOAD || '{}';
    
    let payload;
    try {
        payload = JSON.parse(payloadStr);
    } catch (e) {
        payload = {};
    }
    
    const name = payload.name || 'World';
    
    const response = {
        message: `Hello from Node.js, ${name}!`
    };
    
    console.log(JSON.stringify(response));
}

main();
EOF

CODE=$(base64 -w 0 /tmp/hello.js)

curl -X POST http://localhost:8080/functions \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"hello-nodejs\",
    \"version\": \"1.0.0\",
    \"runtime\": \"nodejs\",
    \"handler\": \"main\",
    \"code\": \"$CODE\",
    \"timeout\": \"30s\",
    \"memory_mb\": 128,
    \"max_concurrency\": 10
  }"
```

## Troubleshooting

### Controller won't start

**Problem**: Database connection error

**Solution**:
```bash
# Check PostgreSQL is running
docker-compose ps postgres

# Check connection
psql -U postgres -d faas -c "SELECT 1"

# Verify environment variables
echo $DB_HOST $DB_PORT $DB_USER
```

### Worker won't start

**Problem**: Redis connection error

**Solution**:
```bash
# Check Redis is running
docker-compose ps redis

# Test Redis connection
redis-cli ping

# Verify environment variables
echo $REDIS_ADDR
```

### Function execution fails

**Problem**: Runtime not found

**Solution**:
```bash
# Verify runtime is installed
go version
python3 --version
node --version

# Check worker logs for specific error
```

### Invocation stays in "pending" status

**Problem**: Worker not processing messages

**Solution**:
```bash
# Check worker is running
ps aux | grep worker

# Check Redis queue
redis-cli LLEN faas:queue:faas_executions

# Restart worker
make run-worker
```

## Next Steps

1. **Explore the API**: Check out `examples/api-examples.sh` for more examples
2. **Read the Architecture**: See `REBUILD_ARCHITECTURE.md` for system design
3. **Customize Configuration**: Set environment variables for your setup
4. **Add Monitoring**: Integrate with Prometheus/Grafana (future enhancement)
5. **Deploy to Production**: Use Kubernetes manifests (see deployment docs)

## Cleanup

```bash
# Stop services
# Press Ctrl+C in controller and worker terminals

# Stop infrastructure
make docker-down

# Clean storage
make clean

# Drop database (optional)
make migrate-down
```

## Support

For issues or questions:
- Check the design documents in the repository
- Review the implementation guide
- Check logs for error messages
