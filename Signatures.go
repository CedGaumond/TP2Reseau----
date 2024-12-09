package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

const Pepper = "Patate" // Secret pepper for hashing

// GenerateSalt generates a random salt based on the current Unix time
func GenerateSalt() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// HashMessage generates a SHA-256 hash from a message with a salt and pepper
func HashMessage(message, salt string) string {
	combined := message + salt + Pepper
	hash := sha256.New()
	hash.Write([]byte(combined))
	return hex.EncodeToString(hash.Sum(nil))
}

// GenerateServerSignature generates a unique signature for the server based on the client's information
func GenerateServerSignature(client Client, salt string) string {
	data := client.Nom + client.Prenom + client.Statut + fmt.Sprintf("%d", client.Niveau)
	return HashMessage(data, salt)
}

// EncryptData encrypts the given data using AES encryption and the provided key
func EncryptData(data, key string) (string, error) {
	// Derive a 32-byte key from the client's signature using SHA-256
	keyHash := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return "", err
	}

	// Prepare the data to be encrypted
	plaintext := []byte(data)
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	cipher.NewCFBEncrypter(block, iv).XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	// Return the encrypted data as a hex string
	return hex.EncodeToString(ciphertext), nil
}

// ProcessClientData processes the client's request and generates the hashes for the client and server signatures, encrypting the server signature
func ProcessClientData(client Client) (string, string, error) {
	// Generate the salt for both signatures
	salt := GenerateSalt()

	// Generate the client's signature hash
	clientSignatureHash := HashMessage(client.SignatureGivenByClient, salt)

	// Generate the server's signature
	serverSignature := GenerateServerSignature(client, salt)

	// Encrypt the server's signature using the client's given signature as the key
	encryptedServerSignature, err := EncryptData(serverSignature, client.SignatureGivenByClient)
	if err != nil {
		return "", "", err
	}

	return clientSignatureHash, encryptedServerSignature, nil
}
