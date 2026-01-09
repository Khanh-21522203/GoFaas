package executor

import (
	"fmt"
	"os/exec"
)

// ExecuteFunction runs a function using a predefined runtime
func ExecuteFunction(functionName string) (string, error) {
	// Assume the function is a Go script stored as "functions/<functionName>.go"
	codePath := fmt.Sprintf("functions/%s.go", functionName)

	// Run the Go script
	cmd := exec.Command("go", "run", codePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("execution failed: %v", err)
	}

	return string(output), nil
}
