package store

import (
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

// UpdateRoomPlayersWithLock allows thread-safe updates to room players using a callback function
func (s *Store) UpdateRoomPlayersWithLock(roomId int, updateFunc func([]*RoomPlayer) []*RoomPlayer) {
	s.roomPlayersMu.Lock()
	defer s.roomPlayersMu.Unlock()

	roomPlayers, ok := s.roomPlayers[roomId]
	if !ok {
		roomPlayers = []*RoomPlayer{}
	}

	s.roomPlayers[roomId] = updateFunc(roomPlayers)
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
		1: {ID: 1, Name: "Alice"},
		2: {ID: 2, Name: "Bob"},
		3: {ID: 3, Name: "Charlie"},
		4: {ID: 4, Name: "David"},
		5: {ID: 5, Name: "Phuc"},
	}

	s.questions = map[int]*Question{
		1: {ID: 1, Title: "Two Sum", Description: "Find two numbers that add up to a target"},
		2: {ID: 2, Title: "Anagram", Description: "Check if two strings are anagrams"},
		3: {ID: 3, Title: "Reverse Binary Tree", Description: "Reverse the order of nodes in a binary tree"},
		4: {ID: 4, Title: "Best time to buy and sell stock", Description: "Buy at the lowest price and sell at the highest price"},
		5: {ID: 5, Title: "Second largest element in an array", Description: "Second largest element in an array"},
	}
}
