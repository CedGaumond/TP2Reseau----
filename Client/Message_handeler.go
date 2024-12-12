package main

import (
	"fmt"
	"log"
	"net"
)

func SendLobbyListRequest(conn net.Conn, client *Client) ([]string, error) {
	// Ensure the connection is not nil
	if conn == nil {
		return nil, fmt.Errorf("connection is nil")
	}

	// Encode the LobbyRequest as TLV
	lobbyRequest := []byte("LobbyRequest")
	length := len(lobbyRequest)
	lobbyRequestTLV, err := EncodeTLV(LobbyRequest, append([]byte{byte(length)}, lobbyRequest...))
	if err != nil {
		return nil, fmt.Errorf("error encoding LobbyRequest: %v", err)
	}
	log.Printf("Encoded TLV: Tag=LobbyRequest, Length=%d, Value=%s", length, string(lobbyRequest))

	// Encode the signature as TLV
	signature := []byte(client.Signature)
	signatureTLV, err := EncodeTLV(ByteData, signature)
	if err != nil {
		return nil, fmt.Errorf("error encoding signature: %v", err)
	}
	log.Printf("Encoded TLV: Tag=ByteData, Length=%d, Value=%s", len(signature), string(signature))

	// Generate a hash of the entire message for integrity verification
	combinedData := append(lobbyRequestTLV, signatureTLV...)
	messageHash := GenerateSignature(combinedData)

	// Log the calculated hash in the required format as a string
	log.Printf("Calculated hash: %s", messageHash)

	// Encode the hash as TLV
	hashTLV, err := EncodeTLV(ByteData, []byte(messageHash))
	if err != nil {
		return nil, fmt.Errorf("error encoding hash: %v", err)
	}
	log.Printf("Encoded TLV: Tag=ByteData, Length=%d, Value=%s", len(messageHash), messageHash)

	// Combine the TLVs into the final message
	finalMessage := append(lobbyRequestTLV, signatureTLV...)
	finalMessage = append(finalMessage, hashTLV...)

	// Log the final message
	log.Printf("Final message (Raw): %v", finalMessage)

	// Directly send the encoded TLV message to the server
	_, err = conn.Write(finalMessage)
	if err != nil {
		return nil, fmt.Errorf("error sending message: %v", err)
	}
	log.Println("Lobby List Request sent successfully.")

	// Wait for the response from the server
	// Read the server's response (you may need to handle timeouts or errors here)
	buf := make([]byte, 1024) // Adjust buffer size if needed
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("error reading from connection: %v", err)
	}

	// Log the raw received data (for debugging purposes)
	log.Printf("Raw data received (%d bytes): %x", n, buf[:n])

	// Decode the TLV response
	_, value, err := DecodeTLV(buf[:n])
	if err != nil {
		return nil, fmt.Errorf("error decoding server response: %v", err)
	}

	// Assuming the value contains a comma-separated list of lobbies
	lobbyList := string(value)
	lobbies := []string{lobbyList} // You can further split this string if needed

	// Return the parsed lobby list
	return lobbies, nil
}

func SendGameRequest(conn net.Conn, client *Client) error {
	// Ensure the connection is not nil
	if conn == nil {
		return fmt.Errorf("connection is nil")
	}

	// 1. GameRequest TLV
	gameRequestBytes := []byte("GameRequest")
	gameRequestTLV, err := EncodeTLV(GameRequest, gameRequestBytes)
	if err != nil {
		return fmt.Errorf("error encoding GameRequest: %v", err)
	}
	log.Printf("Encoded TLV: Tag=GameRequest, Length=%d, Value=%s", len(gameRequestBytes), string(gameRequestBytes))

	// 2. Player Name TLV
	firstname := []byte(client.FirstName)
	playernameTLV, err := EncodeTLV(ByteData, firstname)
	if err != nil {
		return fmt.Errorf("error encoding player name: %v", err)
	}
	log.Printf("Encoded player name: %s", client.FirstName)

	// 3. Signature TLV
	signature := []byte(client.Signature)
	signatureTLV, err := EncodeTLV(ByteData, signature)
	if err != nil {
		return fmt.Errorf("error encoding signature: %v", err)
	}
	log.Printf("Encoded TLV: Tag=ByteData, Length=%d, Value=%s", len(signature), string(signature))

	// Prepare combined data for hash calculation (excluding the hash TLV)
	combinedData := append(gameRequestTLV, playernameTLV...)
	combinedData = append(combinedData, signatureTLV...)

	// 4. Hash TLV
	messageHash := GenerateSignature(combinedData)
	log.Printf("Calculated hash: %s", messageHash)

	hashTLV, err := EncodeTLV(ByteData, []byte(messageHash))
	if err != nil {
		return fmt.Errorf("error encoding hash: %v", err)
	}
	log.Printf("Encoded TLV: Tag=ByteData, Length=%d, Value=%s", len(messageHash), messageHash)

	// Combine all TLVs in the correct order
	finalMessage := append(gameRequestTLV, playernameTLV...)
	finalMessage = append(finalMessage, signatureTLV...)
	finalMessage = append(finalMessage, hashTLV...)

	// Log the final message
	log.Printf("Final message (Raw): %v", finalMessage)

	// Send the encoded TLV message to the client
	_, err = conn.Write(finalMessage)
	if err != nil {
		return fmt.Errorf("error sending message: %v", err)
	}
	log.Println("Message sent successfully.")
	return nil
}
