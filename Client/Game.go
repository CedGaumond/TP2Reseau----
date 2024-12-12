package main

import (
	"github.com/google/uuid"
	"sync"
)

// Global game instance with mutex for synchronization
var GlobalGame = struct {
	sync.RWMutex
	gameId uuid.UUID
}{}

// Function to get the global game ID
func GetGlobalGameID() uuid.UUID {
	GlobalGame.RLock()
	defer GlobalGame.RUnlock()
	return GlobalGame.gameId
}

// Function to set the global game ID
func SetGlobalGameID(newID uuid.UUID) {
	GlobalGame.Lock()
	defer GlobalGame.Unlock()
	GlobalGame.gameId = newID
}
