package handlers

import (
	"golang-realtime/internal/events"
	"golang-realtime/internal/store"
	"golang-realtime/pkg/common/request"
	"golang-realtime/pkg/common/response"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	// Import sync package
)

// Note: The in-memory data and mutexes have been moved to the HandlerRepo struct
// in handlers.go to centralize state management.

func (hr *HandlerRepo) ListRoomsHandler(w http.ResponseWriter, r *http.Request) {
	rooms := hr.store.GetAllRooms()

	err := response.JSON(w, http.StatusOK, rooms, false, "get rooms successfully")

	if err != nil {
		response.JSON(w, http.StatusInternalServerError, nil, true, err.Error())
	}
}

type CreateRoomRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (hr *HandlerRepo) CreateRoomHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateRoomRequest

	err := request.DecodeJSON(w, r, &req)

	if err != nil {
		response.JSON(w, http.StatusBadRequest, nil, true, err.Error())
		return
	}

	// A more robust ID generation is needed for a real application (e.g., UUIDs).
	newRoom := store.Room{
		ID:          hr.store.GetRoomsCount() + 1,
		Name:        req.Name,
		Description: req.Description,
	}

	// this operation is like adding room to Rooms table in database.
	// we shouldn't add rooms to database ?
	hr.store.CreateRoom(&newRoom)

	// Create a broadcaster for the new room.
	// TODO: Implement some kind of channels

	roomManager := hr.gr.CreateRoom(newRoom.ID, hr.store)

	go func() {
		roomManager.Start()
	}()

	err = response.JSON(w, http.StatusCreated, newRoom, false, "create room successfully")

	if err != nil {
		response.JSON(w, http.StatusInternalServerError, nil, true, err.Error())
	}
}

func (hr *HandlerRepo) DeleteRoomHandler(w http.ResponseWriter, r *http.Request) {
	roomIdStr := chi.URLParam(r, "room_id")
	roomId, err := strconv.Atoi(roomIdStr)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, nil, true, "invalid room id")
		return
	}

	// Check if room exists before attempting to delete
	if _, exists := hr.store.GetRoom(roomId); !exists {
		response.JSON(w, http.StatusNotFound, nil, true, "room not found")
		return
	}

	roomManager := hr.gr.GetRoomById(roomId)
	if roomManager == nil {
		response.JSON(w, http.StatusNotFound, nil, true, "room not found")
		return
	}

	go func() {
		e := events.RoomDeleted{
			RoomId: roomId,
		}
		roomManager.Events <- e
	}()

	response.JSON(w, http.StatusOK, nil, false, "delete room successfully")
}

func (hr *HandlerRepo) LeaveRoomHandler(w http.ResponseWriter, r *http.Request) {
	roomIdParam := chi.URLParam(r, "roomId")
	playerIdParam := chi.URLParam(r, "playerId")

	roomId, err := strconv.Atoi(roomIdParam)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, nil, true, "invalid room id")
		return
	}

	playerId, err := strconv.Atoi(playerIdParam)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, nil, true, "invalid player id")
		return
	}

	// Check if room exists before attempting to leave
	roomManager := hr.gr.GetRoomById(roomId)
	if roomManager == nil {
		response.JSON(w, http.StatusNotFound, nil, false, "room not found")
		return
	}

	// Check if player exists before attempting to leave
	if _, exists := hr.store.GetPlayer(playerId); !exists {
		response.JSON(w, http.StatusNotFound, nil, true, "player not found")
		return
	}

	go func() {
		e := events.PlayerLeft{
			PlayerId: playerId,
			RoomId:   roomId,
		}
		roomManager.Events <- e
	}()

	response.JSON(w, http.StatusOK, nil, false, "leave room successfully")
}
