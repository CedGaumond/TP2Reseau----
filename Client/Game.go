package main

import (
	"github.com/google/uuid"
	"github.com/notnil/chess"
	"sync"
)

// Global game instance with mutex for synchronization
var GlobalGame = struct {
	sync.RWMutex
	gameId uuid.UUID   // Unique game identifier
	state  *chess.Game // Chess game state
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

// Function to get the global game state (the chess game)
func GetGlobalGameState() *chess.Game {
	GlobalGame.RLock()
	defer GlobalGame.RUnlock()
	return GlobalGame.state
}

// Function to set the global game state
func SetGlobalGameState(state *chess.Game) {
	GlobalGame.Lock()
	defer GlobalGame.Unlock()
	GlobalGame.state = state
}
