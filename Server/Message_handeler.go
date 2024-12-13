package main

import (
	"fmt"
	"github.com/google/uuid"
	"log"
	"net"
	"strconv"
)

func HandleHelloRequest(conn net.Conn, udpConn *net.UDPConn, clientAddr *net.UDPAddr, data []byte, isTCP bool) error {
	log.Println("Entered HandleHelloRequest")

	var currentIndex int
	var combinedTLV []byte

	// Decode the HelloRequest TLV (Tag=0)
	tag, value, err := DecodeTLV(data[currentIndex:])
	if err != nil {
		log.Printf("Error decoding HelloRequest TLV: %v", err)
		return fmt.Errorf("error decoding HelloRequest: %w", err)
	}
	currentIndex += len(value) + 3
	log.Printf("Decoded TLV: Tag=%d, Value=%s", tag, string(value))

	// Check if it's the HelloRequest (Tag=0)
	if tag != HelloRequest {
		log.Printf("Unexpected tag received: %d", tag)
		return fmt.Errorf("expected HelloRequest, but got tag %d", tag)
	}

	// Combine TLV data for signature validation
	combinedTLV = append(combinedTLV, data[:currentIndex]...)

	// Decode additional client data (e.g., FirstName, LastName, Status, Level)
	var firstName, lastName, status string
	var level int
	tags := []string{"FirstName", "LastName", "Status", "Level"}
	values := []*string{&firstName, &lastName, &status}

	for i, field := range tags {
		tag, value, err = DecodeTLV(data[currentIndex:])
		if err != nil {
			log.Printf("Error decoding %s TLV: %v", field, err)
			return fmt.Errorf("error decoding %s: %w", field, err)
		}
		currentIndex += len(value) + 3
		log.Printf("Decoded TLV: Tag=%d, Value=%s", tag, string(value))
		combinedTLV = append(combinedTLV, data[currentIndex-len(value)-3:currentIndex]...)

		if i < len(values) {
			*values[i] = string(value)
		} else {
			level, err = strconv.Atoi(string(value))
			if err != nil {
				log.Printf("Error converting Level to integer: %v", err)
				return fmt.Errorf("error converting Level: %w", err)
			}
		}
	}

	// Decode the Signature TLV (Tag for signature data)
	tag, value, err = DecodeTLV(data[currentIndex:])
	if err != nil {
		log.Printf("Error decoding Signature TLV: %v", err)
		return fmt.Errorf("error decoding Signature: %w", err)
	}
	currentIndex += len(value) + 3
	receivedSignature := string(value)

	tag, value, err = DecodeTLV(data[currentIndex:])
	if err != nil {
		log.Printf("Error decoding Hash TLV: %v", err)
		return fmt.Errorf("error decoding Hash: %w", err)
	}
	currentIndex += len(value) + 3
	receivedHash := string(value)

	// Compute the signature from the received message (excluding the signature and hash TLVs)
	computedSignature := GenerateSignature(combinedTLV)
	log.Printf("Computed Signature: %s", computedSignature)
	log.Printf("Received Signature: %s", receivedHash)

	// Verify that the computed signature matches the received signature
	if computedSignature != receivedHash {
		log.Printf("Signature mismatch: Computed=%s, Received=%s", computedSignature, receivedSignature)
		return fmt.Errorf("signature mismatch")
	}

	// Verify that the computed hash matches the received hash
	computedHash := GenerateSignature(combinedTLV) // You can reuse the signature function to compute the message hash
	log.Printf("Computed Hash: %s", computedHash)
	log.Printf("Received Hash: %s", receivedHash)

	if computedHash != receivedHash {
		log.Printf("Hash mismatch: Computed=%s, Received=%s", computedHash, receivedHash)
		return fmt.Errorf("hash mismatch")
	}

	log.Println("Signature and Hash verified successfully.")

	// Save client information
	clientKey := ""
	if isTCP && conn != nil {
		clientKey = conn.RemoteAddr().String()
	} else if clientAddr != nil {
		clientKey = clientAddr.String()
	}

	client := Client{
		FirstName: firstName,
		LastName:  lastName,
		Status:    status,
		Level:     level,
		Signature: receivedSignature,
		Address:   clientKey,
	}

	// Add client to the client list (store the client)
	clientList.AddClient(clientKey, client)
	log.Printf("Client saved successfully: Key=%s, Client=%+v", clientKey, client)

	// Send a response back to the client
	if isTCP {
		return SendHelloResponseTCP(conn, computedSignature)
	} else if udpConn != nil && clientAddr != nil {
		return SendHelloResponseUDP(udpConn, clientAddr, computedSignature)
	}
	return fmt.Errorf("invalid connection type")
}

func HandleGameRequest(conn net.Conn, udpConn *net.UDPConn, clientAddr *net.UDPAddr, data []byte, isTCP bool) error {
	log.Println("Entered HandleGameRequest")

	var currentIndex int
	// Decode the first TLV: GameRequest (RequestType)
	tag, requestType, err := DecodeTLV(data[currentIndex:])
	if err != nil {
		log.Printf("Error decoding GameRequest TLV: %v", err)
		return fmt.Errorf("error decoding GameRequest: %w", err)
	}
	currentIndex += len(requestType) + 3

	// Decode the second TLV: Player Name
	tag, playerName, err := DecodeTLV(data[currentIndex:])
	if err != nil {
		log.Printf("Error decoding player name TLV: %v", err)
		return fmt.Errorf("error decoding player name: %w", err)
	}
	currentIndex += len(playerName) + 3
	log.Printf("Decoded TLV: Tag=%d, Player Name=%s", tag, string(playerName))

	// Check if the tag matches the expected tag for Player Name
	if tag != ByteData {
		log.Printf("Unexpected tag for player name: %d", tag)
		return fmt.Errorf("expected ByteData tag for player name, but got tag %d", tag)
	}

	// Decode the third TLV: Signature
	tag, signature, err := DecodeTLV(data[currentIndex:])
	if err != nil {
		log.Printf("Error decoding Signature TLV: %v", err)
		return fmt.Errorf("error decoding Signature: %w", err)
	}
	currentIndex += len(signature) + 3
	log.Printf("Decoded TLV: Tag=%d, Value=%s", tag, string(signature))

	if tag != ByteData {
		log.Printf("Unexpected tag for signature: %d", tag)
		return fmt.Errorf("expected ByteData tag for signature, but got tag %d", tag)
	}

	// Decode the fourth TLV: Hash
	// Check if there is enough data left to decode the hash
	if len(data[currentIndex:]) < 3 {
		log.Println("Not enough data left to decode Hash TLV")
		return fmt.Errorf("not enough data to decode Hash TLV")
	}

	tag, providedHash, err := DecodeTLV(data[currentIndex:])
	if err != nil {
		log.Printf("Error decoding Hash TLV: %v", err)
		return fmt.Errorf("error decoding Hash: %w", err)
	}
	currentIndex += len(providedHash) + 3
	log.Printf("Decoded TLV: Tag=%d, Value=%s", tag, string(providedHash))

	if tag != ByteData {
		log.Printf("Unexpected tag for hash: %d", tag)
		return fmt.Errorf("expected ByteData tag for hash, but got tag %d", tag)
	}

	// Verify the integrity of the message by checking the hash
	combinedData := data[:currentIndex-len(providedHash)-3]
	calculatedHash := GenerateSignature(combinedData)
	log.Printf("Calculated hash: %s", calculatedHash)
	log.Printf("Provided hash: %s", string(providedHash))

	// Compare the calculated hash with the provided hash
	if string(providedHash) != calculatedHash {
		log.Println("Hash mismatch: Provided hash does not match calculated hash")
		return fmt.Errorf("hash mismatch")
	}

	// Determine the client address
	var clientAddress string
	if isTCP {
		clientAddress = conn.RemoteAddr().String()
	} else if clientAddr != nil {
		clientAddress = clientAddr.String()
	} else {
		log.Println("Error: Unable to determine client address")
		return fmt.Errorf("client address is missing")
	}

	// Fetch the client from ClientList
	client, exists := clientList.GetClient(clientAddress)
	if !exists {
		log.Printf("Client with address %s not found", clientAddress)
		return fmt.Errorf("client not found")
	}

	// Validate the signature
	if string(signature) != client.Signature {
		log.Printf("Signature mismatch:\nProvided: %s\nStored: %s", string(signature), client.Signature)
		return fmt.Errorf("signature mismatch")
	}

	log.Println("Signature validated successfully")

	// Create a new game session with the player's name as the creator
	gameID := createNewGame(string(playerName), fmt.Sprintf("Lobby-%s", string(playerName)))
	if gameID == uuid.Nil {
		log.Println("Failed to create a new game session. Lobby might already exist.")
		return fmt.Errorf("failed to create new game session")
	}

	// Set the GameID for the client in ClientList
	err = clientList.SetClientGameID(clientAddress, gameID)
	if err != nil {
		log.Printf("Error setting GameID for client: %v", err)
		return fmt.Errorf("error setting GameID for client: %w", err)
	}

	// Convert the UUID to a byte array
	uuidBytes, err := gameID.MarshalBinary()
	if err != nil {
		log.Printf("Error marshaling UUID to bytes: %v", err)
		return fmt.Errorf("error marshaling UUID: %w", err)
	}

	// Encode the GameResponse TLV with the game UUID
	gameUUID, err := EncodeTLV(UUIDPartie, uuidBytes)
	if err != nil {
		log.Printf("Error encoding GameResponse TLV: %v", err)
		return fmt.Errorf("error encoding GameResponse: %w", err)
	}

	// Send the GameResponse back to the client
	if isTCP {
		if err := SendMessageTCP(conn, UUIDPartie, gameUUID); err != nil {
			log.Printf("Error sending GameResponse over TCP: %v", err)
			return err
		}
		log.Println("GameResponse sent over TCP.")
	} else if udpConn != nil && clientAddr != nil {
		// Append the UUIDPartie TLV to the response
		response := gameUUID
		if err := SendMessageUDP(udpConn, clientAddr, UUIDPartie, response); err != nil {
			log.Printf("Error sending GameResponse over UDP: %v", err)
			return err
		}
		log.Println("GameResponse sent over UDP.")
	} else {
		log.Println("Invalid connection type, cannot send response.")
		return fmt.Errorf("invalid connection type")
	}

	// Log the creator (player's name) for the created game session
	log.Printf("Created new game session for player: %s, Game ID: %s", string(playerName), gameID.String())

	// Ensure the player's name is printed
	log.Printf("The player's name (creator): %s", string(playerName))

	return nil
}

func HandleLobbyListRequest(conn net.Conn, udpConn *net.UDPConn, clientAddr *net.UDPAddr, data []byte, isTCP bool) error {
	log.Println("Entered HandleLobbyListRequest")

	var currentIndex int

	// Decode the first TLV: LobbyRequest (tag 169)
	tag, lobbyData, err := DecodeTLV(data[currentIndex:])
	if err != nil {
		log.Printf("Error decoding LobbyRequest TLV: %v", err)
		return fmt.Errorf("error decoding LobbyRequest: %w", err)
	}
	currentIndex += len(lobbyData) + 3
	log.Printf("Decoded TLV: Tag=%d, Value=%s", tag, string(lobbyData))

	if tag != LobbyRequest {
		log.Printf("Unexpected tag for LobbyRequest: %d", tag)
		return fmt.Errorf("expected LobbyRequest TLV, but got tag %d", tag)
	}

	// Proceed to decode the second TLV: Signature (tag 3)
	tag, signature, err := DecodeTLV(data[currentIndex:])
	if err != nil {
		log.Printf("Error decoding Signature TLV: %v", err)
		return fmt.Errorf("error decoding Signature: %w", err)
	}
	currentIndex += len(signature) + 3
	log.Printf("Decoded TLV: Tag=%d, Value=%s", tag, string(signature))

	if tag != ByteData { // The signature is encoded with the ByteData tag
		log.Printf("Unexpected tag for signature: %d", tag)
		return fmt.Errorf("expected ByteData for signature, but got tag %d", tag)
	}

	// Decode the hash (if needed)
	tag, providedHash, err := DecodeTLV(data[currentIndex:])
	if err != nil {
		log.Printf("Error decoding Hash TLV: %v", err)
		return fmt.Errorf("error decoding Hash: %w", err)
	}
	currentIndex += len(providedHash) + 3
	log.Printf("Decoded TLV: Tag=%d, Value=%s", tag, string(providedHash))

	if tag != ByteData { // The hash is also encoded with the ByteData tag
		log.Printf("Unexpected tag for hash: %d", tag)
		return fmt.Errorf("expected ByteData for hash, but got tag %d", tag)
	}

	// Prepare combined data for hash verification (exclude the hash itself)
	combinedData := data[:currentIndex-len(providedHash)-3]
	calculatedHash := GenerateSignature(combinedData)
	log.Printf("Calculated hash: %s", calculatedHash)
	log.Printf("Provided hash: %s", string(providedHash))

	// Compare the calculated hash with the provided hash
	if string(providedHash) != calculatedHash {
		log.Println("Hash mismatch: Provided hash does not match calculated hash")
		return fmt.Errorf("hash mismatch")
	}

	// Determine the client address
	var clientAddress string
	if isTCP {
		clientAddress = conn.RemoteAddr().String()
	} else if clientAddr != nil {
		clientAddress = clientAddr.String()
	} else {
		log.Println("Error: Unable to determine client address")
		return fmt.Errorf("client address is missing")
	}

	// Fetch the client from ClientList
	client, exists := clientList.GetClient(clientAddress)
	if !exists {
		log.Printf("Client with address %s not found", clientAddress)
		return fmt.Errorf("client not found")
	}

	// Validate the signature
	if string(signature) != client.Signature {
		log.Printf("Signature mismatch:\nProvided: %s\nStored: %s", string(signature), client.Signature)
		return fmt.Errorf("signature mismatch")
	}

	log.Println("Signature validated successfully")

	// Get the list of available lobbies
	gameMutex.RLock()
	var encodedLobbies []byte
	for lobbyName, gameID := range LobbyNameToUUID {
		session := GameStore[gameID]
		if !session.IsLocked { // Include only unlocked lobbies
			// Print the lobby name and its creator
			log.Printf("Lobby: %s, Creator: %s", lobbyName, session.CreatorName)

			// Encode each lobby name as a TLV
			lobbyData, err := EncodeTLV(String, []byte(lobbyName))
			if err != nil {
				log.Printf("Error encoding lobby name %s: %v", lobbyName, err)
				gameMutex.RUnlock()
				return fmt.Errorf("error encoding lobby name: %w", err)
			}
			encodedLobbies = append(encodedLobbies, lobbyData...)
		}
	}
	gameMutex.RUnlock()

	// Encode the full lobby response TLV
	responseTLV, err := EncodeTLV(lobbyResponse, encodedLobbies)
	if err != nil {
		log.Printf("Error encoding LobbyResponse TLV: %v", err)
		return fmt.Errorf("error encoding LobbyResponse: %w", err)
	}

	// Send the LobbyList response back to the client
	if isTCP {
		if err := SendMessageTCP(conn, lobbyResponse, responseTLV); err != nil {
			log.Printf("Error sending LobbyList over TCP: %v", err)
			return err
		}
		log.Println("LobbyList sent over TCP.")
	} else if udpConn != nil && clientAddr != nil {
		if err := SendMessageUDP(udpConn, clientAddr, lobbyResponse, responseTLV); err != nil {
			log.Printf("Error sending LobbyList over UDP: %v", err)
			return err
		}
		log.Println("LobbyList sent over UDP.")
	} else {
		log.Println("Invalid connection type, cannot send response.")
		return fmt.Errorf("invalid connection type")
	}

	return nil
}

var ddees = []byte("sldfkjasldkfjapwoi3")

func HandleBoardRequest(conn net.Conn, udpConn *net.UDPConn, clientAddr *net.UDPAddr, data []byte, isTCP bool) error {
	log.Println("Entered HandleBoardRequest")

	var currentIndex int

	// Decode the first TLV: BoardRequest (tag 50)
	tag, boardRequestData, err := DecodeTLV(data[currentIndex:])
	if err != nil {
		log.Printf("Error decoding BoardRequest TLV: %v", err)
		return fmt.Errorf("error decoding BoardRequest: %w", err)
	}
	currentIndex += len(boardRequestData) + 3
	log.Printf("Decoded TLV: Tag=%d, Value=%s", tag, string(boardRequestData))

	if tag != BoardRequest {
		log.Printf("Unexpected tag for BoardRequest: %d", tag)
		return fmt.Errorf("expected BoardRequest TLV, but got tag %d", tag)
	}

	// Decode the second TLV: Signature (tag 3)
	tag, signature, err := DecodeTLV(data[currentIndex:])
	if err != nil {
		log.Printf("Error decoding Signature TLV: %v", err)
		return fmt.Errorf("error decoding Signature: %w", err)
	}
	currentIndex += len(signature) + 3
	log.Printf("Decoded TLV: Tag=%d, Value=%s", tag, string(signature))

	if tag != ByteData {
		log.Printf("Unexpected tag for Signature: %d", tag)
		return fmt.Errorf("expected ByteData for Signature, but got tag %d")
	}

	// Determine the client address
	var clientAddress string
	if isTCP {
		clientAddress = conn.RemoteAddr().String()
	} else if clientAddr != nil {
		clientAddress = clientAddr.String()
	} else {
		log.Println("Error: Unable to determine client address")
		return fmt.Errorf("client address is missing")
	}

	// Fetch the client
	client, exists := clientList.GetClient(clientAddress)
	if !exists {
		log.Printf("Client with address %s not found", clientAddress)
		return fmt.Errorf("client not found")
	}

	// Fetch the game session
	gameMutex.RLock()
	session, ok := GameStore[client.GameID]
	gameMutex.RUnlock()

	if !ok {
		log.Printf("No game session found for GameID: %s", client.GameID)
		return fmt.Errorf("game session not found")
	}

	// Get the board state
	boardState := session.GetBoardState()
	if boardState == "" {
		log.Printf("No valid board state for GameID: %s", client.GameID)
		return fmt.Errorf("invalid board state")
	}

	// Encode the board state as a TLV
	boardResponseTLV, err := EncodeTLV(BoardResponse, []byte(boardState))
	if err != nil {
		log.Printf("Error encoding BoardResponse TLV: %v", err)
		return fmt.Errorf("error encoding BoardResponse: %w", err)
	}

	// Send the board state to the client
	if isTCP {
		if err := SendMessageTCP(conn, BoardResponse, boardResponseTLV); err != nil {
			log.Printf("Error sending BoardResponse over TCP: %v", err)
			return err
		}
		log.Println("BoardResponse sent over TCP.")
	} else if udpConn != nil && clientAddr != nil {
		if err := SendMessageUDP(udpConn, clientAddr, BoardResponse, boardResponseTLV); err != nil {
			log.Printf("Error sending BoardResponse over UDP: %v", err)
			return err
		}
		log.Println("BoardResponse sent over UDP.")
	} else {
		log.Println("Invalid connection type, cannot send response.")
		return fmt.Errorf("invalid connection type")
	}

	return nil
}

func HandleMoveRequest(conn net.Conn, udpConn *net.UDPConn, clientAddr *net.UDPAddr, data []byte, isTCP bool) error {
	log.Println("Entered HandleMoveRequest")

	var currentIndex int
	var moveNotation, gameIDStr, playerName, signature string

	// First TLV should be ActionRequest (move)
	tag, moveRequestData, err := DecodeTLV(data[currentIndex:])
	if err != nil {
		log.Printf("Error decoding first TLV (ActionRequest): %v", err)
		return fmt.Errorf("error decoding first TLV: %w", err)
	}
	if tag != ActionRequest {
		log.Printf("Unexpected first tag: %d", tag)
		return fmt.Errorf("expected ActionRequest, got tag %d", tag)
	}
	moveNotation = string(moveRequestData)
	currentIndex += len(moveRequestData) + 3
	log.Printf("Move Notation: %s", moveNotation)

	// Second TLV should be GameID (ByteData)
	tag, gameIDData, err := DecodeTLV(data[currentIndex:])
	if err != nil {
		log.Printf("Error decoding GameID TLV: %v", err)
		return fmt.Errorf("error decoding GameID TLV: %w", err)
	}
	if tag != ByteData {
		log.Printf("Unexpected second tag: %d", tag)
		return fmt.Errorf("expected ByteData for GameID, got tag %d", tag)
	}
	gameIDStr = string(gameIDData)
	currentIndex += len(gameIDData) + 3
	log.Printf("Game ID: %s", gameIDStr)

	// Third TLV should be PlayerName (ByteData)
	tag, playerNameData, err := DecodeTLV(data[currentIndex:])
	if err != nil {
		log.Printf("Error decoding PlayerName TLV: %v", err)
		return fmt.Errorf("error decoding PlayerName TLV: %w", err)
	}
	if tag != ByteData {
		log.Printf("Unexpected third tag: %d", tag)
		return fmt.Errorf("expected ByteData for PlayerName, got tag %d", tag)
	}
	playerName = string(playerNameData)
	currentIndex += len(playerNameData) + 3
	log.Printf("Player Name: %s", playerName)

	// Optional: Parse Signature and Hash TLVs if needed
	// For now, we'll just log them
	tag, signatureData, err := DecodeTLV(data[currentIndex:])
	if err == nil && tag == ByteData {
		signature = string(signatureData)
		log.Printf("Signature: %s", signature)
	}

	// Parse the game ID
	gameID, err := uuid.Parse(gameIDStr)
	if err != nil {
		return fmt.Errorf("invalid game ID: %v", err)
	}

	// Fetch the game session
	gameMutex.Lock()
	session, ok := GameStore[gameID]
	gameMutex.Unlock()

	if !ok {
		log.Printf("No game session found for GameID: %s", gameID)
		return fmt.Errorf("game session not found")
	}

	// Attempt to move the piece
	err = Move(session.Game, moveNotation)

	// Get the current board state (whether the move was successful or not)
	boardState := session.GetBoardState()

	// Prepare the response: if the move failed, just send the current board state unchanged
	moveResponseData := boardState

	// Encode the response with the board state
	moveResponseTLV, err := EncodeTLV(ActionResponse, []byte(moveResponseData))
	if err != nil {
		log.Printf("Error encoding MoveResponse TLV: %v", err)
		return fmt.Errorf("error encoding MoveResponse: %w", err)
	}

	// Send the updated board state (unchanged if the move failed)
	if isTCP {
		if err := SendMessageTCP(conn, ActionResponse, moveResponseTLV); err != nil {
			log.Printf("Error sending MoveResponse over TCP: %v", err)
			return err
		}
		log.Println("MoveResponse sent over TCP.")
	} else if udpConn != nil && clientAddr != nil {
		if err := SendMessageUDP(udpConn, clientAddr, ActionResponse, moveResponseTLV); err != nil {
			log.Printf("Error sending MoveResponse over UDP: %v", err)
			return err
		}
		log.Println("MoveResponse sent over UDP.")
	} else {
		log.Println("Invalid connection type, cannot send response.")
		return fmt.Errorf("invalid connection type")
	}

	return nil
}
func HandleJoinRequest(conn net.Conn, udpConn *net.UDPConn, clientAddr *net.UDPAddr, data []byte, isTCP bool) error {
	log.Println("Entered HandleJoinRequest")

	var currentIndex int
	var playerName string

	// First TLV should be ActionRequest (move)
	tag, _, err := DecodeTLV(data[currentIndex:])
	if err != nil {
		log.Printf("Error decoding first TLV (ActionRequest): %v", err)
		return fmt.Errorf("error decoding first TLV: %w", err)
	}
	if tag != JoinLobbyRequest {
		log.Printf("Unexpected first tag: %d", tag)
		return fmt.Errorf("expected ActionRequest, got tag %d", tag)
	}

	// First TLV should be PlayerName (ByteData)
	tag, playerNameData, err := DecodeTLV(data[currentIndex:])
	if err != nil {
		log.Printf("Error decoding PlayerName TLV: %v", err)
		return fmt.Errorf("error decoding PlayerName TLV: %w", err)
	}

	playerName = string(playerNameData)
	log.Printf("Player Name: %s", playerName)

	// Find clients with matching name
	matchedClients := clientList.GetClientByName(playerName)

	var gameIDStr string
	var gameID uuid.UUID

	// If exactly one client found, use its game ID
	if len(matchedClients) == 1 {
		gameID = matchedClients[0].GameID
		gameIDStr = gameID.String()
		log.Printf("Found game ID for player %s: %s", playerName, gameIDStr)
	} else if len(matchedClients) > 1 {
		log.Printf("Multiple clients found with name %s", playerName)
		// Could implement additional logic to disambiguate if needed
		// For now, we'll use the first matched client's game ID
		gameID = matchedClients[0].GameID
		gameIDStr = gameID.String()
	} else {
		// No client found, generate a new game ID
		gameID = uuid.New()
		gameIDStr = gameID.String()
		log.Printf("No existing client found, generated new game ID: %s", gameIDStr)
	}

	// Prepare the response with the GameID
	responseData := []byte(gameIDStr)

	// Encode the GameID as the response TLV
	responseTLV, err := EncodeTLV(ByteData, responseData)
	if err != nil {
		log.Printf("Error encoding GameID TLV: %v", err)
		return fmt.Errorf("error encoding GameID TLV: %w", err)
	}

	// Send the response back to the client
	if isTCP {
		if err := SendMessageTCP(conn, JoinLobbyRequest, responseTLV); err != nil {
			log.Printf("Error sending response over TCP: %v", err)
			return err
		}
		log.Println("Response sent over TCP.")
	} else if udpConn != nil && clientAddr != nil {
		if err := SendMessageUDP(udpConn, clientAddr, JoinLobbyRequest, responseTLV); err != nil {
			log.Printf("Error sending response over UDP: %v", err)
			return err
		}
		log.Println("Response sent over UDP.")
	} else {
		log.Println("Invalid connection type, cannot send response.")
		return fmt.Errorf("invalid connection type")
	}

	return nil
}

// SendHelloResponseTCP sends a HelloResponse (Tag 101) to the TCP client with the signature
func SendHelloResponseTCP(conn net.Conn, signature string) error {
	// Send the HelloResponse (Tag 101) to the TCP client
	return SendMessageTCP(conn, HelloResponse, []byte(signature))
}

// SendHelloResponseUDP sends a HelloResponse (Tag 101) to the UDP client with the signature
func SendHelloResponseUDP(conn *net.UDPConn, clientAddr *net.UDPAddr, signature string) error {
	// Send the HelloResponse (Tag 101) to the UDP client
	return SendMessageUDP(conn, clientAddr, HelloResponse, []byte(signature))
}

// SendMessageTCP sends a message with a specified tag to the TCP connection
func SendMessageTCP(conn net.Conn, tag Tag, message []byte) error {
	// Encode the message with the specified tag
	encodedMessage, err := EncodeTLV(tag, message)
	if err != nil {
		return fmt.Errorf("error encoding message with tag %d: %w", tag, err)
	}

	// Send the encoded message to the TCP client
	_, err = conn.Write(encodedMessage)
	if err != nil {
		return fmt.Errorf("error sending message with tag %d: %w", tag, err)
	}

	log.Printf("Message sent with tag %d to TCP client: %s\n", tag, string(message))
	return nil
}

// SendMessageUDP sends a message with a specified tag to the UDP client
func SendMessageUDP(conn *net.UDPConn, clientAddr *net.UDPAddr, tag Tag, message []byte) error {
	// Encode the message with the specified tag
	encodedMessage, err := EncodeTLV(tag, message)
	if err != nil {
		return fmt.Errorf("error encoding message with tag %d: %w", tag, err)
	}

	// Send the encoded message to the UDP client at the specified address
	_, err = conn.WriteToUDP(encodedMessage, clientAddr)
	if err != nil {
		return fmt.Errorf("error sending message with tag %d to %s: %w", tag, clientAddr, err)
	}

	log.Printf("Message sent with tag %d to %s: %s\n", tag, clientAddr, string(message))
	return nil
}
