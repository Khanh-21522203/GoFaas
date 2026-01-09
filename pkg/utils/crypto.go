package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// SHA256Hash calculates SHA256 hash of data
func SHA256Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// SHA256String calculates SHA256 hash of a string
func SHA256String(s string) string {
	return SHA256Hash([]byte(s))
}
