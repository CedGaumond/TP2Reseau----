// UDPServer.go
package main

import (
	"fmt"
	"net"
	"sync"
)

// HandleClient handles incoming UDP messages
func HandleClient(conn *net.UDPConn, buffer []byte, clientAddr *net.UDPAddr, wg *sync.WaitGroup) {
	defer wg.Done()

	// Print the received message
	fmt.Printf("Message received from %s: %s\n", clientAddr, string(buffer))

	// Send a response back to the client with "Hi"
	_, err := conn.WriteToUDP([]byte("Hi"), clientAddr)
	if err != nil {
		fmt.Println("Error sending response:", err)
		return
	}
}

// ListenUDP listens for incoming UDP packets on the specified port and IP address
func ListenUDP(ip string, port int, wg *sync.WaitGroup) {
	addr := net.UDPAddr{
		Port: port,
		IP:   net.ParseIP(ip),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Println("Error starting UDP listener:", err)
		return
	}
	defer conn.Close()

	buffer := make([]byte, 1024)

	// Listen for incoming data
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading message:", err)
			continue
		}

		// Start a new goroutine to handle the client
		wg.Add(1)
		go HandleClient(conn, buffer[:n], clientAddr, wg)
	}
}
