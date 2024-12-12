package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
)

// GenerateSignature computes a SHA-256 hash for any given TLV-encoded message.
func GenerateSignature(message []byte) string {
	hash := sha256.New()
	hash.Write(message)
	return fmt.Sprintf("%x", hash.Sum(nil)) // Hex-encoded hash
}

// GenerateRandomSignature creates a random signature for the client
func GenerateRandomSignature() string {
	// Create a random byte slice for the signature
	signature := make([]byte, 32) // 32 bytes = 256 bits
	_, err := rand.Read(signature)
	if err != nil {
		log.Fatalf("Failed to generate random signature: %v", err)
	}
	return hex.EncodeToString(signature)
}
