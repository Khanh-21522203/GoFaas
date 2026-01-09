#!/bin/sh
set -e

# Function code is mounted at /app/function
cd /app/function

# Check if main.py exists
if [ ! -f "main.py" ]; then
    echo "Error: main.py not found in /app/function"
    exit 1
fi

# Execute the Python function
exec python3 main.py
