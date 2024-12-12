package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

var client = &Client{
	FirstName: "John",
	LastName:  "Doe",
	Status:    "Active",
	Level:     500,
}

// Main function
func main() {
	scanner := bufio.NewScanner(os.Stdin)

	// Create a wait group to manage graceful shutdown
	var wg sync.WaitGroup

	// Channel to handle graceful shutdown
	stop := make(chan struct{})

	// Listen for shutdown signals in a separate goroutine
	go func() {
		// Create a channel to receive OS signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Wait for a signal (e.g., Ctrl+C)
		<-sigChan

		// Notify all goroutines to stop
		fmt.Println("\nReceived shutdown signal. Stopping listener...")
		close(stop)
	}()

	// Flag to check if the connection is established
	var connectionEstablished bool
	// Channel to notify when the connection is established
	connectionReady := make(chan struct{})

	// Main loop to keep asking for connections
	for {
		// If the connection is not yet established, prompt for connection type
		if !connectionEstablished {
			fmt.Print("\nEnter connection type (tcp/udp or 'exit' to quit): ")
			scanner.Scan()
			connectionType := strings.ToLower(scanner.Text())

			if connectionType == "exit" {
				fmt.Println("Exiting client.")
				close(stop)
				break
			}

			if connectionType != "tcp" && connectionType != "udp" {
				fmt.Println("Invalid connection type. Please use 'tcp' or 'udp'.")
				continue
			}

			// Prompt for server address
			fmt.Print("Enter server address (e.g., localhost:8080): ")
			scanner.Scan()
			serverAddr := scanner.Text()

			var conn interface{}
			var isTCP bool

			// Create a listener based on TCP or UDP
			if connectionType == "tcp" {
				// Create a new continuous TCP listener
				listener := NewContinuousTCPListener(serverAddr, client)
				conn = listener
				isTCP = true

				// Add one goroutine to the wait group
				wg.Add(1)

				// Start the listener in a separate goroutine
				go func() {
					defer wg.Done()
					// Connect and send initial hello message
					if err := listener.Connect(); err != nil {
						log.Printf("Failed to connect: %v", err)
						return
					}

					// Send initial hello message
					if err := listener.SendInitialHello(); err != nil {
						log.Printf("Failed to send initial hello: %v", err)
						return
					}

					// Start listening for messages
					fmt.Println("Starting TCP listener...")
					listener.Listen()

					// After the listener starts, notify the main loop
					connectionReady <- struct{}{}
				}()
			} else if connectionType == "udp" {
				// Create a new continuous UDP listener
				listener := NewContinuousUDPListener(serverAddr, client)
				conn = listener
				isTCP = false

				// Add one goroutine to the wait group
				wg.Add(1)

				// Start the listener in a separate goroutine
				go func() {
					defer wg.Done()
					// Connect and send initial hello message
					if err := listener.Connect(); err != nil {
						log.Printf("Failed to connect: %v", err)
						return
					}

					// Send initial hello message
					if err := listener.SendInitialHelloUDP(); err != nil {
						log.Printf("Failed to send initial hello: %v", err)
						return
					}

					// Start listening for messages
					fmt.Println("Starting UDP listener...")
					listener.Listen()

					// After the listener starts, notify the main loop
					connectionReady <- struct{}{}
				}()
			}

			// Wait for the connection to be established
			<-connectionReady
			connectionEstablished = true

			// Once the connection is established, prompt for user actions
			handleUserActions(scanner, client, conn, isTCP)
		}
	}

	// Wait for all goroutines to finish before exiting the program
	wg.Wait()
}
func handleUserActions(scanner *bufio.Scanner, client *Client, conn interface{}, isTCP bool) {
	for {
		// Display available actions
		fmt.Println("\nSelect an action:")
		fmt.Println("1. Create a Game")
		fmt.Println("2. Join a Game")
		fmt.Println("3. See Lobby List")
		fmt.Println("4. Exit")
		fmt.Print("Enter your choice (1-4): ")

		scanner.Scan()
		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			// Create a game (send request to server)
			fmt.Println("Creating a new game...")

			// Prepare the data for the GameRequest

			// Ensure the correct connection type and call SendGameRequest
			if isTCP {
				// Assert the conn to *ContinuousTCPListener
				tcpListener, ok := conn.(*ContinuousTCPListener)
				if !ok {
					fmt.Println("Error: Invalid TCP connection type")
					continue
				}
				// Send the game request using TCP connection
				if err := SendGameRequest(tcpListener.conn, client); err != nil {
					fmt.Printf("Error creating game: %v\n", err)
					continue
				}
			} else {
				// Assert the conn to *ContinuousUDPListener
				udpListener, ok := conn.(*ContinuousUDPListener)
				if !ok {
					fmt.Println("Error: Invalid UDP connection type")
					continue
				}
				// Send the game request using UDP connection
				if err := SendGameRequest(udpListener.conn, client); err != nil {
					fmt.Printf("Error creating game: %v\n", err)
					continue
				}
			}
			// Simulate server response
			fmt.Println("Game created successfully!")
		case "2":
			// Join a game (send request to server)
			fmt.Println("Enter the game ID to join: ")
			scanner.Scan()
			gameID := scanner.Text()
			// Simulate joining the game
			fmt.Printf("Joining game with ID %s...\n", gameID)
			// Simulate server response
			fmt.Printf("Successfully joined game %s!\n", gameID)
		case "3":
			// See lobby list (send request to server)
			fmt.Println("Fetching lobby list...")

			var lobbyList []string
			var err error

			// Ensure the correct connection type (TCP or UDP) and call SendLobbyListRequest
			if isTCP {
				// Assert the conn to *ContinuousTCPListener
				tcpListener, ok := conn.(*ContinuousTCPListener)
				if !ok {
					fmt.Println("Error: Invalid TCP connection type")
					continue
				}
				// Send the lobby list request using TCP connection
				lobbyList, err = SendLobbyListRequest(tcpListener.conn, client)
			} else {
				// Assert the conn to *ContinuousUDPListener
				udpListener, ok := conn.(*ContinuousUDPListener)
				if !ok {
					fmt.Println("Error: Invalid UDP connection type")
					continue
				}
				// Send the lobby list request using UDP connection
				lobbyList, err = SendLobbyListRequest(udpListener.conn, client)
			}

			// Handle the error if the lobby list request fails
			if err != nil {
				fmt.Printf("Error fetching lobby list: %v\n", err)
				continue
			}

			// Display the lobby list
			fmt.Println("Available lobbies:")
			for _, lobby := range lobbyList {
				fmt.Println(lobby)
			}
		case "4":
			// Exit the menu
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Invalid choice, please select a valid option.")
		}
	}
}
