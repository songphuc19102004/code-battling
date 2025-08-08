package channels

import (
	"errors"
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
	Listerners map[int]chan<- events.SseEvent
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
		Listerners: make(map[int]chan<- events.SseEvent),
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

		case events.CorrectSolutionResult:
			if err := rm.processCorrectSolutionResult(e); err != nil {
				rm.logger.Error("failed to process correct solution result event", "error", err)
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

// func (rm *RoomManager) dispatchEvent(event events.SseEvent) {
// 	rm.logger.Info("Hit dispatchEvent()",
// 		"Number of Listeners", len(rm.Listerners),
// 		"Event", event)

// 	for playerId, listener := range rm.Listerners {
// 		// Capture the listener variable properly
// 		go func(l chan<- events.SseEvent, pid int) {
// 			rm.logger.Info("dispatching to", "player_id", pid)
// 			select {
// 			case l <- event:
// 				// Successfully sent
// 				rm.logger.Info("event sent to", "player_id", pid)
// 			default:
// 				// Channel is full or closed, log but don't block
// 				rm.logger.Warn("failed to send event to listener", "player_id", pid)
// 			}
// 		}(listener, playerId)
// 	}
// }

func (rm *RoomManager) dispatchEvent(event events.SseEvent) {
	// Safely copy listeners to avoid race conditions
	rm.Mu.RLock()
	if rm.Listerners == nil {
		rm.Mu.RUnlock()
		rm.logger.Warn("no listeners map found")
		return
	}

	listeners := make(map[int]chan<- events.SseEvent)
	for pid, listener := range rm.Listerners {
		listeners[pid] = listener
	}
	rm.Mu.RUnlock()

	rm.logger.Info("Hit dispatchEvent()",
		"Number of Listeners", len(listeners),
		"Event", event)

	for playerId, listener := range listeners {
		// Capture the listener variable properly
		go func(l chan<- events.SseEvent, pid int) {
			defer func() {
				if r := recover(); r != nil {
					rm.logger.Error("panic while dispatching event", "error", r, "player_id", pid, "event", event)
				}
			}()

			rm.logger.Info("dispatching to", "player_id", pid)
			select {
			case l <- event:
				// Successfully sent
				rm.logger.Info("event sent to", "player_id", pid)
			default:
				// Channel is full or closed, log but don't block
				rm.logger.Warn("failed to send event to listener - channel full or closed", "player_id", pid)
			}
		}(listener, playerId)
	}
}

// This function only send the submitted code to the worker queue
func (rm *RoomManager) processSolutionSubmitted(event events.SolutionSubmitted) error {
	// Process the solution submitted event
	data := fmt.Sprintf("playerId:%d,roomId:%d\n\n", event.PlayerId, rm.RoomId)

	// for development stage, all solution are correct
	// ------------------------------------------
	correctSolutionResult := events.CorrectSolutionResult{
		SolutionSubmitted: event,
		RoomID:            event.RoomId,
	}

	//-------------------------------------------

	sseEvent := events.SseEvent{
		EventType: events.SOLUTION_SUBMITTED,
		Data:      data,
	}

	go rm.dispatchEvent(sseEvent)

	rm.logger.Info("solution submitted", "event", event)

	go func() {
		rm.Events <- correctSolutionResult
	}()

	return nil
}

// This function will work with database (updating tables) and send the fetchLeaderboard event to the frontend
func (rm *RoomManager) processCorrectSolutionResult(e events.CorrectSolutionResult) error {
	rm.logger.Info("processCorrectSoltuionResult() hit", "event", e)

	err := rm.store.UpdatePlayerScoreAndRecalculateLeaderboard(e.RoomID, e.SolutionSubmitted.PlayerId, 50)
	if err != nil {
		return err
	}

	sseEvent := events.SseEvent{
		EventType: events.CORRECT_SOLUTION_SUBMITTED,
		Data:      "",
	}

	go rm.dispatchEvent(sseEvent)

	return nil
}

func (rm *RoomManager) processPlayerJoined(event events.PlayerJoined) error {
	// Process the player joined event
	// Add player to room
	player, ok := rm.store.GetPlayer(event.PlayerID)
	if !ok {
		return errors.New("player not found")
	}

	if !rm.store.PlayerInRoom(event.RoomID, event.PlayerID) {
		roomPlayer := &store.RoomPlayer{
			PlayerID: player.ID,
			RoomID:   event.RoomID,
			Score:    0,
			Place:    0,
		}
		rm.store.AddRoomPlayer(event.RoomID, roomPlayer)
	}

	rm.logger.Info("player joined", "event", event)

	data := fmt.Sprintf("playerId:%d,roomId:%d\n\n", event.PlayerID, rm.RoomId)

	sseEvent := events.SseEvent{
		EventType: events.PLAYER_JOINED,
		Data:      data,
	}

	go rm.dispatchEvent(sseEvent)

	return nil
}

func (rm *RoomManager) processPlayerLeft(event events.PlayerLeft) error {
	// Process the player left event
	data := fmt.Sprintf("playerId:%d,roomId:%d\n\n", event.PlayerId, rm.RoomId)

	sseEvent := events.SseEvent{
		EventType: events.PLAYER_LEFT,
		Data:      data,
	}

	go rm.dispatchEvent(sseEvent)
	rm.logger.Info("player left", "event", event)

	return nil
}

func (rm *RoomManager) processRoomDeleted(event events.RoomDeleted) error {
	// Process the room deleted event
	data := fmt.Sprintf("roomId:%d\n\n", rm.RoomId)

	sseEvent := events.SseEvent{
		EventType: events.ROOM_DELETED,
		Data:      data,
	}

	go rm.dispatchEvent(sseEvent)
	rm.logger.Info("room deleted", "event", event)

	return nil
}
