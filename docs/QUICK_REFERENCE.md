# FaaS Platform Quick Reference

## Quick Start Commands

```bash
# Setup
make docker-up && make migrate-up && make build-runtime-images

# Run services
make run-controller  # Terminal 1
make run-worker      # Terminal 2

# Cleanup
make docker-down && make clean
```

## API Endpoints

### Function Management

```bash
# Create function
POST /functions
{
  "name": "string",
  "version": "string",
  "runtime": "go|python|nodejs",
  "handler": "string",
  "code": "base64-encoded-string",
  "timeout": "30s",
  "memory_mb": 128,
  "max_concurrency": 10,
  "environment": {},
  "metadata": {}
}

# List functions
GET /functions?runtime=go

# Get function
GET /functions/{id}

# Update function
PUT /functions/{id}
{
  "handler": "string",
  "code": "base64-encoded-string",
  "timeout": "60s",
  "memory_mb": 256
}

# Delete function
DELETE /functions/{id}
```

### Function Invocation

```bash
# Invoke function
POST /invoke
{
  "function_id": "uuid",
  "payload": {},
  "headers": {},
  "timeout": "30s"
}

# Get invocation result
GET /invocations/{id}

# List invocations
GET /invocations?function_id=uuid&status=completed
```

### Health Check

```bash
GET /health
```

## Environment Variables

### Server
- `SERVER_ADDR` - HTTP server address (default: `:8080`)

### Database
- `DB_HOST` - PostgreSQL host (default: `localhost`)
- `DB_PORT` - PostgreSQL port (default: `5432`)
- `DB_USER` - Database user (default: `postgres`)
- `DB_PASSWORD` - Database password (default: `postgres`)
- `DB_NAME` - Database name (default: `faas`)
- `DB_SSLMODE` - SSL mode (default: `disable`)

### Redis
- `REDIS_ADDR` - Redis address (default: `localhost:6379`)
- `REDIS_PASSWORD` - Redis password (default: empty)
- `REDIS_DB` - Redis database (default: `0`)

### Storage
- `STORAGE_TYPE` - Storage type (default: `local`)
- `STORAGE_BASE_DIR` - Base directory (default: `./storage/functions`)

### Worker
- `WORKER_ID` - Worker identifier (default: auto-generated)
- `WORKER_WORK_DIR` - Work directory (default: `./storage/work`)
- `WORKER_USE_CONTAINER` - Enable container execution (default: `true`)
- `WORKER_RUNTIME_TYPE` - Runtime type: "simple" or "container" (default: `container`)

## Function Code Format

### Go Function

```go
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
	json.Unmarshal([]byte(payloadStr), &req)
	
	response := Response{
		Message: fmt.Sprintf("Hello, %s!", req.Name),
	}
	
	output, _ := json.Marshal(response)
	fmt.Println(string(output))
}
```

### Python Function

```python
#!/usr/bin/env python3
import json
import os

def main():
    payload_str = os.environ.get('FUNCTION_PAYLOAD', '{}')
    payload = json.loads(payload_str)
    
    name = payload.get('name', 'World')
    
    response = {'message': f'Hello, {name}!'}
    print(json.dumps(response))

if __name__ == '__main__':
    main()
```

### Node.js Function

```javascript
#!/usr/bin/env node

function main() {
    const payloadStr = process.env.FUNCTION_PAYLOAD || '{}';
    const payload = JSON.parse(payloadStr);
    
    const name = payload.name || 'World';
    const response = {message: `Hello, ${name}!`};
    
    console.log(JSON.stringify(response));
}

main();
```

## Common Tasks

### Create and Invoke Function

```bash
# 1. Encode function code
CODE=$(base64 -w 0 function.go)

# 2. Create function
RESPONSE=$(curl -s -X POST http://localhost:8080/functions \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"test\",\"version\":\"1.0.0\",\"runtime\":\"go\",\"handler\":\"main\",\"code\":\"$CODE\",\"timeout\":\"30s\",\"memory_mb\":128,\"max_concurrency\":10}")

# 3. Extract function ID
FUNCTION_ID=$(echo $RESPONSE | jq -r '.data.id')

# 4. Invoke function
INVOKE_RESPONSE=$(curl -s -X POST http://localhost:8080/invoke \
  -H "Content-Type: application/json" \
  -d "{\"function_id\":\"$FUNCTION_ID\",\"payload\":{\"name\":\"Test\"}}")

# 5. Extract invocation ID
INVOCATION_ID=$(echo $INVOKE_RESPONSE | jq -r '.data.invocation_id')

# 6. Wait and get result
sleep 3
curl -s http://localhost:8080/invocations/$INVOCATION_ID | jq
```

### Database Operations

```bash
# Connect to database
psql -U postgres -d faas

# List functions
SELECT id, name, version, runtime FROM functions;

# List invocations
SELECT id, function_id, status, created_at FROM invocations ORDER BY created_at DESC LIMIT 10;

# Check invocation status
SELECT status, COUNT(*) FROM invocations GROUP BY status;
```

### Redis Operations

```bash
# Connect to Redis
redis-cli

# Check queue size
LLEN faas:queue:faas_executions

# Check processing queue
LLEN faas:processing:faas_executions

# Check dead letter queue
LLEN faas:dead_letter:faas_executions

# View queue contents (non-destructive)
LRANGE faas:queue:faas_executions 0 -1
```

## Troubleshooting

### Check Service Status

```bash
# Controller health
curl http://localhost:8080/health

# Database connection
psql -U postgres -d faas -c "SELECT 1"

# Redis connection
redis-cli ping
```

### View Logs

```bash
# Controller logs (if running in background)
tail -f controller.log

# Worker logs
tail -f worker.log

# Database logs
docker-compose logs postgres

# Redis logs
docker-compose logs redis
```

### Common Issues

**Issue**: Function stays in "pending" status
**Solution**: Check worker is running and processing messages

**Issue**: "Runtime not found" error
**Solution**: Ensure Go/Python/Node.js is installed

**Issue**: Database connection error
**Solution**: Verify PostgreSQL is running and credentials are correct

**Issue**: Redis connection error
**Solution**: Verify Redis is running and address is correct

## Performance Tips

1. **Increase worker count**: Run multiple workers for parallel execution
2. **Optimize function code**: Minimize dependencies and execution time
3. **Use appropriate timeouts**: Set realistic timeouts for functions
4. **Monitor queue depth**: Scale workers based on queue size
5. **Database indexing**: Ensure proper indexes for query patterns

## Security Considerations

1. **Function isolation**: Current implementation uses process isolation
2. **Input validation**: All inputs are validated before processing
3. **Error handling**: Errors don't expose sensitive information
4. **Database access**: Use connection pooling and prepared statements
5. **Future**: Add authentication, authorization, and container isolation

## Monitoring

### Key Metrics to Track

- Function execution count
- Function execution duration
- Queue depth
- Worker utilization
- Database connection pool usage
- Error rates by function
- Invocation status distribution

### Health Indicators

- Controller responding to /health
- Worker processing messages
- Database queries succeeding
- Redis operations succeeding
- Function executions completing

## Development Workflow

```bash
# 1. Make code changes
vim internal/core/function/service.go

# 2. Format code
make fmt

# 3. Run tests
make test

# 4. Build
make build

# 5. Test locally
make run-controller
make run-worker

# 6. Commit changes
git add .
git commit -m "Add feature X"
```

## Production Checklist

- [ ] Set strong database passwords
- [ ] Configure Redis authentication
- [ ] Set up SSL/TLS for API
- [ ] Implement authentication
- [ ] Add rate limiting
- [ ] Set up monitoring
- [ ] Configure log aggregation
- [ ] Set up alerting
- [ ] Implement backup strategy
- [ ] Document runbooks
- [ ] Load test the system
- [ ] Set up CI/CD pipeline

## Resources

- **Architecture**: See `REBUILD_ARCHITECTURE.md`
- **Setup Guide**: See `GETTING_STARTED.md`
- **Implementation**: See `IMPLEMENTATION_GUIDE.md`
- **Summary**: See `IMPLEMENTATION_SUMMARY.md`
- **Container Execution**: See `CONTAINER_EXECUTION.md` (NEW)
- **Container Testing**: See `CONTAINER_TESTING.md` (NEW)
- **Container Migration**: See `CONTAINER_MIGRATION.md` (NEW)
