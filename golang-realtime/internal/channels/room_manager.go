package channels

import (
	"context"
	"fmt"
	"golang-realtime/internal/events"
	"golang-realtime/internal/executor"
	"golang-realtime/internal/store"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// event-based
// each room will have a room manager, acting as a broadcaster for room-related events to all connected clients
// events is a single queue that received events from multiple sources and process it, then send to all listeners
// listeners are all the clients connected to the room, represented by their client IDs
type RoomManager struct {
	RoomId        int32
	Events        chan any
	Listerners    map[int32]chan<- events.SseEvent
	worker        *executor.WorkerPool
	logger        *slog.Logger
	queries       *store.Queries
	Mu            sync.RWMutex // Protects Listerners map
	leaderboardMu sync.Mutex   // Protects leaderboard calculation
}

// basically, GlobalRooms struct holds all the RoomManagers (channel) of each room
type GlobalRooms struct {
	Mu      sync.RWMutex
	worker  *executor.WorkerPool
	logger  *slog.Logger
	queries *store.Queries
	// roomId -> roomManager
	Rooms map[int32]*RoomManager
}

func NewGlobalRooms(queries *store.Queries, logger *slog.Logger, worker *executor.WorkerPool) *GlobalRooms {
	// Initialize testing rooms for development
	rooms := map[int32]*RoomManager{
		1: NewRoomManager(1, queries, worker),
		2: NewRoomManager(2, queries, worker),
		3: NewRoomManager(3, queries, worker),
	}

	for _, rm := range rooms {
		go rm.Start()
	}

	return &GlobalRooms{
		Rooms:   rooms,
		worker:  worker,
		logger:  logger,
		queries: queries,
	}
}

func (gr *GlobalRooms) GetRoomById(roomId int32) *RoomManager {
	gr.Mu.RLock()
	defer gr.Mu.RUnlock()
	return gr.Rooms[roomId]
}

func (gr *GlobalRooms) CreateRoom(roomId int32, queries *store.Queries) *RoomManager {
	rm := NewRoomManager(roomId, queries, gr.worker)
	gr.Mu.Lock()
	gr.Rooms[roomId] = rm
	gr.Mu.Unlock()
	go rm.Start() // Start the room manager
	return rm
}

func NewRoomManager(roomId int32, queries *store.Queries, worker *executor.WorkerPool) *RoomManager {
	return &RoomManager{
		RoomId:        roomId,
		Events:        make(chan any, 10),
		Listerners:    make(map[int32]chan<- events.SseEvent),
		logger:        slog.Default(),
		queries:       queries,
		Mu:            sync.RWMutex{},
		leaderboardMu: sync.Mutex{}, // Initialize the new mutex
		worker:        worker,
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

	listeners := make(map[int32]chan<- events.SseEvent)
	for pid, listener := range rm.Listerners {
		listeners[pid] = listener
	}
	rm.Mu.RUnlock()

	rm.logger.Info("Hit dispatchEvent()",
		"Number of Listeners", len(listeners),
		"Event", e)

	for playerId, listener := range listeners {
		// Capture the listener variable properly
		go func(l chan<- events.SseEvent, pid int32) {
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

func (rm *RoomManager) dispatchEventToPlayer(e events.SseEvent, playerID int32) {
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
		rm.Mu.RUnlock()
		return
	}

	rm.Mu.RUnlock()

	// Capture the listener variable properly
	go func(l chan<- events.SseEvent, pid int32) {
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

	// Execute job
	o := rm.worker.ExecuteJob(event.Language, event.Code)

	result := events.SolutionResult{
		SolutionSubmitted: event,
		Result:            o,
	}

	rm.Events <- result

	return nil
}

func (rm *RoomManager) processSolutionResult(e events.SolutionResult) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	rm.logger.Info("processSoltuionResult() hit", "event", e)

	// compiled failed
	if !e.Result.Sucess {
		rm.logger.Info("solution failed", "event", e)
		sseEvent := events.SseEvent{
			EventType: events.WRONG_SOLUTION_SUBMITTED,
			Data:      fmt.Sprintf("log:%v", e.Result.Output),
		}

		// send compilation error to the player
		go rm.dispatchEventToPlayer(sseEvent, e.SolutionSubmitted.PlayerId)

		return nil
	}

	rm.queries.AddRoomPlayerScore(ctx, store.AddRoomPlayerScoreParams{
		RoomID:      e.SolutionSubmitted.RoomId,
		PlayerID:    e.SolutionSubmitted.PlayerId,
		ScoreTooAdd: pgtype.Int4{Int32: 50},
	})

	// Recalculate leaderboard after score update
	if err := rm.calculateLeaderboard(ctx); err != nil {
		rm.logger.Error("failed to calculate leaderboard after solution result", "error", err)
		// non-fatal, but should be monitored
	}

	sseEvent := events.SseEvent{
		EventType: events.CORRECT_SOLUTION_SUBMITTED,
		Data:      "",
	}

	// send event to the whole room
	go rm.dispatchEvent(sseEvent)

	return nil
}

// Helper method to check if player is in room
func (rm *RoomManager) playerInRoom(ctx context.Context, roomID, playerID int32) bool {
	_, err := rm.queries.GetRoomPlayer(ctx, store.GetRoomPlayerParams{
		RoomID:   roomID,
		PlayerID: playerID,
	})
	return err == nil
}

// Helper method to add player to room
func (rm *RoomManager) addPlayerToRoom(ctx context.Context, roomID, playerID int32) error {
	// place := rm.queries.room
	createParams := store.CreateRoomPlayerParams{
		RoomID:   roomID,
		PlayerID: playerID,
		Score: pgtype.Int4{
			Int32: 12,
			Valid: true,
		},
		Place: pgtype.Int4{
			Int32: 10,
			Valid: true,
		},
	}

	_, err := rm.queries.CreateRoomPlayer(ctx, createParams)
	return err
}

// Helper method to remove player from room
func (rm *RoomManager) removePlayerFromRoom(ctx context.Context, roomID, playerID int32) error {
	return rm.queries.DeleteRoomPlayer(ctx, store.DeleteRoomPlayerParams{
		RoomID:   roomID,
		PlayerID: playerID,
	})
}

// calculateLeaderboard recalculates and updates player ranks in a single, atomic, and concurrency-safe operation.
func (rm *RoomManager) calculateLeaderboard(ctx context.Context) error {
	// Lock to prevent concurrent calculations for the same room, which could cause deadlocks or race conditions.
	rm.leaderboardMu.Lock()
	defer rm.leaderboardMu.Unlock()

	rm.logger.Info("Starting leaderboard calculation for room", "room_id", rm.RoomId)

	// Use the new, highly efficient single query to update all ranks.
	// This avoids transactions in Go code and looping, pushing the logic to the database where it's most performant.
	err := rm.queries.UpdateRoomPlayerRanks(ctx, rm.RoomId)
	if err != nil {
		rm.logger.Error("Failed to update player ranks via single query", "room_id", rm.RoomId, "error", err)
		return err
	}

	rm.logger.Info("Finished calculating leaderboard for room", "room_id", rm.RoomId)
	return nil
}

func (rm *RoomManager) processPlayerJoined(event events.PlayerJoined) error {
	// Process the player joined event
	// Add player to room
	ctx := context.Background()
	player, err := rm.queries.GetPlayer(ctx, event.PlayerID)
	if err != nil {
		return err
	}

	if !rm.playerInRoom(ctx, event.RoomID, event.PlayerID) {
		rm.logger.Info("player is not in room, adding to room...",
			"player", player,
			"room", event.RoomID)
		err := rm.addPlayerToRoom(ctx, event.RoomID, player.ID)
		if err != nil {
			rm.logger.Error("failed to add player to room", "error", err)
			return err
		}
	}

	// Recalculate leaderboard after a player joins
	err = rm.calculateLeaderboard(ctx)
	if err != nil {
		rm.logger.Error("failed to calculate leaderboard after player joined", "error", err)
		// This is not fatal to the join operation, but should be monitored.
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
	ctx := context.Background()

	// Process the player left event
	data := fmt.Sprintf("playerId:%d,roomId:%d\n\n", event.PlayerId, rm.RoomId)

	err := rm.removePlayerFromRoom(ctx, event.RoomId, event.PlayerId)
	if err != nil {
		rm.logger.Error("failed to remove player from room", "error", err)
	}

	// Recalculate leaderboard after a player leaves
	err = rm.calculateLeaderboard(ctx)
	if err != nil {
		rm.logger.Error("failed to calculate leaderboard after player left", "error", err)
	}

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

	ctx := context.Background()

	// Process the room deleted event
	data := fmt.Sprintf("roomId:%d\n\n", rm.RoomId)

	sseEvent := events.SseEvent{
		EventType: events.ROOM_DELETED,
		Data:      data,
	}

	err := rm.queries.DeleteRoom(ctx, event.RoomId)
	if err != nil {
		rm.logger.Error("failed to delete room from database", "error", err)
	}

	rm.logger.Info("room deleted", "roomID", event.RoomId)

	go rm.dispatchEvent(sseEvent)

	return nil
}
