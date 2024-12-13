package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type ClientInfo struct {
	Signature   string
	FirstName   string
	LastName    string
	Status      string
	Level       int
	ConnectedAt time.Time
}

// ClientRegistry gère les clients connectés
type ClientRegistry struct {
	mu      sync.RWMutex
	clients map[string]*ClientInfo
}

// NewClientRegistry crée un nouveau registre de clients
func NewClientRegistry() *ClientRegistry {
	return &ClientRegistry{
		clients: make(map[string]*ClientInfo),
	}
}

// AddClient ajoute ou met à jour un client dans le registre
func (cr *ClientRegistry) AddClient(address string, info *ClientInfo) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.clients[address] = info
}

// GetClient récupère un client par son adresse
func (cr *ClientRegistry) GetClient(address string) (*ClientInfo, bool) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	client, exists := cr.clients[address]
	return client, exists
}

// TCPServer gère le serveur TCP et les connexions des clients
type TCPServer struct {
	listener       net.Listener
	clientRegistry *ClientRegistry
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// NewTCPServer crée une nouvelle instance du serveur TCP
func NewTCPServer(port int) *TCPServer {
	return &TCPServer{
		clientRegistry: NewClientRegistry(),
		stopChan:       make(chan struct{}),
	}
}

func (srv *TCPServer) handleClientConnection(conn net.Conn) {
	clientAddress := conn.RemoteAddr().String()

	// Tampon pour accumuler les données entrantes
	buf := make([]byte, 2048)
	remainingData := []byte{}

	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err == net.ErrClosed {
				log.Printf("Connexion fermée : %s\n", clientAddress)
			} else if err.Error() == "EOF" {
				log.Printf("Le client a fermé la connexion : %s\n", clientAddress)
			} else {
				log.Printf("Erreur de lecture de la connexion TCP %s : %v\n", clientAddress, err)
			}
			return
		}

		// Traiter les données lues
		fullData := append(remainingData, buf[:n]...)

		// After processing the data, clear the buffer
		remainingData, err = srv.processIncomingData(fullData, clientAddress, conn)
		if err != nil {
			log.Printf("Erreur lors du traitement des données entrantes de %s : %v\n", clientAddress, err)
			return
		}

		// Clear the remainingData buffer for the next batch of data
		remainingData = []byte{}
	}
}

func (srv *TCPServer) processIncomingData(data []byte, clientAddress string, conn net.Conn) ([]byte, error) {
	// Log the raw data received
	log.Printf("Raw data received from %s: %x", clientAddress, data)

	// Ensure valid data is provided (at least tag + length)
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

	// Log the decoded tag and the value associated with it
	log.Printf("Decoded tag: %d (%s), Value: %v", tag, GetTagName(tag), value)

	// Handle known tags based on your protocol
	switch tag {
	case HelloRequest:
		// Handle the HelloRequest for a TCP connection
		if err := HandleHelloRequest(conn, nil, nil, data, true); err != nil {
			log.Printf("Error handling HelloRequest: %v", err)
			return nil, err
		}
		log.Println("HelloRequest successfully processed.")
		return data[len(value)+3:], nil // Skip the processed bytes

	case GameRequest:
		// Handle the GameRequest
		if err := HandleGameRequest(conn, nil, nil, data, true); err != nil {
			log.Printf("Error handling GameRequest: %v", err)
			return nil, err
		}
		log.Println("GameRequest successfully processed, GameResponse sent.")
		return data[len(value)+3:], nil // Skip the processed bytes

	case LobbyRequest:
		// Handle the LobbyListRequest
		if err := HandleLobbyListRequest(conn, nil, nil, data, true); err != nil {
			log.Printf("Error handling LobbyListRequest: %v", err)
			return nil, err
		}
		log.Println("LobbyListRequest successfully processed, LobbyResponse sent.")
		return data[len(value)+3:], nil // Skip the processed bytes

	case JoinLobbyRequest:
		// Handle the JoinLobbyRequest
		if err := HandleJoinRequest(conn, nil, nil, data, true); err != nil {
			log.Printf("Error handling JoinLobbyRequest: %v", err)
			return nil, err
		}
		log.Println("JoinLobbyRequest successfully processed.")
		return data[len(value)+3:], nil // Skip the processed bytes

	case BoardRequest:
		// Handle the BoardRequest (game board-related logic)
		if err := HandleBoardRequest(conn, nil, nil, data, true); err != nil {
			log.Printf("Error handling BoardRequest: %v", err)
			return nil, err
		}
		log.Println("BoardRequest successfully processed, BoardResponse sent.")
		return data[len(value)+3:], nil // Skip the processed bytes

	case ActionRequest:
		// Handle the MoveRequest (ActionRequest for making a move)
		if err := HandleMoveRequest(conn, nil, nil, data, true); err != nil {
			log.Printf("Error handling ActionRequest (MoveRequest): %v", err)
			return nil, err
		}
		log.Println("ActionRequest (MoveRequest) successfully processed.")
		return data[len(value)+3:], nil // Skip the processed bytes

	default:
		// Log unknown tags for debugging
		log.Printf("Unknown tag encountered: %d (%s)", tag, GetTagName(tag))
		// Return the remaining data for further processing
		return data[len(value)+3:], nil // Skip the processed bytes
	}
}

// Start commence à écouter les connexions TCP entrantes
func (srv *TCPServer) Start(port int) error {
	var err error
	srv.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("erreur lors du démarrage du serveur TCP sur le port %d : %w", port, err)
	}

	srv.wg.Add(1)
	go func() {
		defer srv.wg.Done()
		defer srv.listener.Close()

		log.Printf("Serveur TCP en écoute sur le port %d...\n", port)

		for {
			select {
			case <-srv.stopChan:
				log.Println("Arrêt du serveur TCP...")
				return
			default:
				conn, err := srv.listener.Accept()
				if err != nil {
					log.Printf("Erreur lors de l'acceptation de la connexion TCP : %v\n", err)
					continue
				}
				go srv.handleClientConnection(conn)
			}
		}
	}()

	return nil
}

// Stop arrête proprement le serveur TCP
func (srv *TCPServer) Stop() {
	close(srv.stopChan)
	srv.wg.Wait()
	if srv.listener != nil {
		srv.listener.Close()
	}
}

// Erreur personnalisée pour gérer les données insuffisantes lors du décodage TLV
var ErrInsufficientData = errors.New("données insuffisantes pour le décodage TLV")

// startTCPServer est une fonction de commodité pour démarrer le serveur TCP
func startTCPServer() {
	server := NewTCPServer(8080)

	// Démarrer le serveur
	if err := server.Start(8080); err != nil {
		log.Fatalf("Échec du démarrage du serveur TCP : %v", err)
	}

	// Optionnel : Ajouter un moyen d'arrêter le serveur proprement (par exemple, sur signal)
	// Pour l'instant, le serveur fonctionnera indéfiniment
	select {}
}
