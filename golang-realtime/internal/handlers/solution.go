package handlers

import (
	"golang-realtime/internal/channels"
	"golang-realtime/internal/events"
	"golang-realtime/pkg/common/request"
	"net/http"
	"time"
)

type SubmitSolutionRequest struct {
	QuestionId  int32     `json:"question_id"` // Changed to int32
	RoomId      int32     `json:"room_id"`     // Changed to int32
	Language    string    `json:"language"`
	Code        string    `json:"code"`
	PlayerId    int32     `json:"player_id"` // Changed to int32
	SubmittedAt time.Time `json:"submitted_at"`
}

func (hr *HandlerRepo) SubmitSolutionHandler(w http.ResponseWriter, r *http.Request) {
	var req SubmitSolutionRequest
	var roomManager *channels.RoomManager

	if err := request.DecodeJSON(w, r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	roomManager = hr.gr.GetRoomById(req.RoomId) // Use GetRoomById method instead of direct access
	if roomManager == nil {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	// Immediately acknowledge the request to the client.
	w.WriteHeader(http.StatusAccepted)

	// In a real application, you'd have more sophisticated validation logic here.
	// insert event to room manager
	roomManager.Events <- events.SolutionSubmitted{
		PlayerId:      req.PlayerId,
		RoomId:        req.RoomId,
		QuestionId:    req.QuestionId,
		Language:      req.Language,
		Code:          req.Code,
		SubmittedTime: req.SubmittedAt,
	}
}
