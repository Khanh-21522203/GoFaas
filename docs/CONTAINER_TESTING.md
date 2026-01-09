# Container Execution Testing Guide

## Quick Start Test

### 1. Setup

```bash
# Start infrastructure
make docker-up
make migrate-up

# Build runtime images
make build-runtime-images

# Verify images
docker images | grep faas-runtime
```

Expected output:
```
faas-runtime-go        latest    abc123    2 minutes ago    500MB
faas-runtime-python    latest    def456    2 minutes ago    200MB
faas-runtime-nodejs    latest    ghi789    2 minutes ago    180MB
```

### 2. Start Services

```bash
# Terminal 1: Controller
export WORKER_USE_CONTAINER=true
make run-controller

# Terminal 2: Worker
export WORKER_USE_CONTAINER=true
make run-worker
```

### 3. Test Container Execution

#### Test Go Function

```bash
# Create test function
cat > /tmp/test-go.go << 'EOF'
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
		"message": fmt.Sprintf("Hello from container, %s!", req.Name),
		"runtime": "go",
	}
	
	output, _ := json.Marshal(result)
	fmt.Println(string(output))
}
EOF

# Encode and create function
CODE=$(base64 -w 0 /tmp/test-go.go)

RESPONSE=$(curl -s -X POST http://localhost:8080/functions \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"test-container-go\",
    \"version\": \"1.0.0\",
    \"runtime\": \"go\",
    \"handler\": \"main\",
    \"code\": \"$CODE\",
    \"timeout\": \"30s\",
    \"memory_mb\": 128,
    \"max_concurrency\": 10
  }")

echo "Function created:"
echo $RESPONSE | jq '.data | {id, name, runtime}'

FUNCTION_ID=$(echo $RESPONSE | jq -r '.data.id')

# Invoke function
INVOKE_RESPONSE=$(curl -s -X POST http://localhost:8080/invoke \
  -H "Content-Type: application/json" \
  -d "{
    \"function_id\": \"$FUNCTION_ID\",
    \"payload\": {\"name\": \"Container Test\"}
  }")

echo "Invocation created:"
echo $INVOKE_RESPONSE | jq '.data'

INVOCATION_ID=$(echo $INVOKE_RESPONSE | jq -r '.data.invocation_id')

# Wait for execution
echo "Waiting for execution..."
sleep 5

# Get result
curl -s http://localhost:8080/invocations/$INVOCATION_ID | jq '.data'
```

Expected result:
```json
{
  "id": "...",
  "function_id": "...",
  "status": "completed",
  "result": "{\"message\":\"Hello from container, Container Test!\",\"runtime\":\"go\"}",
  "metrics": {
    "duration": 1500000000,
    "memory_peak": 45678912
  }
}
```

#### Test Python Function

```bash
cat > /tmp/test-python.py << 'EOF'
#!/usr/bin/env python3
import json
import os
import sys

def main():
    payload_str = os.environ.get('FUNCTION_PAYLOAD', '{}')
    payload = json.loads(payload_str)
    
    name = payload.get('name', 'World')
    
    result = {
        'message': f'Hello from Python container, {name}!',
        'runtime': 'python',
        'version': sys.version
    }
    
    print(json.dumps(result))

if __name__ == '__main__':
    main()
EOF

CODE=$(base64 -w 0 /tmp/test-python.py)

curl -X POST http://localhost:8080/functions \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"test-container-python\",
    \"version\": \"1.0.0\",
    \"runtime\": \"python\",
    \"handler\": \"main\",
    \"code\": \"$CODE\",
    \"timeout\": \"30s\",
    \"memory_mb\": 128,
    \"max_concurrency\": 10
  }" | jq '.data.id'
```

#### Test Node.js Function

```bash
cat > /tmp/test-nodejs.js << 'EOF'
#!/usr/bin/env node

function main() {
    const payloadStr = process.env.FUNCTION_PAYLOAD || '{}';
    const payload = JSON.parse(payloadStr);
    
    const name = payload.name || 'World';
    
    const result = {
        message: `Hello from Node.js container, ${name}!`,
        runtime: 'nodejs',
        version: process.version
    };
    
    console.log(JSON.stringify(result));
}

main();
EOF

CODE=$(base64 -w 0 /tmp/test-nodejs.js)

curl -X POST http://localhost:8080/functions \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"test-container-nodejs\",
    \"version\": \"1.0.0\",
    \"runtime\": \"nodejs\",
    \"handler\": \"main\",
    \"code\": \"$CODE\",
    \"timeout\": \"30s\",
    \"memory_mb\": 128,
    \"max_concurrency\": 10
  }" | jq '.data.id'
```

## Advanced Tests

### Test Resource Limits

#### Memory Limit Test

```bash
# Create function that allocates memory
cat > /tmp/memory-test.py << 'EOF'
#!/usr/bin/env python3
import json

def main():
    # Try to allocate 200MB (will fail if limit is 128MB)
    try:
        data = bytearray(200 * 1024 * 1024)
        result = {'status': 'success', 'allocated': '200MB'}
    except MemoryError:
        result = {'status': 'failed', 'error': 'MemoryError'}
    
    print(json.dumps(result))

if __name__ == '__main__':
    main()
EOF

CODE=$(base64 -w 0 /tmp/memory-test.py)

# Create with 128MB limit
curl -X POST http://localhost:8080/functions \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"memory-test\",
    \"version\": \"1.0.0\",
    \"runtime\": \"python\",
    \"handler\": \"main\",
    \"code\": \"$CODE\",
    \"timeout\": \"30s\",
    \"memory_mb\": 128,
    \"max_concurrency\": 10
  }"
```

#### Timeout Test

```bash
# Create function that sleeps
cat > /tmp/timeout-test.py << 'EOF'
#!/usr/bin/env python3
import json
import time

def main():
    # Sleep for 60 seconds (will timeout if limit is 30s)
    time.sleep(60)
    result = {'status': 'completed'}
    print(json.dumps(result))

if __name__ == '__main__':
    main()
EOF

CODE=$(base64 -w 0 /tmp/timeout-test.py)

# Create with 5s timeout
curl -X POST http://localhost:8080/functions \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"timeout-test\",
    \"version\": \"1.0.0\",
    \"runtime\": \"python\",
    \"handler\": \"main\",
    \"code\": \"$CODE\",
    \"timeout\": \"5s\",
    \"memory_mb\": 128,
    \"max_concurrency\": 10
  }"

# Invoke and check for timeout status
```

### Test Error Handling

```bash
# Create function that fails
cat > /tmp/error-test.py << 'EOF'
#!/usr/bin/env python3
import sys

def main():
    print("This function will fail", file=sys.stderr)
    sys.exit(1)

if __name__ == '__main__':
    main()
EOF

CODE=$(base64 -w 0 /tmp/error-test.py)

curl -X POST http://localhost:8080/functions \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"error-test\",
    \"version\": \"1.0.0\",
    \"runtime\": \"python\",
    \"handler\": \"main\",
    \"code\": \"$CODE\",
    \"timeout\": \"30s\",
    \"memory_mb\": 128,
    \"max_concurrency\": 10
  }"
```

## Manual Container Testing

### Test Runtime Image Directly

```bash
# Test Go runtime
echo 'package main
import "fmt"
func main() {
    fmt.Println("Hello from Go!")
}' > /tmp/test/main.go

docker run --rm \
  -v /tmp/test:/app/function:ro \
  faas-runtime-go:latest

# Test Python runtime
echo 'print("Hello from Python!")' > /tmp/test/main.py

docker run --rm \
  -v /tmp/test:/app/function:ro \
  faas-runtime-python:latest

# Test Node.js runtime
echo 'console.log("Hello from Node.js!")' > /tmp/test/main.js

docker run --rm \
  -v /tmp/test:/app/function:ro \
  faas-runtime-nodejs:latest
```

### Test with Payload

```bash
# Create function that uses payload
cat > /tmp/test/main.py << 'EOF'
import json
import os

payload_str = os.environ.get('FUNCTION_PAYLOAD', '{}')
payload = json.loads(payload_str)
print(f"Received: {payload}")
EOF

docker run --rm \
  -v /tmp/test:/app/function:ro \
  -e FUNCTION_PAYLOAD='{"name":"Test","value":123}' \
  faas-runtime-python:latest
```

## Monitoring Container Execution

### Watch Docker Containers

```bash
# In a separate terminal
watch -n 1 'docker ps -a | grep faas-runtime'
```

### Monitor Worker Logs

```bash
# Watch worker logs for container operations
tail -f worker.log | grep -i container
```

### Check Docker Stats

```bash
# Monitor resource usage
docker stats --no-stream
```

## Debugging

### Check Container Logs

```bash
# List recent containers
docker ps -a --filter "ancestor=faas-runtime-go" --format "{{.ID}} {{.Status}}"

# View logs from specific container
docker logs <container-id>
```

### Inspect Failed Containers

```bash
# Keep failed containers for inspection
# Modify docker/client.go temporarily:
# AutoRemove: false

# Then inspect
docker inspect <container-id>
docker logs <container-id>
```

### Test Function Locally

```bash
# Run function outside FaaS platform
mkdir -p /tmp/debug
echo 'your function code' > /tmp/debug/main.go

docker run -it --rm \
  -v /tmp/debug:/app/function:ro \
  -e FUNCTION_PAYLOAD='{"test":"data"}' \
  faas-runtime-go:latest
```

## Performance Testing

### Measure Cold Start Time

```bash
# Create simple function
cat > /tmp/perf-test.py << 'EOF'
import json
print(json.dumps({'status': 'ok'}))
EOF

CODE=$(base64 -w 0 /tmp/perf-test.py)

# Create function
FUNC_ID=$(curl -s -X POST http://localhost:8080/functions \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"perf-test\",
    \"version\": \"1.0.0\",
    \"runtime\": \"python\",
    \"handler\": \"main\",
    \"code\": \"$CODE\",
    \"timeout\": \"30s\",
    \"memory_mb\": 128,
    \"max_concurrency\": 10
  }" | jq -r '.data.id')

# Invoke and measure
time curl -s -X POST http://localhost:8080/invoke \
  -H "Content-Type: application/json" \
  -d "{\"function_id\": \"$FUNC_ID\", \"payload\": {}}"
```

### Concurrent Execution Test

```bash
# Invoke function multiple times concurrently
for i in {1..10}; do
  curl -s -X POST http://localhost:8080/invoke \
    -H "Content-Type: application/json" \
    -d "{\"function_id\": \"$FUNC_ID\", \"payload\": {\"iteration\": $i}}" &
done

wait
echo "All invocations completed"
```

## Cleanup

```bash
# Remove all FaaS containers
docker ps -a | grep faas-runtime | awk '{print $1}' | xargs docker rm -f

# Remove runtime images
docker rmi faas-runtime-go:latest
docker rmi faas-runtime-python:latest
docker rmi faas-runtime-nodejs:latest

# Clean work directory
rm -rf ./storage/work/*
```

## Troubleshooting

### Issue: Runtime image not found

```bash
# Rebuild images
make build-runtime-images

# Or manually
cd runtime-images
./build.sh
```

### Issue: Permission denied on Docker socket

```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Logout and login again
```

### Issue: Container creation fails

```bash
# Check Docker daemon
docker info

# Check disk space
df -h

# Check Docker logs
journalctl -u docker -n 50
```

### Issue: Function times out immediately

```bash
# Check worker logs
tail -f worker.log

# Verify timeout configuration
curl http://localhost:8080/functions/<id> | jq '.data.config.timeout'
```

## Success Criteria

✅ All three runtime images build successfully
✅ Functions execute in containers (verify with `docker ps`)
✅ Function output captured correctly
✅ Resource limits enforced
✅ Timeouts work correctly
✅ Failed functions return proper error messages
✅ Containers cleaned up after execution

## Next Steps

After successful testing:
1. Review `CONTAINER_EXECUTION.md` for architecture details
2. Explore custom runtime images
3. Implement container pooling for warm starts
4. Add monitoring and metrics collection
5. Deploy to production environment
