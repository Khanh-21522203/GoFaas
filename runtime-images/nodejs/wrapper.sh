#!/bin/sh
set -e

# Function code is mounted at /app/function
cd /app/function

# Check if main.js exists
if [ ! -f "main.js" ]; then
    echo "Error: main.js not found in /app/function"
    exit 1
fi

# Execute the Node.js function
exec node main.js
