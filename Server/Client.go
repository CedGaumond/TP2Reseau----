package main

import (
	"fmt"
	"sync"
)

type Client struct {
	FirstName string
	LastName  string
	Status    string
	Level     int
	Signature string
	Address   string
}

// ClientList struct to manage multiple clients
type ClientList struct {
	mu      sync.Mutex
	clients map[string]Client
}

// Global ClientList instance
var clientList = NewClientList()

// NewClientList creates and returns a new ClientList
func NewClientList() *ClientList {
	return &ClientList{
		clients: make(map[string]Client),
	}
}

// AddClient adds a new client to the list
func (cl *ClientList) AddClient(address string, client Client) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	// Add the client to the map using address as key
	cl.clients[address] = client
}

// GetClient retrieves a client by their address
func (cl *ClientList) GetClient(address string) (Client, bool) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	client, exists := cl.clients[address]
	return client, exists
}

// GetClientSignature retrieves the signature of a client by their address
func (cl *ClientList) GetClientSignature(address string) (string, error) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	client, exists := cl.clients[address]
	if !exists {
		return "", fmt.Errorf("client with address %s not found", address)
	}
	return client.Signature, nil
}

// GetAllClients returns a list of all stored clients
func (cl *ClientList) GetAllClients() []Client {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	allClients := []Client{}
	for _, client := range cl.clients {
		allClients = append(allClients, client)
	}

	return allClients
}

// PrintAllClients prints out all the stored clients
func (cl *ClientList) PrintAllClients() {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	fmt.Println("All Clients:")
	for _, client := range cl.clients {
		fmt.Printf("Address: %s, Signature: %s\n", client.Address, client.Signature)
	}
}
