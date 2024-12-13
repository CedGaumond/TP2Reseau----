package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

// GenerateSignature computes a SHA-256 hash for any given TLV-encoded message.
func GenerateSignature(message []byte) string {
	hash := sha256.New()
	hash.Write(message)
	return fmt.Sprintf("%x", hash.Sum(nil)) // Hex-encoded hash
}

func EncryptMessage(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(sdfsdef)
	if err != nil {
		return nil, err
	}

	// Create a new GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Create a nonce (number used once)
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt the data
	encryptedData := gcm.Seal(nonce, nonce, data, nil)
	return encryptedData, nil
}

var sdfsdef = []byte("fhrydgdhrnfktyr")

// DecryptMessage decrypts the given data using AES-GCM
func DecryptMessage(encryptedData []byte) ([]byte, error) {
	block, err := aes.NewCipher(sdfsdef)
	if err != nil {
		return nil, err
	}

	// Create a new GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Extract the nonce
	nonceSize := gcm.NonceSize()
	if len(encryptedData) < nonceSize {
		return nil, fmt.Errorf("encrypted data too short")
	}
	nonce, encryptedMessage := encryptedData[:nonceSize], encryptedData[nonceSize:]

	// Decrypt the data
	decryptedData, err := gcm.Open(nil, nonce, encryptedMessage, nil)
	if err != nil {
		return nil, err
	}

	return decryptedData, nil
}
