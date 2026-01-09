#!/bin/bash

# FaaS Platform API Examples

BASE_URL="http://localhost:8080"

echo "=== FaaS Platform API Examples ==="
echo ""

# 1. Create a Go function
echo "1. Creating a Go function..."
GO_CODE=$(base64 -w 0 examples/functions/hello.go)
FUNCTION_RESPONSE=$(curl -s -X POST $BASE_URL/functions \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"hello-go\",
    \"version\": \"1.0.0\",
    \"runtime\": \"go\",
    \"handler\": \"main\",
    \"code\": \"$GO_CODE\",
    \"timeout\": \"30s\",
    \"memory_mb\": 128,
    \"max_concurrency\": 10,
    \"environment\": {},
    \"metadata\": {\"description\": \"Hello function in Go\"}
  }")

FUNCTION_ID=$(echo $FUNCTION_RESPONSE | jq -r '.data.id')
echo "Function created with ID: $FUNCTION_ID"
echo ""

# 2. List functions
echo "2. Listing all functions..."
curl -s $BASE_URL/functions | jq '.data[] | {id, name, runtime, version}'
echo ""

# 3. Get function details
echo "3. Getting function details..."
curl -s $BASE_URL/functions/$FUNCTION_ID | jq '.data | {id, name, runtime, handler}'
echo ""

# 4. Invoke function
echo "4. Invoking function..."
INVOCATION_RESPONSE=$(curl -s -X POST $BASE_URL/invoke \
  -H "Content-Type: application/json" \
  -d "{
    \"function_id\": \"$FUNCTION_ID\",
    \"payload\": {\"name\": \"FaaS Platform\"},
    \"headers\": {}
  }")

INVOCATION_ID=$(echo $INVOCATION_RESPONSE | jq -r '.data.invocation_id')
echo "Invocation created with ID: $INVOCATION_ID"
echo ""

# 5. Wait for execution
echo "5. Waiting for execution to complete..."
sleep 3

# 6. Get invocation result
echo "6. Getting invocation result..."
curl -s $BASE_URL/invocations/$INVOCATION_ID | jq '.data | {id, status, result, metrics}'
echo ""

# 7. List invocations
echo "7. Listing invocations for function..."
curl -s "$BASE_URL/invocations?function_id=$FUNCTION_ID" | jq '.data[] | {id, status, created_at}'
echo ""

echo "=== Examples completed ==="
