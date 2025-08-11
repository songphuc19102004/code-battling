package store

import (
	"errors"
	"log"
	"sort"
	"sync"
)

type Store struct {
	// Rooms data and mutex
	roomsMu sync.RWMutex
	rooms   map[int]*Room

	// RoomPlayers data and mutex
	roomPlayersMu sync.RWMutex
	roomPlayers   map[int][]*RoomPlayer

	// Players data and mutex
	playersMu sync.RWMutex
	players   map[int]*Player

	// Questions set
	questions map[int]*Question

	// Languages data and mutex
}

// NewStore creates a new store instance with initialized data
func NewStore() *Store {
	store := &Store{
		rooms:       make(map[int]*Room),
		roomPlayers: make(map[int][]*RoomPlayer),
		players:     make(map[int]*Player),
		questions:   make(map[int]*Question),
	}
	store.initInMemoryData()
	return store
}

// Room methods
func (s *Store) GetRoom(roomId int) (*Room, bool) {
	s.roomsMu.RLock()
	defer s.roomsMu.RUnlock()
	room, ok := s.rooms[roomId]
	return room, ok
}

func (s *Store) GetAllRooms() map[int]*Room {
	s.roomsMu.RLock()
	defer s.roomsMu.RUnlock()
	// Return a copy to prevent external modification
	result := make(map[int]*Room)
	for id, room := range s.rooms {
		result[id] = room
	}
	return result
}

func (s *Store) CreateRoom(room *Room) {
	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()
	s.rooms[room.ID] = room
}

func (s *Store) DeleteRoom(roomId int) bool {
	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()
	if _, exists := s.rooms[roomId]; exists {
		delete(s.rooms, roomId)
		return true
	}
	return false
}

func (s *Store) GetRoomsCount() int {
	s.roomsMu.RLock()
	defer s.roomsMu.RUnlock()
	return len(s.rooms)
}

// Player methods
func (s *Store) GetPlayer(playerId int) (*Player, bool) {
	s.playersMu.RLock()
	defer s.playersMu.RUnlock()
	player, ok := s.players[playerId]
	return player, ok
}

func (s *Store) GetPlayerByName(name string) (*Player, bool) {
	s.playersMu.RLock()
	defer s.playersMu.RUnlock()
	for _, player := range s.players {
		if player.Name == name {
			return player, true
		}
	}
	return nil, false
}

func (s *Store) GetAllPlayers() map[int]*Player {
	s.playersMu.RLock()
	defer s.playersMu.RUnlock()
	// Return a copy to prevent external modification
	result := make(map[int]*Player)
	for id, player := range s.players {
		result[id] = player
	}
	return result
}

func (s *Store) CreatePlayer(player *Player) {
	s.playersMu.Lock()
	defer s.playersMu.Unlock()
	s.players[player.ID] = player
}

func (s *Store) GetPlayersCount() int {
	s.playersMu.RLock()
	defer s.playersMu.RUnlock()
	return len(s.players)
}

// RoomPlayer methods
func (s *Store) GetRoomPlayers(roomId int) ([]*RoomPlayer, bool) {
	s.roomPlayersMu.RLock()
	defer s.roomPlayersMu.RUnlock()
	roomPlayers, ok := s.roomPlayers[roomId]
	if !ok {
		return nil, false
	}
	// Return a copy to prevent external modification
	result := make([]*RoomPlayer, len(roomPlayers))
	copy(result, roomPlayers)
	return result, true
}

func (s *Store) SetRoomPlayers(roomId int, roomPlayers []*RoomPlayer) {
	s.roomPlayersMu.Lock()
	defer s.roomPlayersMu.Unlock()
	s.roomPlayers[roomId] = roomPlayers
}

func (s *Store) AddRoomPlayer(roomId int, player *RoomPlayer) {
	s.roomPlayersMu.Lock()
	defer s.roomPlayersMu.Unlock()
	roomPlayers, ok := s.roomPlayers[roomId]
	if !ok {
		roomPlayers = make([]*RoomPlayer, 0)
	}
	roomPlayers = append(roomPlayers, player)
	s.roomPlayers[roomId] = roomPlayers
}

func (s *Store) RemoveRoomPlayer(roomId int, playerId int) {
	s.roomPlayersMu.Lock()
	defer s.roomPlayersMu.Unlock()
	logger := log.Default()

	logger.Printf("len room players before %v", len(s.roomPlayers[roomId]))
	logger.Printf("room players before\n")
	for _, player := range s.roomPlayers[roomId] {
		logger.Printf("room player %v", *player)
	}

	for i := range s.roomPlayers[roomId] {
		if s.roomPlayers[roomId][i].PlayerID == playerId {
			s.roomPlayers[roomId] = append(s.roomPlayers[roomId][:i], s.roomPlayers[roomId][i+1:]...)
			break
		}
	}

	logger.Printf("len room players AFTER %v", len(s.roomPlayers[roomId]))
	logger.Printf("room players AFTER\n")
	for _, player := range s.roomPlayers[roomId] {
		logger.Printf("room player %v", *player)
	}
}

func (s *Store) PlayerInRoom(roomId int, playerId int) bool {
	s.roomPlayersMu.RLock()
	defer s.roomPlayersMu.RUnlock()

	roomPlayers, ok := s.roomPlayers[roomId]
	if !ok {
		return false
	}
	for _, player := range roomPlayers {
		if player.PlayerID == playerId {
			return true
		}
	}
	return false
}

func (s *Store) UpdatePlayerScoreAndRecalculateLeaderboard(roomId int, playerId int, scoreChange int) error {
	s.roomPlayersMu.Lock()
	defer s.roomPlayersMu.Unlock()

	roomPlayers, ok := s.roomPlayers[roomId]
	if !ok {
		return errors.New("room not found")
	}

	// Find player and update score
	var playerFound bool
	for _, player := range roomPlayers {
		if player.PlayerID == playerId {
			player.Score += scoreChange
			playerFound = true
			break
		}
	}

	if !playerFound {
		return errors.New("player not found in room")
	}

	// Sort the players by score descending
	sort.Slice(roomPlayers, func(i, j int) bool {
		return roomPlayers[i].Score > roomPlayers[j].Score
	})

	// Update placement
	for i, player := range roomPlayers {
		player.Place = i + 1
	}

	return nil
}

func (s *Store) CalculateLeaderboard(roomId int) error {
	s.roomPlayersMu.Lock()
	defer s.roomPlayersMu.Unlock()

	roomPlayers, ok := s.roomPlayers[roomId]
	if !ok {
		return errors.New("room not found")
	}

	// Sort the players by score descending
	sort.Slice(roomPlayers, func(i, j int) bool {
		return roomPlayers[i].Score > roomPlayers[j].Score
	})

	// Update placement
	for i, player := range roomPlayers {
		player.Place = i + 1
	}

	return nil
}

// Update a specific room player
func (s *Store) UpdateRoomPlayer(roomId int, p *RoomPlayer) error {
	logger := log.Default()
	logger.Println("UpdateRoomPlayer() hit in store.go")
	logger.Printf("Update roomplayer %v for roomID %v", p, roomId)

	s.roomPlayersMu.Lock()
	defer s.roomPlayersMu.Unlock()

	roomPlayers, ok := s.roomPlayers[roomId]
	if !ok {
		return errors.New("room not found")
	}

	var targetPlayer *RoomPlayer
	for _, player := range roomPlayers {
		if player.PlayerID == p.PlayerID {
			targetPlayer = player
			break
		}
	}

	if targetPlayer == nil {
		return errors.New("player not found in room")
	}

	targetPlayer.Score = p.Score
	targetPlayer.Place = p.Place

	return nil
}

func (s *Store) GetLeaderboardForRoom(roomId int) ([]LeaderboardEntry, error) {
	s.roomPlayersMu.RLock()
	s.playersMu.RLock()
	defer s.roomPlayersMu.RUnlock()
	defer s.playersMu.RUnlock()

	roomPlayers, ok := s.roomPlayers[roomId]
	if !ok {
		return nil, errors.New("room not found")
	}

	var leaderboardEntries []LeaderboardEntry

	for _, rp := range roomPlayers {
		player, ok := s.players[rp.PlayerID]
		if !ok {
			// This is the condition that was causing the panic.
			// By handling it here, we prevent the nil pointer dereference.
			log.Printf("Player with ID %d found in room %d but not in global players list. Skipping.", rp.PlayerID, roomId)
			continue
		}
		leaderboardEntries = append(leaderboardEntries, LeaderboardEntry{
			PlayerName: player.Name,
			Score:      rp.Score,
			Place:      rp.Place,
		})
	}
	return leaderboardEntries, nil
}

// initInMemoryData populates the store with initial seed data
func (s *Store) initInMemoryData() {
	// Initialize Rooms
	s.rooms = map[int]*Room{
		1: {ID: 1, Name: "Nerd's room", Description: "We welcome nerds"},
		2: {ID: 2, Name: "FPTU Hackathon", Description: "Coding round 2 for FPTU Hackathon"},
		3: {ID: 3, Name: "Gooning Chamber", Description: "Welcome to the Gooning Chamber!"},
	}

	// Initialize Room Players
	s.roomPlayers = map[int][]*RoomPlayer{
		1: {
			{RoomID: 1, PlayerID: 1, Score: 100, Place: 1},
			{RoomID: 1, PlayerID: 2, Score: 80, Place: 2},
			{RoomID: 1, PlayerID: 3, Score: 60, Place: 3},
			{RoomID: 1, PlayerID: 4, Score: 50, Place: 4},
			{RoomID: 1, PlayerID: 5, Score: 20, Place: 5},
		},
	}

	// Initialize Players
	s.players = map[int]*Player{
		1: {ID: 1, Name: "alice", Password: "123"},
		2: {ID: 2, Name: "bob", Password: "123"},
		3: {ID: 3, Name: "charlie", Password: "123"},
		4: {ID: 4, Name: "david", Password: "123"},
		5: {ID: 5, Name: "phuc", Password: "123"},
	}

	s.questions = map[int]*Question{
		1: {ID: 1, Title: "Two Sum", Description: "Find two numbers that add up to a target"},
		2: {ID: 2, Title: "Anagram", Description: "Check if two strings are anagrams"},
		3: {ID: 3, Title: "Reverse Binary Tree", Description: "Reverse the order of nodes in a binary tree"},
		4: {ID: 4, Title: "Best time to buy and sell stock", Description: "Buy at the lowest price and sell at the highest price"},
		5: {ID: 5, Title: "Second largest element in an array", Description: "Second largest element in an array"},
	}

}
