package main

import (
	"fmt"
	"net"
)

var kef = []byte("thisisaverysecretkeythatis32byteslong")

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

	// Encode the signature as TLV
	signature := []byte(client.Signature)
	signatureTLV, err := EncodeTLV(ByteData, signature)
	if err != nil {
		return nil, fmt.Errorf("error encoding signature: %v", err)
	}

	// Generate a hash of the entire message for integrity verification
	combinedData := append(lobbyRequestTLV, signatureTLV...)
	messageHash := GenerateSignature(combinedData)

	// Encode the hash as TLV
	hashTLV, err := EncodeTLV(ByteData, []byte(messageHash))
	if err != nil {
		return nil, fmt.Errorf("error encoding hash: %v", err)
	}

	// Combine the TLVs into the final message
	finalMessage := append(lobbyRequestTLV, signatureTLV...)
	finalMessage = append(finalMessage, hashTLV...)

	// Directly send the encoded TLV message to the server
	_, err = conn.Write(finalMessage)
	if err != nil {
		return nil, fmt.Errorf("error sending message: %v", err)
	}

	// Wait for the response from the server
	buf := make([]byte, 1024) // Adjust buffer size if needed
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("error reading from connection: %v", err)
	}

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

func SendJoinGameRequest(conn net.Conn, client *Client, playername string) error {
	// 1. First, send the JoinLobbyRequest Tag
	joinLobbyRequestTLV, err := EncodeTLV(JoinLobbyRequest, []byte{})
	if err != nil {
		return fmt.Errorf("error encoding JoinLobbyRequest: %v", err)
	}

	// Send the JoinLobbyRequest TLV
	_, err = conn.Write(joinLobbyRequestTLV)
	if err != nil {
		return fmt.Errorf("error sending JoinLobbyRequest: %v", err)
	}

	// 2. Prepare and send the PlayerName TLV
	playerNameData := []byte(playername)
	playerNameTLV, err := EncodeTLV(ByteData, playerNameData)
	if err != nil {
		return fmt.Errorf("error encoding player name: %v", err)
	}

	// Send the PlayerName TLV
	_, err = conn.Write(playerNameTLV)
	if err != nil {
		return fmt.Errorf("error sending player name: %v", err)
	}

	// Wait for the response
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("error reading server response: %v", err)
	}

	// Decode the response to get the GameID
	responseTag, _, err := DecodeTLV(buf[:n])
	if err != nil {
		return fmt.Errorf("error decoding server response: %v", err)
	}

	// Verify the response tag
	if responseTag != ByteData {
		return fmt.Errorf("unexpected response tag: %d", responseTag)
	}

	return nil
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

	// 2. Player Name TLV
	playerName := []byte(client.FirstName) // Ensure we are sending the correct player name
	playerNameTLV, err := EncodeTLV(ByteData, playerName)
	if err != nil {
		return fmt.Errorf("error encoding player name: %v", err)
	}

	// 3. Signature TLV
	signature := []byte(client.Signature) // Signature of the client
	signatureTLV, err := EncodeTLV(ByteData, signature)
	if err != nil {
		return fmt.Errorf("error encoding signature: %v", err)
	}

	// Prepare combined data for hash calculation (excluding the hash TLV)
	combinedData := append(gameRequestTLV, playerNameTLV...) // Combine GameRequest and PlayerName TLVs
	combinedData = append(combinedData, signatureTLV...)     // Add Signature TLV

	// 4. Hash TLV
	messageHash := GenerateSignature(combinedData) // Generate hash over the combined data

	hashTLV, err := EncodeTLV(ByteData, []byte(messageHash))
	if err != nil {
		return fmt.Errorf("error encoding hash: %v", err)
	}

	// Combine all TLVs in the correct order
	finalMessage := append(gameRequestTLV, playerNameTLV...)
	finalMessage = append(finalMessage, signatureTLV...)
	finalMessage = append(finalMessage, hashTLV...) // Final combined message

	// Send the encoded TLV message to the client/server
	_, err = conn.Write(finalMessage)
	if err != nil {
		return fmt.Errorf("error sending message: %v", err)
	}

	return nil
}

// SendBoardRequest sends a BoardRequest and signature TLVs to the server
func SendBoardRequest(conn net.Conn, gameID string, signature []byte) error {
	// Create the BoardRequest TLV
	boardRequestData := []byte(gameID) // or any other relevant data
	boardRequestTLV, err := EncodeTLV(BoardRequest, boardRequestData)
	if err != nil {
		return fmt.Errorf("error encoding BoardRequest TLV: %v", err)
	}

	// Send the BoardRequest TLV
	_, err = conn.Write(boardRequestTLV)
	if err != nil {
		return fmt.Errorf("error sending BoardRequest TLV: %v", err)
	}

	// Create the Signature TLV
	signatureTLV, err := EncodeTLV(ByteData, signature)
	if err != nil {
		return fmt.Errorf("error encoding Signature TLV: %v", err)
	}

	// Send the Signature TLV
	_, err = conn.Write(signatureTLV)
	if err != nil {
		return fmt.Errorf("error sending Signature TLV: %v", err)
	}
	return nil
}

func SendMoveRequest(conn net.Conn, client *Client, move string) error {
	if conn == nil {
		return fmt.Errorf("connection is nil")
	}

	// 1. ActionRequest TLV (move request)
	actionRequestData := []byte(move) // The move made by the player
	actionRequestTLV, err := EncodeTLV(ActionRequest, actionRequestData)
	if err != nil {
		return fmt.Errorf("error encoding ActionRequest: %v", err)
	}

	gameIDStr := GlobalGame.gameId.String()
	gameIDData := []byte(gameIDStr)
	gameIDTLV, err := EncodeTLV(ByteData, gameIDData)
	if err != nil {
		return fmt.Errorf("error encoding GameID: %v", err)
	}

	playerNameData := []byte(client.FirstName)
	playerNameTLV, err := EncodeTLV(ByteData, playerNameData)
	if err != nil {
		return fmt.Errorf("error encoding player name: %v", err)
	}

	// 4. Signature TLV (client signature for integrity)
	signatureData := []byte(client.Signature)
	signatureTLV, err := EncodeTLV(ByteData, signatureData)
	if err != nil {
		return fmt.Errorf("error encoding signature: %v", err)
	}

	// Prepare combined data for hash calculation (excluding the hash TLV)
	combinedData := append(actionRequestTLV, gameIDTLV...)
	combinedData = append(combinedData, playerNameTLV...)
	combinedData = append(combinedData, signatureTLV...)

	// 5. Hash TLV (for message integrity)
	messageHash := GenerateSignature(combinedData) // Generate hash over the combined data

	hashTLV, err := EncodeTLV(ByteData, []byte(messageHash))
	if err != nil {
		return fmt.Errorf("error encoding hash: %v", err)
	}

	// Combine all TLVs in the correct order
	finalMessage := append(actionRequestTLV, gameIDTLV...)
	finalMessage = append(finalMessage, playerNameTLV...)
	finalMessage = append(finalMessage, signatureTLV...)
	finalMessage = append(finalMessage, hashTLV...)

	// Encrypt the entire message
	encryptedMessage, err := EncryptMessage(finalMessage)
	if err != nil {
		return fmt.Errorf("error encrypting message: %v", err)
	}

	// Send the encrypted message to the server
	_, err = conn.Write(encryptedMessage)
	if err != nil {
		return fmt.Errorf("error sending encrypted message: %v", err)
	}

	return nil
}
