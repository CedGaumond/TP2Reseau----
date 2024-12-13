package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/notnil/chess"
	"io"
	"log"
)

func encryptAES(key, message []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Generate a random IV
	iv := make([]byte, aes.BlockSize)
	_, err = rand.Read(iv)
	if err != nil {
		return nil, err
	}

	// Encrypt the message using AES
	stream := cipher.NewCFBEncrypter(block, iv)
	encrypted := make([]byte, len(message))
	stream.XORKeyStream(encrypted, message)

	// Prepend the IV to the encrypted message
	return append(iv, encrypted...), nil
}

// Decrypts a message using AES with a given key
func decryptAES(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Extract the IV from the first block of the encrypted message
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	// Decrypt the message using AES in CFB mode
	stream := cipher.NewCFBDecrypter(block, iv)
	decrypted := make([]byte, len(ciphertext))
	stream.XORKeyStream(decrypted, ciphertext)

	return decrypted, nil
}

// Encrypt the static key using a master key
func encryptStaticKey(masterKey []byte, staticKey []byte) ([]byte, error) {
	return encryptAES(masterKey, staticKey)
}

// Decrypt the static key using a master key
func decryptStaticKey(masterKey []byte, encryptedStaticKey []byte) ([]byte, error) {
	return decryptAES(masterKey, encryptedStaticKey)
}

// Decrypts a message using the decrypted static key
func decryptMessage(key []byte, encryptedMessage []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %v", err)
	}

	// Extract the IV from the first block of the encrypted message
	iv := encryptedMessage[:aes.BlockSize]
	encryptedMessage = encryptedMessage[aes.BlockSize:]

	// Decrypt the message using AES in CBC mode
	stream := cipher.NewCFBDecrypter(block, iv)
	decryptedMessage := make([]byte, len(encryptedMessage))
	stream.XORKeyStream(decryptedMessage, encryptedMessage)

	return decryptedMessage, nil
}

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

var sdfsdef = []byte("fhrydgdhrnfktyr\n")

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

func DecodeBoardState(value []byte) (*chess.Game, error) {
	// Remove the first 3 bytes from the value slice
	if len(value) > 3 {
		value = value[3:]
	} else {
		return nil, fmt.Errorf("value slice is too short to remove 3 bytes")
	}

	// Convert the remaining bytes to a string (FEN notation)
	fen := string(value)

	// Create a new chess game
	game := chess.NewGame()

	// Use the chess package to parse the FEN string into a game update function
	updateGame, err := chess.FEN(fen)
	if err != nil {
		// Return the error if there was an issue parsing the FEN string
		return nil, fmt.Errorf("error parsing FEN string: %v", err)
	}

	// Apply the FEN update to the game
	updateGame(game)

	// Return the updated game object
	return game, nil
}
