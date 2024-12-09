// main.go
package main

import (
	"sync"
)

func main() {
	var wg sync.WaitGroup

	// Run the UDP server in a separate goroutine
	go ListenUDP("127.0.0.1", 8081, &wg)

	// Run the TCP server in a separate goroutine
	go ListenTCP("127.0.0.1", 8080, &wg)

	// Wait for all servers to finish (in this case, they run indefinitely)
	wg.Wait()
}
