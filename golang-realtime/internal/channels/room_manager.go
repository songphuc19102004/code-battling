package channels

import (
	"fmt"
	"golang-realtime/internal/events"
	"golang-realtime/internal/store"
	"log/slog"
	"sync"
)

// event-based
// each room will have a room manager, acting as a broadcaster for room-related events to all connected clients
// events is a single queue that received events from multiple sources and process it, then send to all listeners
// listeners are all the clients connected to the room, represented by their client IDs
type RoomManager struct {
	RoomId     int
	Events     chan any
	Listerners map[int]chan<- any
	logger     *slog.Logger
	store      *store.Store
	Mu         sync.RWMutex
}

// basically, GlobalRooms struct holds all the RoomManagers (channel) of each room
type GlobalRooms struct {
	Mu sync.RWMutex
	// roomId -> roomManager
	Rooms map[int]*RoomManager
}

func NewGlobalRooms(store *store.Store) *GlobalRooms {
	// Initialize testing rooms for development
	rooms := map[int]*RoomManager{
		1: NewRoomManager(1, store),
		2: NewRoomManager(2, store),
		3: NewRoomManager(3, store),
	}

	for _, rm := range rooms {
		go rm.Start()
	}

	return &GlobalRooms{
		Rooms: rooms,
	}
}

func (gr *GlobalRooms) GetRoomById(roomId int) *RoomManager {
	gr.Mu.RLock()
	defer gr.Mu.RUnlock()
	return gr.Rooms[roomId]
}

func (gr *GlobalRooms) CreateRoom(roomId int, store *store.Store) *RoomManager {
	rm := NewRoomManager(roomId, store)
	gr.Mu.Lock()
	gr.Rooms[roomId] = rm
	gr.Mu.Unlock()
	return rm
}

func NewRoomManager(roomId int, store *store.Store) *RoomManager {
	return &RoomManager{
		RoomId:     roomId,
		Events:     make(chan any),
		Listerners: make(map[int]chan<- any),
		logger:     slog.Default(),
		store:      store,
		Mu:         sync.RWMutex{},
	}
}

func (rm *RoomManager) Start() {
	for event := range rm.Events {
		switch e := event.(type) {
		case events.SolutionSubmitted:
			if err := rm.processSolutionSubmitted(e); err != nil {
				rm.logger.Error("failed to process solution submitted event", "error", err)
			}
		case events.PlayerJoined:
			if err := rm.processPlayerJoined(e); err != nil {
				rm.logger.Error("failed to process player joined event", "error", err)
			}
		case events.PlayerLeft:
			if err := rm.processPlayerLeft(e); err != nil {
				rm.logger.Error("failed to process player left event", "error", err)
			}
		case events.RoomDeleted:
			if err := rm.processRoomDeleted(e); err != nil {
				rm.logger.Error("failed to process room deleted event", "error", err)
			}
		}
	}
}

func (rm *RoomManager) dispatchEvent(event any) {
	rm.Mu.RLock()
	defer rm.Mu.RUnlock()

	rm.logger.Info("Hit dispatchEvent()",
		"Number of Listeners", len(rm.Listerners),
		"Event", event)

	for playerId, listener := range rm.Listerners {
		// Capture the listener variable properly
		go func(l chan<- any, pid int) {
			rm.logger.Info("dispatching", "listener_id", pid)
			select {
			case l <- event:
				// Successfully sent
			default:
				// Channel is full or closed, log but don't block
				rm.logger.Warn("failed to send event to listener", "player_id", pid)
			}
		}(listener, playerId)
	}
}

func (rm *RoomManager) processSolutionSubmitted(event events.SolutionSubmitted) error {
	// Process the solution submitted event
	data := fmt.Sprintf("data: playerId:%d,roomId:%d\n\n", event.PlayerId, rm.RoomId)
	rm.dispatchEvent(data)
	rm.logger.Info("solution submitted", "event", event)

	return nil
}

func (rm *RoomManager) processPlayerJoined(event events.PlayerJoined) error {
	// Process the player joined event
	data := fmt.Sprintf("data: playerId:%d,roomId:%d\n\n", event.PlayerId, rm.RoomId)
	go rm.dispatchEvent(data)
	rm.logger.Info("player joined", "event", event)

	return nil
}

func (rm *RoomManager) processPlayerLeft(event events.PlayerLeft) error {
	// Process the player left event
	data := fmt.Sprintf("data: playerId:%d,roomId:%d\n\n", event.PlayerId, rm.RoomId)
	go rm.dispatchEvent(data)
	rm.logger.Info("player left", "event", event)

	return nil
}

func (rm *RoomManager) processRoomDeleted(event events.RoomDeleted) error {
	// Process the room deleted event
	data := fmt.Sprintf("data: roomId:%d\n\n", rm.RoomId)
	go rm.dispatchEvent(data)
	rm.logger.Info("room deleted", "event", event)

	return nil
}
