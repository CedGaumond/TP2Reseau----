package main

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type UDPServer struct {
	conn           *net.UDPConn
	clientRegistry *ClientRegistry
	gameRegistry   *GameRegistry
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

type GameRegistry struct {
	games map[string]*Game
	mu    sync.Mutex
}

type Game struct {
	ID        string
	Players   map[string]*Player
	Started   bool
	Actions   []string
	CreatedAt time.Time
}

type Player struct {
	Username string
	Address  *net.UDPAddr
}

func NewUDPServer() *UDPServer {
	return &UDPServer{
		clientRegistry: NewClientRegistry(),
		gameRegistry:   &GameRegistry{games: make(map[string]*Game)},
		stopChan:       make(chan struct{}),
	}
}

func (srv *UDPServer) Start(port int) error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	srv.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to start UDP server: %w", err)
	}

	srv.wg.Add(1)
	go srv.listen()
	log.Printf("UDP server listening on port %d...\n", port)
	return nil
}

func (srv *UDPServer) listen() {
	defer srv.wg.Done()
	defer srv.conn.Close()

	for {
		select {
		case <-srv.stopChan:
			log.Println("Shutting down UDP server...")
			return
		default:
			// Prepare buffer for reading
			buf := make([]byte, 2048)
			n, clientAddr, err := srv.conn.ReadFromUDP(buf)
			if err != nil {
				log.Printf("Error reading UDP message: %v\n", err)
				continue
			}

			// Log received data (for debugging purposes)
			log.Printf("[DEBUG] Received data from %s: %x\n", clientAddr.String(), buf[:n])

			// Process the received data in a goroutine
			srv.wg.Add(1)
			go func(data []byte, addr *net.UDPAddr) {
				defer srv.wg.Done()
				srv.handleClientConnection(addr, data)
			}(buf[:n], clientAddr)
		}
	}
}

func (srv *UDPServer) handleClientConnection(clientAddr *net.UDPAddr, initialData []byte) {
	if clientAddr == nil {
		log.Println("Error: nil client address")
		return
	}

	clientAddress := clientAddr.String()
	clientInfo := &ClientInfo{ConnectedAt: time.Now()}

	remainingData := initialData

	// Process the received data
	var err error
	remainingData, err = srv.processIncomingData(remainingData, clientInfo, clientAddress, srv.conn, clientAddr)
	if err != nil {
		log.Printf("Error processing incoming data from %s: %v\n", clientAddress, err)
		return
	}

	srv.clientRegistry.AddClient(clientAddress, clientInfo)
}

func (srv *UDPServer) processIncomingData(data []byte, clientInfo *ClientInfo, clientAddress string, conn *net.UDPConn, clientAddr *net.UDPAddr) ([]byte, error) {
	// Log the raw data received
	log.Printf("Raw data received from %s: %v", clientAddress, data)

	// Ensure all parameters are non-nil
	if data == nil || clientInfo == nil || conn == nil || clientAddr == nil {
		return nil, fmt.Errorf("nil parameter passed to processIncomingData")
	}

	// Check if we have at least 3 bytes for the tag and length of the first TLV element
	if len(data) < 3 {
		log.Println("Insufficient data received, waiting for more.")
		return data, nil
	}

	// Decode the first TLV message to get the tag
	tag, value, err := DecodeTLV(data)
	if err != nil {
		log.Printf("Error decoding TLV tag: %v", err)
		return nil, fmt.Errorf("failed to decode TLV tag: %w", err)
	}

	// Log the decoded tag and value associated with it
	log.Printf("Decoded tag: %d (%s), Value: %v", tag, GetTagName(tag), value)

	// Process the request based on the tag
	switch tag {
	case HelloRequest:
		// Handle the HelloRequest for a UDP connection
		if err := HandleHelloRequest(nil, conn, clientAddr, data, false); err != nil {
			log.Printf("Error handling HelloRequest: %v", err)
			return nil, err
		}
		log.Println("HelloRequest successfully processed.")
		return data[len(value)+3:], nil // Skip the processed bytes

	case GameRequest:
		// Handle GameRequest (game-related logic)
		if err := HandleGameRequest(nil, conn, clientAddr, data, false); err != nil {
			log.Printf("Error handling GameRequest: %v", err)
			return nil, err
		}
		log.Println("GameRequest successfully processed.")
		return data[len(value)+3:], nil // Skip the processed bytes

	default:
		// Log unknown tags for debugging
		log.Printf("Unknown tag encountered: %d (%s)", tag, GetTagName(tag))
		return data[len(value)+3:], nil // Skip the processed bytes
	}
}

func (srv *UDPServer) handleGameRequest(clientAddr *net.UDPAddr, data []byte) error {
	// Example game logic, modify based on your actual game rules and protocol
	log.Printf("Processing game request from %s", clientAddr.String())
	// You can use clientAddr to identify the player and handle their game actions
	return nil
}

func (srv *UDPServer) Stop() {
	close(srv.stopChan)
	srv.wg.Wait()
}

func startUDPServer() {
	server := NewUDPServer()

	// Start the server
	if err := server.Start(8081); err != nil {
		log.Fatalf("Failed to start UDP server: %v", err)
	}

	select {}
}
