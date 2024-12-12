package main

import (
	"crypto/sha256"
	"fmt"
)

// GenerateSignature computes a SHA-256 hash for any given TLV-encoded message.
func GenerateSignature(message []byte) string {
	hash := sha256.New()
	hash.Write(message)
	return fmt.Sprintf("%x", hash.Sum(nil)) // Hex-encoded hash
}
