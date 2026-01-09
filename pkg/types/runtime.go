package types

// RuntimeType represents supported function runtimes
type RuntimeType string

const (
	RuntimeGo     RuntimeType = "go"
	RuntimePython RuntimeType = "python"
	RuntimeNodeJS RuntimeType = "nodejs"
)

// IsValid checks if the runtime type is supported
func (r RuntimeType) IsValid() bool {
	switch r {
	case RuntimeGo, RuntimePython, RuntimeNodeJS:
		return true
	default:
		return false
	}
}

// String returns the string representation of the runtime
func (r RuntimeType) String() string {
	return string(r)
}
