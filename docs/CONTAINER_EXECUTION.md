# Container-Based Function Execution

## Overview

The FaaS platform now supports **container-based function execution** using Docker. This provides:

- **Isolation**: Each function runs in its own container
- **Security**: Functions cannot access host filesystem or other functions
- **Resource Control**: Memory and CPU limits enforced at container level
- **Consistency**: Same runtime environment across development and production

## Architecture

### Execution Flow

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Worker receives execution request from queue             │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. ContainerRuntime prepares execution                      │
│    - Writes function code to temporary directory            │
│    - Selects appropriate runtime image                      │
│    - Prepares environment variables                         │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. Docker creates container                                 │
│    - Mounts function code as read-only volume               │
│    - Sets resource limits (memory, CPU)                     │
│    - Injects payload and environment variables              │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│ 4. Container executes function                              │
│    - Wrapper script runs function code                      │
│    - Function reads payload from environment                │
│    - Function writes output to stdout                       │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│ 5. Worker collects results                                  │
│    - Captures stdout/stderr                                 │
│    - Records exit code                                      │
│    - Collects resource usage metrics                        │
│    - Removes container                                      │
└─────────────────────────────────────────────────────────────┘
```

### Container Lifecycle

```
Create → Start → Wait → Collect Logs → Remove
   ↓       ↓       ↓         ↓            ↓
 Config  Execute Timeout?  Output      Cleanup
```

## Runtime Images

### Base Images

Three pre-built runtime images are provided:

1. **faas-runtime-go:latest** - Go 1.22 runtime
2. **faas-runtime-python:latest** - Python 3.11 runtime
3. **faas-runtime-nodejs:latest** - Node.js 20 runtime

### Image Structure

Each runtime image contains:
- Language runtime and standard libraries
- Wrapper script for function execution
- Standard entry point
- Function code mount point at `/app/function`

### Building Runtime Images

```bash
# Build all runtime images
make build-runtime-images

# Or manually
cd runtime-images
./build.sh

# Verify images
docker images | grep faas-runtime
```

## Configuration

### Environment Variables

```bash
# Enable container-based execution (default: true)
export WORKER_USE_CONTAINER=true

# Or disable to use simple runtime
export WORKER_USE_CONTAINER=false

# Worker work directory for temporary files
export WORKER_WORK_DIR=./storage/work
```

### Worker Configuration

The worker automatically selects the runtime based on configuration:

```go
// In cmd/worker/main.go
if cfg.Worker.UseContainer {
    // Use container runtime
    rt, err = runtime.NewContainerRuntime(cfg.Worker.WorkDir, logger)
} else {
    // Use simple runtime (direct execution)
    rt, err = runtime.NewSimpleRuntime(cfg.Worker.WorkDir)
}
```

## Function Development

### Writing Functions

Functions remain unchanged - they still read from environment variables and write to stdout:

#### Go Function

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

func main() {
	payloadStr := os.Getenv("FUNCTION_PAYLOAD")
	
	var req Request
	json.Unmarshal([]byte(payloadStr), &req)
	
	result := map[string]string{
		"message": fmt.Sprintf("Hello, %s!", req.Name),
	}
	
	output, _ := json.Marshal(result)
	fmt.Println(string(output))
}
```

#### Python Function

```python
#!/usr/bin/env python3
import json
import os

def main():
    payload_str = os.environ.get('FUNCTION_PAYLOAD', '{}')
    payload = json.loads(payload_str)
    
    name = payload.get('name', 'World')
    result = {'message': f'Hello, {name}!'}
    
    print(json.dumps(result))

if __name__ == '__main__':
    main()
```

#### Node.js Function

```javascript
#!/usr/bin/env node

function main() {
    const payloadStr = process.env.FUNCTION_PAYLOAD || '{}';
    const payload = JSON.parse(payloadStr);
    
    const name = payload.name || 'World';
    const result = {message: `Hello, ${name}!`};
    
    console.log(JSON.stringify(result));
}

main();
```

### Deployment Workflow

1. **Write function code** (as shown above)
2. **Encode to base64**:
   ```bash
   CODE=$(base64 -w 0 function.go)
   ```
3. **Create function via API**:
   ```bash
   curl -X POST http://localhost:8080/functions \
     -H "Content-Type: application/json" \
     -d "{
       \"name\": \"my-function\",
       \"version\": \"1.0.0\",
       \"runtime\": \"go\",
       \"handler\": \"main\",
       \"code\": \"$CODE\",
       \"timeout\": \"30s\",
       \"memory_mb\": 128,
       \"max_concurrency\": 10
     }"
   ```
4. **Invoke function**:
   ```bash
   curl -X POST http://localhost:8080/invoke \
     -H "Content-Type: application/json" \
     -d "{
       \"function_id\": \"<function-id>\",
       \"payload\": {\"name\": \"Container\"}
     }"
   ```

## Resource Limits

### Memory Limits

Memory limits are enforced at the container level:

```go
// In function creation request
{
  "memory_mb": 128  // Container limited to 128 MB
}
```

If a function exceeds its memory limit, Docker will kill the container with OOM (Out of Memory).

### CPU Limits

CPU limits are enforced using Docker CPU shares:

```go
// Calculated from memory allocation
cpuShares := memoryMB / 128  // 1 share per 128 MB
```

### Timeout

Timeouts are enforced using context cancellation:

```go
// In function configuration
{
  "timeout": "30s"  // Function must complete within 30 seconds
}
```

If a function exceeds its timeout:
1. Context is cancelled
2. Container is killed with SIGKILL
3. Invocation status set to "timeout"

## Security Considerations

### Isolation

- Each function runs in a separate container
- Containers use bridge networking (isolated from host)
- Function code mounted as read-only
- No access to host filesystem
- No access to other containers

### Resource Protection

- Memory limits prevent memory exhaustion
- CPU limits prevent CPU starvation
- Timeout prevents infinite loops
- Network isolation prevents lateral movement

### Future Enhancements

- [ ] User namespaces for additional isolation
- [ ] Seccomp profiles to restrict syscalls
- [ ] AppArmor/SELinux profiles
- [ ] Network policies
- [ ] Read-only root filesystem

## Monitoring and Debugging

### Container Logs

Container logs are automatically collected and stored in the invocation result:

```bash
# Get invocation result
curl http://localhost:8080/invocations/<invocation-id>

# Response includes logs
{
  "result": "function output",
  "error": {
    "stack": "container logs if failed"
  }
}
```

### Resource Usage

Resource usage metrics are collected:

```json
{
  "metrics": {
    "duration": 1500000000,
    "memory_peak": 45678912,
    "network_in": 0,
    "network_out": 0
  }
}
```

### Debugging Failed Executions

1. **Check invocation status**:
   ```bash
   curl http://localhost:8080/invocations/<id>
   ```

2. **Review error details**:
   ```json
   {
     "error": {
       "type": "RuntimeError",
       "message": "Function exited with code 1",
       "stack": "error logs from container"
     }
   }
   ```

3. **Test function locally**:
   ```bash
   # Run function in container manually
   docker run --rm \
     -v $(pwd)/function:/app/function:ro \
     -e FUNCTION_PAYLOAD='{"name":"test"}' \
     faas-runtime-go:latest
   ```

## Performance Considerations

### Container Startup Overhead

- **Cold start**: ~100-500ms for container creation and startup
- **Warm start**: Not implemented yet (future enhancement)

### Optimization Strategies

1. **Pre-pull images**: Ensure runtime images are pre-pulled on workers
2. **Minimize function size**: Smaller code = faster volume mount
3. **Optimize base images**: Use Alpine-based images for smaller size
4. **Future: Container pooling**: Reuse containers for multiple invocations

## Troubleshooting

### Issue: "Failed to create docker client"

**Cause**: Docker daemon not running or not accessible

**Solution**:
```bash
# Check Docker is running
docker ps

# Verify Docker socket permissions
ls -la /var/run/docker.sock

# On Linux, add user to docker group
sudo usermod -aG docker $USER
```

### Issue: "Failed to pull image"

**Cause**: Runtime images not built

**Solution**:
```bash
# Build runtime images
make build-runtime-images

# Verify images exist
docker images | grep faas-runtime
```

### Issue: "Container execution timed out"

**Cause**: Function takes longer than configured timeout

**Solution**:
```bash
# Increase timeout in function configuration
curl -X PUT http://localhost:8080/functions/<id> \
  -H "Content-Type: application/json" \
  -d '{"timeout": "60s"}'
```

### Issue: "Container killed (OOM)"

**Cause**: Function exceeded memory limit

**Solution**:
```bash
# Increase memory limit
curl -X PUT http://localhost:8080/functions/<id> \
  -H "Content-Type: application/json" \
  -d '{"memory_mb": 256}'
```

## Comparison: Simple vs Container Runtime

| Feature | Simple Runtime | Container Runtime |
|---------|---------------|-------------------|
| Isolation | Process-level | Container-level |
| Security | Low | High |
| Resource Limits | None | Memory, CPU |
| Startup Time | Fast (~10ms) | Slower (~100-500ms) |
| Overhead | Minimal | Moderate |
| Production Ready | No | Yes |
| Use Case | Development | Production |

## Migration Guide

### From Simple to Container Runtime

1. **Build runtime images**:
   ```bash
   make build-runtime-images
   ```

2. **Update worker configuration**:
   ```bash
   export WORKER_USE_CONTAINER=true
   ```

3. **Restart worker**:
   ```bash
   make run-worker
   ```

4. **Test existing functions** - No code changes required!

### Rollback to Simple Runtime

```bash
export WORKER_USE_CONTAINER=false
make run-worker
```

## Future Enhancements

### Short Term
- [ ] Container pooling for warm starts
- [ ] Better resource usage metrics
- [ ] Container health checks
- [ ] Custom base images per function

### Long Term
- [ ] Kubernetes integration
- [ ] Multi-node worker pools
- [ ] Function auto-scaling
- [ ] GPU support
- [ ] Custom runtime support

## References

- Docker SDK for Go: https://docs.docker.com/engine/api/sdk/
- Container best practices: https://docs.docker.com/develop/dev-best-practices/
- Resource constraints: https://docs.docker.com/config/containers/resource_constraints/
