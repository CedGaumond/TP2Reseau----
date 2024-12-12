package main

import "time"

func main() {
	// Lancer le serveur TCP en goroutine
	go startTCPServer()

	// Lancer le serveur UDP en goroutine
	go startUDPServer()

	// Attendre pendant 24 heures (1 jour)
	select {
	case <-time.After(time.Hour * 24):
		// Rien à faire, le programme attend
	}
}
