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
	// Get payload from environment variable
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
