package handlers

import (
	"encoding/json"
	"fmt"
	"golang-realtime/internal/events"
	"net/http"
)

func (hr *HandlerRepo) EventHandler(w http.ResponseWriter, r *http.Request) {
	playerId, roomId, err := getRequestPlayerIdAndRoomId(r, hr.logger)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set http headers required for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Get the room manager for the requested room.
	roomManager := hr.gr.GetRoomById(roomId)
	if roomManager == nil {
		http.Error(w, "room not found or not active", http.StatusNotFound)
		return
	}

	// listen for incoming SseEvents
	listen := make(chan events.SseEvent) // Add buffer to prevent blocking

	// Properly lock when modifying listeners
	roomManager.Mu.Lock()
	if roomManager.Listerners == nil {
		roomManager.Listerners = make(map[int32]chan<- events.SseEvent)
	}
	roomManager.Listerners[playerId] = listen
	roomManager.Mu.Unlock()

	defer hr.logger.Info("SSE connection closed", "player_id", playerId, "room_id", roomId)
	defer close(listen)
	defer func() {
		roomManager.Mu.Lock()
		delete(roomManager.Listerners, playerId)
		roomManager.Mu.Unlock()
		go func() {
			roomManager.Events <- events.PlayerLeft{PlayerId: playerId, RoomId: roomId}
		}()
	}()

	hr.logger.Info("SSE connection established", "player_id", playerId, "room_id", roomId)
	// player joined event
	go func() {
		roomManager.Events <- events.PlayerJoined{PlayerID: playerId, RoomID: roomId}
	}()

	for {
		select {
		case <-r.Context().Done():
			hr.logger.Info("SSE client disconnected", "player_id", playerId, "room_id", roomId)
			// player left event
			return
		case event, ok := <-listen:
			if !ok {
				hr.logger.Info("SSE client disconnected", "player_id", playerId, "room_id", roomId)
				return
			}

			hr.logger.Info("Sending event to player", "player_id", playerId, "event", event, "room_id", roomId)
			data, err := json.Marshal(event)
			if err != nil {
				hr.logger.Error("failed to marshal SSE event", "error", err, "player_id", playerId)
				return // Client is likely gone, so exit
			}

			if event.EventType != "" {
				fmt.Fprintf(w, "event: %s\n", event.EventType)
			}

			fmt.Fprintf(w, "data: %s\n\n", string(data))

			w.(http.Flusher).Flush()
		}
	}
}
