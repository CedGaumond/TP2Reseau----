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

// HandleUnknownTag handles unknown tags and logs an error
func HandleUnknownTag(tag Tag) {
	log.Printf("Received unknown tag: %s\n", GetTagName(tag))
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
