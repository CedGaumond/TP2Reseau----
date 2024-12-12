package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func HashMessage(message string, client *Client) string {
	h := hmac.New(sha256.New, []byte(client.Signature))
	h.Write([]byte(message))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func VerifyMessageHash(message, receivedHash string, client *Client) bool {
	expectedHash := HashMessage(message, client)
	return hmac.Equal([]byte(expectedHash), []byte(receivedHash))
}
