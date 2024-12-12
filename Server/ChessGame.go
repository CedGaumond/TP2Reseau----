package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/notnil/chess"
)

// GameSession represents a chess game session with additional lobby information
type GameSession struct {
	ID            uuid.UUID
	Game          *chess.Game
	CreatorName   string
	LobbyName     string
	JoinedPlayers []string
	MaxPlayers    int
	IsLocked      bool
}

// GameStore is a map to store game sessions by their UUID
var GameStore = make(map[uuid.UUID]GameSession)
var LobbyNameToUUID = make(map[string]uuid.UUID)
var gameMutex = &sync.RWMutex{}

// createNewGame creates a new chess game session and adds it to the GameStore
func createNewGame(creatorName string, lobbyName string) uuid.UUID {
	gameMutex.Lock()
	defer gameMutex.Unlock()

	if _, exists := LobbyNameToUUID[lobbyName]; exists {
		log.Printf("Lobby name %s already exists", lobbyName)
		return uuid.Nil
	}

	gameID := uuid.New()
	game := chess.NewGame()

	session := GameSession{
		ID:            gameID,
		Game:          game,
		CreatorName:   creatorName,
		LobbyName:     lobbyName,
		JoinedPlayers: []string{creatorName},
		MaxPlayers:    2,
		IsLocked:      false,
	}

	GameStore[gameID] = session
	LobbyNameToUUID[lobbyName] = gameID
	return gameID
}

// joinGame allows a player to join an existing game lobby
func joinGame(lobbyName string, playerName string) (uuid.UUID, error) {
	gameMutex.Lock()
	defer gameMutex.Unlock()

	// Find the game UUID by lobby name
	gameID, exists := LobbyNameToUUID[lobbyName]
	if !exists {
		return uuid.Nil, fmt.Errorf("lobby %s does not exist", lobbyName)
	}

	// Retrieve the game session
	session, ok := GameStore[gameID]
	if !ok {
		return uuid.Nil, fmt.Errorf("game session not found for lobby %s", lobbyName)
	}

	// Check if the game is already locked or full
	if session.IsLocked {
		return uuid.Nil, fmt.Errorf("lobby %s is locked", lobbyName)
	}

	if len(session.JoinedPlayers) >= session.MaxPlayers {
		return uuid.Nil, fmt.Errorf("lobby %s is full", lobbyName)
	}

	// Check if player is already in the lobby
	for _, existingPlayer := range session.JoinedPlayers {
		if existingPlayer == playerName {
			return uuid.Nil, fmt.Errorf("player %s is already in the lobby", playerName)
		}
	}

	// Add the player to the lobby
	session.JoinedPlayers = append(session.JoinedPlayers, playerName)

	// If max players reached, lock the game
	if len(session.JoinedPlayers) >= session.MaxPlayers {
		session.IsLocked = true
	}

	// Update the game store
	GameStore[gameID] = session
	return gameID, nil
}

// listAvailableLobbies returns a list of available lobbies
func listAvailableLobbies() []string {
	gameMutex.RLock()
	defer gameMutex.RUnlock()

	availableLobbies := []string{}
	for lobbyName, gameID := range LobbyNameToUUID {
		session := GameStore[gameID]
		if !session.IsLocked {
			availableLobbies = append(availableLobbies, lobbyName)
		}
	}
	return availableLobbies
}
