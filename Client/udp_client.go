package main

import (
	"encoding/base64"
	"fmt"
	"github.com/google/uuid"
	"github.com/notnil/chess"
	"log"
	"net"
	"sync"
)

// ContinuousUDPListener manages a persistent UDP connection
type ContinuousUDPListener struct {
	serverAddr string
	client     *Client
	conn       *net.UDPConn
	clientAddr *net.UDPAddr // Store the server's address
	stopChan   chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex
}

// NewContinuousUDPListener creates a new ContinuousUDPListener instance
func NewContinuousUDPListener(serverAddr string, client *Client) *ContinuousUDPListener {
	return &ContinuousUDPListener{
		serverAddr: serverAddr,
		client:     client,
		stopChan:   make(chan struct{}),
	}
}

// Connect establishes a UDP connection to the server
func (l *ContinuousUDPListener) Connect() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Resolve server's UDP address
	udpAddr, err := net.ResolveUDPAddr("udp", l.serverAddr)
	if err != nil {
		return fmt.Errorf("error resolving server address: %v", err)
	}

	// Establish UDP connection (connectionless, so we don't use a listener)
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("error connecting to server: %v", err)
	}

	l.conn = conn
	l.clientAddr = udpAddr

	log.Println("Connected to server:", l.serverAddr)
	return nil
}

// SendInitialHelloUDP sends the initial hello message to the server using UDP
func (l *ContinuousUDPListener) SendInitialHelloUDP() error {
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

	combinedTLV = append(combinedTLV, signatureTLV...)
	combinedTLV = append(combinedTLV, hashTLV...)

	_, err = l.conn.Write(combinedTLV)
	if err != nil {
		return fmt.Errorf("error sending combined TLV message: %v", err)
	}

	return nil
}

func (l *ContinuousUDPListener) Listen() {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()

		buf := make([]byte, 1024) // Buffer for incoming data
		for {
			select {
			case <-l.stopChan:
				log.Println("UDP listener stopped.")
				return
			default:
				// Read from the UDP connection indefinitely
				n, _, err := l.conn.ReadFromUDP(buf)
				if err != nil {
					// Handle read errors
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						// Timeout error, ignore and continue listening
						continue
					}

					log.Printf("Error reading from UDP connection: %v", err)
					return
				}

				// Decode the TLV message
				tag, value, err := DecodeTLV(buf[:n])
				if err != nil {
					log.Printf("Error decoding server response: %v", err)
					continue
				}

				// Print the decoded response
				log.Printf("Received TLV response: Tag=%s, Length=%d, Value=%x", GetTagName(tag), len(value), value)

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

					// Print the game moves (PGN)
					fmt.Println(game.String())

				// Other existing cases...
				case HelloRequest:
					log.Println("Received HelloRequest")
					// Handle HelloRequest

				case HelloResponse:
					log.Println("Received HelloResponse")
					// Handle HelloResponse

				default:
					log.Printf("Unknown tag: %v", tag)
					log.Printf("Data (hex): %x", value)
					log.Printf("Data (string): %s", string(value))
					log.Printf("Data (base64): %s", base64.StdEncoding.EncodeToString(value))
				}
			}
		}
	}()
}

// Stop terminates the UDP connection and listener
func (l *ContinuousUDPListener) Stop() {
	close(l.stopChan)
	l.wg.Wait()

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.conn != nil {
		l.conn.Close()
		l.conn = nil
	}
}
