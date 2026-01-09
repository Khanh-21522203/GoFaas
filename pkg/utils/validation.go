package utils

import (
	"fmt"
	"regexp"
)

var (
	// FunctionNameRegex validates function names (alphanumeric, hyphens, underscores)
	FunctionNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	
	// VersionRegex validates semantic versions
	VersionRegex = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
)

// ValidateFunctionName validates function name format
func ValidateFunctionName(name string) error {
	if name == "" {
		return fmt.Errorf("function name cannot be empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("function name too long (max 255 characters)")
	}
	if !FunctionNameRegex.MatchString(name) {
		return fmt.Errorf("function name must contain only alphanumeric characters, hyphens, and underscores")
	}
	return nil
}

// ValidateVersion validates version format
func ValidateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}
	if !VersionRegex.MatchString(version) {
		return fmt.Errorf("version must follow semantic versioning (e.g., 1.0.0)")
	}
	return nil
}
