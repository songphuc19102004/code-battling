package handlers

import (
	"encoding/json"
	"fmt"
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

	// Get the room manager for the requested room.
	roomManager := hr.gr.GetRoomById(roomId)
	if roomManager == nil {
		http.Error(w, "room not found or not active", http.StatusNotFound)
		return
	}

	events := make(chan any, 10) // Add buffer to prevent blocking

	// Properly lock when modifying listeners
	roomManager.Mu.Lock()
	if roomManager.Listerners == nil {
		roomManager.Listerners = make(map[int]chan<- any)
	}
	roomManager.Listerners[playerId] = events
	roomManager.Mu.Unlock()

	defer hr.logger.Info("SSE connection closed", "player_id", playerId, "room_id", roomId)
	defer close(events)
	defer func() {
		roomManager.Mu.Lock()
		delete(roomManager.Listerners, playerId)
		roomManager.Mu.Unlock()
	}()

	hr.logger.Info("SSE connection established", "player_id", playerId, "room_id", roomId)

	for {
		select {
		case <-r.Context().Done():
			hr.logger.Info("SSE client disconnected", "player_id", playerId, "room_id", roomId)
			return
		case event := <-events:
			hr.logger.Info("Sending event to player", "player_id", playerId, "event", event, "room_id", roomId)
			data, _ := json.Marshal(event)
			content := fmt.Sprintf("data: %s\n\n", string(data))
			if _, err := w.Write([]byte(content)); err != nil {
				hr.logger.Error("failed to write SSE data", "error", err, "player_id", playerId)
				return
			}
			w.(http.Flusher).Flush()
		}
	}
}
