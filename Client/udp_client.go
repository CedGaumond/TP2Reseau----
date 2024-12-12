package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net"
	"sync"
	"time"
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

	log.Printf("Sent combined TLV message: Tag=HelloRequest, with Signature and Hash")
	return nil
}

func (l *ContinuousUDPListener) Listen() {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		defer func() {
			if l.conn != nil {
				l.conn.Close()
			}
		}()

		buf := make([]byte, 1024) // Buffer size, adjust if needed for larger messages
		for {
			select {
			case <-l.stopChan:
				log.Println("UDP listener stopped.")
				return
			default:
				// Set a read timeout to prevent blocking forever
				l.conn.SetReadDeadline(time.Now().Add(5 * time.Second))

				// Read from the connection
				n, err := l.conn.Read(buf)
				if err != nil {
					// Handle connection errors
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						// Timeout error, continue listening
						continue
					}

					log.Printf("Error reading from server: %v", err)
					return
				}

				// Log raw data for debugging
				log.Printf("Raw data received (%d bytes): %x", n, buf[:n])

				// Decode the TLV message
				tag, value, err := DecodeTLV(buf[:n])
				if err != nil {
					log.Printf("Error decoding server response: %v", err)
					continue
				}

				// Print the decoded response
				log.Printf("Received TLV response: Tag=%s, Length=%d, Value=%x", GetTagName(tag), len(value), value)

				switch tag {
				case UUIDPartie:
					// Check if there's enough data
					if len(value) < 16 {
						log.Printf("Insufficient data for UUID: %d bytes", len(value))
						continue
					}

					// Extract the last 16 bytes as the UUID
					uuidBytes := value[len(value)-16:]

					var uuidValue uuid.UUID
					err = uuidValue.UnmarshalBinary(uuidBytes)
					if err != nil {
						log.Printf("Error unmarshaling UUID: %v", err)
						continue
					}

					log.Printf("Received UUID: %s", uuidValue.String())

					// Save the UUID to the global game variable
					SetGlobalGameID(uuidValue)

				case HelloRequest:
					log.Println("Received HelloRequest")
					// Add specific handling for HelloRequest

				case HelloResponse:
					log.Println("Received HelloResponse")
					// Add specific handling for HelloResponse

				case UUIDClient:
					log.Println("Received UUIDClient")
					// Add specific handling for UUIDClient

				case Signature:
					log.Println("Received Signature")
					// Add specific handling for Signature

				case String:
					log.Printf("Received String: %s", string(value))
					// Add specific handling for String type

				case Int:
					// Convert byte slice to int
					intValue := binary.BigEndian.Uint32(value)
					log.Printf("Received Int: %d", intValue)
					// Add specific handling for Int type

				case ByteData:
					log.Printf("Received ByteData (hex): %x", value)
					// Add specific handling for ByteData

				case GameRequest:
					log.Println("Received GameRequest")
					// Add specific handling for GameRequest

				case GameResponse:
					log.Println("Received GameResponse")
					// Add specific handling for GameResponse

				case ActionRequest:
					log.Println("Received ActionRequest")
					// Add specific handling for ActionRequest

				case ActionResponse:
					log.Println("Received ActionResponse")
					// Add specific handling for ActionResponse

				default:
					// Print the received data in a human-readable format for unknown tags
					log.Printf("Received unknown tag: %v", tag)
					log.Printf("Received data (hex): %x", value)
					log.Printf("Received data (string): %s", string(value))
					log.Printf("Received data (base64): %s", base64.StdEncoding.EncodeToString(value))
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
