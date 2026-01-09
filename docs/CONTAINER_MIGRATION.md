# Migration Guide: Simple to Container Runtime

## Overview

This guide helps you migrate from the simple (process-based) runtime to the container-based runtime for production deployment.

## Why Migrate?

| Aspect | Simple Runtime | Container Runtime |
|--------|---------------|-------------------|
| **Isolation** | Process-level only | Full container isolation |
| **Security** | Functions access host | Isolated from host |
| **Resource Limits** | Not enforced | Memory & CPU limits |
| **Multi-tenancy** | Not safe | Safe for multiple users |
| **Production Ready** | No | Yes |

## Prerequisites

Before migrating, ensure you have:

- [x] Docker Engine installed and running
- [x] Docker socket accessible (`/var/run/docker.sock`)
- [x] Sufficient disk space for images (~1GB)
- [x] User has Docker permissions

## Migration Steps

### Step 1: Verify Docker Installation

```bash
# Check Docker is running
docker info

# Verify Docker version (20.10+ recommended)
docker --version

# Test Docker access
docker ps
```

If you get permission errors:
```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Logout and login again, or:
newgrp docker
```

### Step 2: Build Runtime Images

```bash
# Build all runtime images
make build-runtime-images

# This will build:
# - faas-runtime-go:latest
# - faas-runtime-python:latest
# - faas-runtime-nodejs:latest
```

Verify images:
```bash
docker images | grep faas-runtime
```

Expected output:
```
faas-runtime-go        latest    abc123    2 minutes ago    500MB
faas-runtime-python    latest    def456    2 minutes ago    200MB
faas-runtime-nodejs    latest    ghi789    2 minutes ago    180MB
```

### Step 3: Update Configuration

#### Option A: Environment Variables

```bash
# Enable container runtime
export WORKER_USE_CONTAINER=true

# Restart worker
make run-worker
```

#### Option B: Configuration File

Create `.env` file:
```bash
# Worker configuration
WORKER_USE_CONTAINER=true
WORKER_WORK_DIR=./storage/work
```

Load and restart:
```bash
source .env
make run-worker
```

### Step 4: Test Container Execution

Create a test function:

```bash
# Simple test function
cat > /tmp/test.py << 'EOF'
import json
print(json.dumps({'status': 'container execution works!'}))
EOF

CODE=$(base64 -w 0 /tmp/test.py)

# Create function
FUNC_ID=$(curl -s -X POST http://localhost:8080/functions \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"migration-test\",
    \"version\": \"1.0.0\",
    \"runtime\": \"python\",
    \"handler\": \"main\",
    \"code\": \"$CODE\",
    \"timeout\": \"30s\",
    \"memory_mb\": 128,
    \"max_concurrency\": 10
  }" | jq -r '.data.id')

# Invoke function
INV_ID=$(curl -s -X POST http://localhost:8080/invoke \
  -H "Content-Type: application/json" \
  -d "{\"function_id\": \"$FUNC_ID\", \"payload\": {}}" \
  | jq -r '.data.invocation_id')

# Wait and check result
sleep 3
curl -s http://localhost:8080/invocations/$INV_ID | jq '.data.status'
```

Expected: `"completed"`

### Step 5: Verify Container Execution

While a function is running, check Docker:

```bash
# In another terminal
watch -n 1 'docker ps | grep faas-runtime'
```

You should see containers being created and destroyed.

### Step 6: Test Existing Functions

All existing functions should work without modification:

```bash
# List existing functions
curl -s http://localhost:8080/functions | jq '.data[] | {id, name, runtime}'

# Invoke each function to verify
# (Use the invocation API as shown above)
```

### Step 7: Monitor Performance

Compare execution times:

```bash
# Before migration (simple runtime)
# Average: ~50-100ms

# After migration (container runtime)
# Average: ~200-500ms (includes container startup)
```

The increased latency is expected due to container creation overhead.

## Rollback Plan

If you need to rollback:

### Quick Rollback

```bash
# Disable container runtime
export WORKER_USE_CONTAINER=false

# Restart worker
make run-worker
```

### Permanent Rollback

1. Update configuration:
   ```bash
   # In .env or environment
   WORKER_USE_CONTAINER=false
   ```

2. Restart services:
   ```bash
   # Stop worker
   pkill -f "worker"
   
   # Start with simple runtime
   make run-worker
   ```

3. Verify:
   ```bash
   # Check worker logs
   tail -f worker.log | grep "runtime"
   
   # Should see: "Initializing simple runtime"
   ```

## Troubleshooting

### Issue: Docker daemon not accessible

**Symptoms**:
```
Failed to create docker client: Cannot connect to Docker daemon
```

**Solution**:
```bash
# Check Docker is running
sudo systemctl status docker

# Start Docker
sudo systemctl start docker

# Enable on boot
sudo systemctl enable docker
```

### Issue: Permission denied on Docker socket

**Symptoms**:
```
permission denied while trying to connect to Docker daemon socket
```

**Solution**:
```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Apply changes (logout/login or)
newgrp docker

# Verify
docker ps
```

### Issue: Runtime images not found

**Symptoms**:
```
Failed to ensure runtime image: image not found
```

**Solution**:
```bash
# Rebuild images
make build-runtime-images

# Verify
docker images | grep faas-runtime

# If build fails, check Dockerfiles
cd runtime-images
./build.sh
```

### Issue: Container creation fails

**Symptoms**:
```
Failed to create container: no space left on device
```

**Solution**:
```bash
# Check disk space
df -h

# Clean up Docker
docker system prune -a

# Remove unused images
docker image prune -a
```

### Issue: Functions timeout immediately

**Symptoms**:
All functions show status "timeout" within seconds

**Solution**:
```bash
# Check worker logs
tail -f worker.log

# Verify timeout configuration
curl http://localhost:8080/functions/<id> | jq '.data.config.timeout'

# Increase timeout if needed
curl -X PUT http://localhost:8080/functions/<id> \
  -H "Content-Type: application/json" \
  -d '{"timeout": "60s"}'
```

### Issue: High memory usage

**Symptoms**:
Worker process using excessive memory

**Solution**:
```bash
# Check Docker stats
docker stats --no-stream

# Reduce function memory limits
# Update function configuration to use less memory

# Clean up stopped containers
docker container prune
```

## Performance Optimization

### Pre-pull Images

Pre-pull runtime images on all workers:

```bash
# On each worker node
docker pull faas-runtime-go:latest
docker pull faas-runtime-python:latest
docker pull faas-runtime-nodejs:latest
```

### Adjust Resource Limits

Fine-tune resource limits based on your workload:

```bash
# For memory-intensive functions
{
  "memory_mb": 512,  # Increase memory
  "timeout": "60s"   # Increase timeout
}

# For CPU-intensive functions
{
  "memory_mb": 256,
  "timeout": "120s"  # Longer timeout
}
```

### Monitor Container Overhead

```bash
# Check container creation time
time docker run --rm faas-runtime-python:latest echo "test"

# Typical: 100-300ms
```

## Production Deployment

### Multi-Worker Setup

Deploy multiple workers for high availability:

```bash
# Worker 1
WORKER_ID=worker-1 WORKER_USE_CONTAINER=true make run-worker

# Worker 2
WORKER_ID=worker-2 WORKER_USE_CONTAINER=true make run-worker

# Worker 3
WORKER_ID=worker-3 WORKER_USE_CONTAINER=true make run-worker
```

### Resource Monitoring

Set up monitoring for:
- Container creation rate
- Container execution time
- Memory usage per container
- CPU usage per container
- Failed container count

### Logging

Configure centralized logging:
```bash
# Docker logging driver
docker run --log-driver=syslog ...

# Or use Docker logging plugins
```

## Validation Checklist

After migration, verify:

- [ ] All runtime images built successfully
- [ ] Worker starts with container runtime
- [ ] Test function executes in container
- [ ] Existing functions still work
- [ ] Resource limits enforced
- [ ] Timeouts work correctly
- [ ] Failed functions return proper errors
- [ ] Containers cleaned up after execution
- [ ] Performance acceptable
- [ ] No Docker errors in logs

## Next Steps

After successful migration:

1. **Monitor Performance**: Track execution times and resource usage
2. **Optimize Images**: Reduce image sizes for faster startup
3. **Implement Pooling**: Add container pooling for warm starts (future)
4. **Add Monitoring**: Integrate with Prometheus/Grafana
5. **Security Hardening**: Add seccomp profiles, AppArmor policies
6. **Scale Workers**: Add more workers based on load

## Support

For issues during migration:
1. Check worker logs: `tail -f worker.log`
2. Check Docker logs: `docker logs <container-id>`
3. Review `CONTAINER_EXECUTION.md` for architecture details
4. Review `CONTAINER_TESTING.md` for testing procedures

## Conclusion

Container-based execution provides:
- ✅ Production-ready isolation
- ✅ Resource limit enforcement
- ✅ Better security
- ✅ Multi-tenancy support

The migration is straightforward and can be rolled back if needed. All existing functions work without modification.
