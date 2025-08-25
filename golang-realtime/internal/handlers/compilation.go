package handlers

import (
	"golang-realtime/internal/events"
	"golang-realtime/pkg/common/response"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// this handler will mimick the isolate running process
func (hr *HandlerRepo) GetIsolateTestHandler(w http.ResponseWriter, r *http.Request) {
	ct := events.CompilationTest{}
	roomID := chi.URLParam(r, "room_id")
	roomIDInt, err := strconv.ParseInt(roomID, 10, 32)
	if err != nil {
		response.JSON(w, http.StatusOK, nil, false, "get isolate test successfully")
		return
	}

	rm := hr.gr.GetRoomById(int32(roomIDInt))

	go func() {
		rm.Events <- ct
	}()

	response.JSON(w, http.StatusOK, nil, false, "get isolate test successfully")
}
