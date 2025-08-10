package channels

import (
	"errors"
	"fmt"
	"golang-realtime/internal/crunner"
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
	crunner    crunner.CRunner
	logger     *slog.Logger
	store      *store.Store
	Mu         sync.RWMutex
}

// basically, GlobalRooms struct holds all the RoomManagers (channel) of each room
type GlobalRooms struct {
	Mu sync.RWMutex
	// roomId -> roomManager
	Rooms   map[int]*RoomManager
	crunner crunner.CRunner
}

func NewGlobalRooms(store *store.Store, crunner crunner.CRunner) *GlobalRooms {
	// Initialize testing rooms for development
	rooms := map[int]*RoomManager{
		1: NewRoomManager(1, store, crunner),
		2: NewRoomManager(2, store, crunner),
		3: NewRoomManager(3, store, crunner),
	}

	for _, rm := range rooms {
		go rm.Start()
	}

	return &GlobalRooms{
		Rooms:   rooms,
		crunner: crunner,
	}
}

func (gr *GlobalRooms) GetRoomById(roomId int) *RoomManager {
	gr.Mu.RLock()
	defer gr.Mu.RUnlock()
	return gr.Rooms[roomId]
}

func (gr *GlobalRooms) CreateRoom(roomId int, store *store.Store) *RoomManager {
	rm := NewRoomManager(roomId, store, gr.crunner)
	gr.Mu.Lock()
	gr.Rooms[roomId] = rm
	gr.Mu.Unlock()
	return rm
}

func NewRoomManager(roomId int, store *store.Store, crunner crunner.CRunner) *RoomManager {
	return &RoomManager{
		RoomId:     roomId,
		Events:     make(chan any),
		Listerners: make(map[int]chan<- events.SseEvent),
		logger:     slog.Default(),
		store:      store,
		Mu:         sync.RWMutex{},
		crunner:    crunner,
	}
}

func (rm *RoomManager) Start() {
	for event := range rm.Events {
		switch e := event.(type) {
		case events.SolutionSubmitted:
			if err := rm.processSolutionSubmitted(e); err != nil {
				rm.logger.Error("failed to process solution submitted event", "error", err)
			}

		case events.SolutionResult:
			if err := rm.processSolutionResult(e); err != nil {
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

func (rm *RoomManager) dispatchEvent(e events.SseEvent) {
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
		"Event", e)

	for playerId, listener := range listeners {
		// Capture the listener variable properly
		go func(l chan<- events.SseEvent, pid int) {
			defer func() {
				if r := recover(); r != nil {
					rm.logger.Error("panic while dispatching event", "error", r, "player_id", pid, "event", e)
				}
			}()

			rm.logger.Info("dispatching to", "player_id", pid)
			select {
			case l <- e:
				// Successfully sent
				rm.logger.Info("event sent to", "player_id", pid)
			default:
				// Channel is full or closed, log but don't block
				rm.logger.Warn("failed to send event to listener - channel full or closed", "player_id", pid)
			}
		}(listener, playerId)
	}
}

func (rm *RoomManager) dispatchEventToPlayer(e events.SseEvent, playerID int) {
	rm.Mu.RLock()
	if rm.Listerners == nil {
		rm.Mu.RUnlock()
		rm.logger.Warn("no listeners map found")
		return
	}

	// find the target listener
	var listener chan<- events.SseEvent
	for pid, l := range rm.Listerners {
		if pid == playerID {
			listener = l
		}
	}

	if listener == nil {
		rm.logger.Error("listener not found", "player_id", playerID)
	}

	rm.Mu.RUnlock()

	// Capture the listener variable properly
	go func(l chan<- events.SseEvent, pid int) {
		defer func() {
			if r := recover(); r != nil {
				rm.logger.Error("panic while dispatching event", "error", r, "player_id", pid, "event", e)
			}
		}()

		rm.logger.Info("dispatching to", "player_id", pid)
		select {
		case l <- e:
			// Successfully sent
			rm.logger.Info("event sent to", "player_id", pid)
		default:
			// Channel is full or closed, log but don't block
			rm.logger.Warn("failed to send event to listener - channel full or closed", "player_id", pid)
		}
	}(listener, playerID)
}

func (rm *RoomManager) processSolutionSubmitted(event events.SolutionSubmitted) error {
	rm.logger.Info("solution submitted", "event", event)

	// spin up container and run in the background
	go func() {
		runResult, err := rm.crunner.Run(rm.logger)
		if err != nil {
			rm.logger.Error("failed to run container", "error", err)
			return
		}

		e := events.SolutionResult{
			SolutionSubmitted: event,
			RunResult:         runResult,
		}

		rm.Events <- e
	}()

	return nil
}

// This function will work with database (updating tables) and send the fetchLeaderboard event to the frontend
func (rm *RoomManager) processSolutionResult(e events.SolutionResult) error {
	rm.logger.Info("processCorrectSoltuionResult() hit", "event", e)

	// compiled failed
	if e.RunResult.Result == crunner.Failure {
		rm.logger.Info("solution failed", "event", e)
		e.RunResult.Log = "runtime error: index out of range"
		sseEvent := events.SseEvent{
			EventType: events.WRONG_SOLUTION_SUBMITTED,
			Data:      fmt.Sprintf("log:%v", e.RunResult.Log),
		}

		go rm.dispatchEventToPlayer(sseEvent, e.SolutionSubmitted.PlayerId)

		return nil
	}

	// TODO: refactor two 2 separate functions
	err := rm.store.UpdatePlayerScoreAndRecalculateLeaderboard(e.SolutionSubmitted.RoomId, e.SolutionSubmitted.PlayerId, 50)
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
		rm.store.CalculateLeaderboard(event.RoomID)
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
	rm.Mu.Lock()
	defer rm.Mu.Unlock()

	// Process the player left event
	data := fmt.Sprintf("playerId:%d,roomId:%d\n\n", event.PlayerId, rm.RoomId)

	rm.store.RemoveRoomPlayer(event.RoomId, event.PlayerId)

	rm.store.CalculateLeaderboard(event.RoomId)

	sseEvent := events.SseEvent{
		EventType: events.PLAYER_LEFT,
		Data:      data,
	}

	go rm.dispatchEvent(sseEvent)
	rm.logger.Info("player left", "event", event)

	return nil
}

// TODO: Complete this shit
func (rm *RoomManager) processRoomDeleted(event events.RoomDeleted) error {
	rm.Mu.Lock()
	defer rm.Mu.Unlock()

	// Process the room deleted event
	data := fmt.Sprintf("roomId:%d\n\n", rm.RoomId)

	sseEvent := events.SseEvent{
		EventType: events.ROOM_DELETED,
		Data:      data,
	}

	rm.store.DeleteRoom(event.RoomId)

	rm.logger.Info("room deleted", "roomID", event.RoomId)

	go rm.dispatchEvent(sseEvent)

	return nil
}
