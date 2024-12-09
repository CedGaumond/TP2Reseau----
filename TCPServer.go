package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
)

// HandleTCPClient processes the TCP client's request
func HandleTCPClient(conn net.Conn, wg *sync.WaitGroup) {
	defer conn.Close()
	defer wg.Done()

	// Read all the data sent by the client
	data, err := io.ReadAll(conn)
	if err != nil {
		fmt.Println("Error reading connection:", err)
		return
	}

	// Deserialize the data into a Client struct
	var request Client
	err = json.Unmarshal(data, &request)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	// Process the client data and generate the hashes for both client and server signatures
	clientSignatureHash, encryptedServerSignature, err := ProcessClientData(request)
	if err != nil {
		fmt.Println("Error processing client data:", err)
		return
	}

	// Prepare the response with the client signature hash and encrypted server signature
	response := fmt.Sprintf("Client signature hash: %s\nEncrypted server signature: %s", clientSignatureHash, encryptedServerSignature)

	// Send the response back to the client
	_, err = conn.Write([]byte(response))
	if err != nil {
		fmt.Println("Error sending response:", err)
		return
	}
}

// Start the server to listen for incoming connections
func ListenTCP(ip string, port int, wg *sync.WaitGroup) {
	addr := fmt.Sprintf("%s:%d", ip, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("Error starting TCP server:", err)
		return
	}
	defer listener.Close()

	// Accept incoming connections and handle them
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		// Handle the client in a new goroutine
		wg.Add(1)
		go HandleTCPClient(conn, wg)
	}
}
