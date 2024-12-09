// Users.go
package main

import (
	"fmt"
	"sync"
)

// Client represents a client with basic information
type Client struct {
	Nom                    string // Client's last name
	Prenom                 string // Client's first name
	Statut                 string // Client's status (active/inactive)
	Niveau                 int    // Client's level, ranging from 0-1000
	SignatureGivenByServer string // Signature provided by the server
	SignatureGivenByClient string // Signature provided by the client
}

// ClientsMap is a map that contains all the clients, protected by a Mutex for concurrent access
var ClientsMap = make(map[string]Client)
var clientsMutex sync.RWMutex

// AddClient adds a new client to ClientsMap
func AddClient(client Client) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	ClientsMap[client.Nom] = client
	fmt.Printf("Client added: %s %s\n", client.Nom, client.Prenom)
}

// GetClient retrieves a client by their name from the map
func GetClient(nom string) (Client, bool) {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()
	client, exists := ClientsMap[nom]
	return client, exists
}

// RemoveClient removes a client by their name from the map
func RemoveClient(nom string) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	delete(ClientsMap, nom)
	fmt.Printf("Client removed: %s\n", nom)
}

// ListClients returns all clients in the map
func ListClients() []Client {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()
	var clientList []Client
	for _, client := range ClientsMap {
		clientList = append(clientList, client)
	}
	return clientList
}
