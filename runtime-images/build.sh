#!/bin/bash
set -e

echo "Building FaaS runtime images..."

# Build Go runtime
echo "Building Go runtime image..."
cd go
docker build -t faas-runtime-go:latest .
cd ..

# Build Python runtime
echo "Building Python runtime image..."
cd python
docker build -t faas-runtime-python:latest .
cd ..

# Build Node.js runtime
echo "Building Node.js runtime image..."
cd nodejs
docker build -t faas-runtime-nodejs:latest .
cd ..

echo "All runtime images built successfully!"
echo ""
echo "Available images:"
docker images | grep faas-runtime
