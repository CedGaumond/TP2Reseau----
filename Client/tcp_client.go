package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/notnil/chess"
	"log"
	"net"
	"sync"
)

// ContinuousTCPListener manages a persistent TCP connection
type ContinuousTCPListener struct {
	serverAddr string
	client     *Client
	conn       *net.TCPConn
	stopChan   chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex
}

// NewContinuousTCPListener creates a new ContinuousTCPListener instance
func NewContinuousTCPListener(serverAddr string, client *Client) *ContinuousTCPListener {
	return &ContinuousTCPListener{
		serverAddr: serverAddr,
		client:     client,
		stopChan:   make(chan struct{}),
	}
}

func (l *ContinuousTCPListener) Connect() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Resolve server's TCP address
	tcpAddr, err := net.ResolveTCPAddr("tcp", l.serverAddr)
	if err != nil {
		return fmt.Errorf("error resolving server address: %v", err)
	}

	// Establish TCP connection
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return fmt.Errorf("error connecting to server: %v", err)
	}

	l.conn = conn
	return nil
}

func (l *ContinuousTCPListener) SendInitialHello() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.conn == nil {
		return fmt.Errorf("no active connection")
	}

	// Prepare the HelloRequest TLV (Tag=0)
	helloRequestTLV, err := EncodeTLV(HelloRequest, []byte("HelloRequest"))
	if err != nil {
		return fmt.Errorf("error encoding HelloRequest: %v", err)
	}

	// Encode each part of the client data as TLV
	firstNameTLV, err := EncodeTLV(String, []byte(l.client.FirstName))
	if err != nil {
		return fmt.Errorf("error encoding First Name: %v", err)
	}

	lastNameTLV, err := EncodeTLV(String, []byte(l.client.LastName))
	if err != nil {
		return fmt.Errorf("error encoding Last Name: %v", err)
	}

	statusTLV, err := EncodeTLV(String, []byte(l.client.Status))
	if err != nil {
		return fmt.Errorf("error encoding Status: %v", err)
	}

	levelTLV, err := EncodeTLV(Int, []byte(fmt.Sprintf("%d", l.client.Level)))
	if err != nil {
		return fmt.Errorf("error encoding Level: %v", err)
	}

	// Combine all TLVs into a single byte slice
	var combinedTLV []byte
	combinedTLV = append(combinedTLV, helloRequestTLV...)
	combinedTLV = append(combinedTLV, firstNameTLV...)
	combinedTLV = append(combinedTLV, lastNameTLV...)
	combinedTLV = append(combinedTLV, statusTLV...)
	combinedTLV = append(combinedTLV, levelTLV...)

	// Generate a hash of the entire message for integrity verification (GenerateSignature)
	messageHash := GenerateSignature(combinedTLV)
	messageHash = messageHash

	// Randomly generate the signature for the client
	l.client.Signature = GenerateRandomSignature()

	// Print the generated signature and message hash for debugging
	fmt.Printf("Generated Random Signature: %s\n", l.client.Signature)
	fmt.Printf("Message Hash for Integrity: %s\n", messageHash)

	// Encode the signature as a TLV (using ByteData tag)
	signatureTLV, err := EncodeTLV(ByteData, []byte(l.client.Signature))
	if err != nil {
		return fmt.Errorf("error encoding signature: %v", err)
	}

	// Encode the hash as a TLV (using ByteData tag)
	hashTLV, err := EncodeTLV(ByteData, []byte(messageHash))
	if err != nil {
		return fmt.Errorf("error encoding hash: %v", err)
	}

	// Append the signature and hash TLVs to the combined message
	combinedTLV = append(combinedTLV, signatureTLV...)
	combinedTLV = append(combinedTLV, hashTLV...)

	// Send the final TLV message (including the signature and hash)
	_, err = l.conn.Write(combinedTLV)
	if err != nil {
		return fmt.Errorf("error sending combined TLV message: %v", err)
	}

	return nil
}

func (l *ContinuousTCPListener) Listen() {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		defer func() {
			if l.conn != nil {
				l.conn.Close()
			}
		}()

		buf := make([]byte, 1024)
		for {
			select {
			case <-l.stopChan:
				log.Println("TCP listener stopped.")
				return
			default:
				// Read from the connection indefinitely
				n, err := l.conn.Read(buf)
				if err != nil {
					// Handle connection errors
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						// Timeout error, ignore and continue listening
						continue
					}

					// Handle disconnection
					if err.Error() == "EOF" {
						log.Println("Server closed the connection.")
						return
					}

					log.Printf("Error reading from server: %v", err)
					return
				}

				// Log raw data for debugging

				// Decode the TLV message
				tag, value, err := DecodeTLV(buf[:n])
				if err != nil {
					log.Printf("Error decoding server response: %v", err)
					continue
				}

				switch tag {
				case lobbyResponse:

				case UUIDPartie:
					// Process UUIDPartie (existing functionality)
					if len(value) < 16 {
						log.Printf("Insufficient data for UUID: %d bytes", len(value))
						continue
					}

					uuidBytes := value[len(value)-16:]
					var uuidValue uuid.UUID
					err = uuidValue.UnmarshalBinary(uuidBytes)
					if err != nil {
						log.Printf("Error unmarshaling UUID: %v", err)
						continue
					}
					SetGlobalGameID(uuidValue)

				case BoardResponse:

					// Decode the board state from the received value (FEN string)
					game, err := DecodeBoardState(value)
					if err != nil {
						log.Printf("Error decoding board state: %v", err)
						continue
					}

					// Print the board state
					fmt.Println(game.Position().Board().Draw())

					if game.Outcome() != chess.NoOutcome {
						fmt.Printf("Game completed. %s by %s.\n", game.Outcome(), game.Method())
					}

				case JoinLobbyRequest:

					// First, decode the TLV for the UUID
					uuidTag, uuidValue, err := DecodeTLV(value)
					if err != nil {
						log.Printf("Error decoding UUID TLV: %v", err)
						continue
					}

					// Verify the tag is ByteData
					if uuidTag != ByteData {
						log.Printf("Unexpected tag for UUID: %d, expected ByteData", uuidTag)
						continue
					}

					// Ensure the UUID value has enough data (16 bytes)
					if len(uuidValue) < 16 {
						log.Printf("Insufficient data for UUID: %d bytes", len(uuidValue))
						continue
					}

					// Extract the last 16 bytes for the UUID (same as UUIDPartie)
					uuidBytes := uuidValue[len(uuidValue)-16:]

					// Unmarshal the UUID from the bytes
					var gameID uuid.UUID
					err = gameID.UnmarshalBinary(uuidBytes)
					if err != nil {

						continue
					}

					// Log the parsed UUID

					SetGlobalGameID(gameID)

				case HelloRequest:

					// Handle HelloRequest

				case HelloResponse:

					// Handle HelloResponse

				default:

				}
			}
		}
	}()
}

// Stop terminates the TCP connection and listener
func (l *ContinuousTCPListener) Stop() {
	close(l.stopChan)
	l.wg.Wait()

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.conn != nil {
		l.conn.Close()
		l.conn = nil
	}
}
