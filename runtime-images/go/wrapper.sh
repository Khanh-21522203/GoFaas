#!/bin/sh
set -e

# Function code is mounted at /app/function
cd /app/function

# Check if main.go exists
if [ ! -f "main.go" ]; then
    echo "Error: main.go not found in /app/function"
    exit 1
fi

# Execute the Go function
exec go run main.go
