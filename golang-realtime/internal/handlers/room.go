package handlers

import (
	"golang-realtime/internal/store"
	"golang-realtime/pkg/common/request"
	"golang-realtime/pkg/common/response"
	"net/http"
	"strconv"
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
	roomIdStr := r.URL.Query().Get("roomId") // Keeping query parameter as chi.URLParam requires chi import
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

	// Delete room from memory and also delete its broadcaster
	// TODO: Implement some kind of pub/sub
	hr.store.DeleteRoom(roomId)

	response.JSON(w, http.StatusOK, nil, false, "delete room successfully")
}

type CreatePlayerRequest struct {
	Name string `json:"name"`
}

// CreatePlayerHandler handles the creation of a new player.
func (hr *HandlerRepo) CreatePlayerHandler(w http.ResponseWriter, r *http.Request) {
	var req CreatePlayerRequest
	if err := request.DecodeJSON(w, r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, nil, true, err.Error())
		return
	}

	// Note: Simple ID generation. In a real-world scenario, use UUIDs or a database sequence.
	newPlayer := &store.Player{
		ID:   hr.store.GetPlayersCount() + 1,
		Name: req.Name,
	}

	hr.store.CreatePlayer(newPlayer)
	hr.logger.Info("New player created", "player_id", newPlayer.ID, "name", newPlayer.Name)
	hr.logger.Info("Player list now is", "playerList", hr.store.GetAllPlayers())

	err := response.JSON(w, http.StatusCreated, newPlayer, false, "Player created successfully")
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, nil, true, err.Error())
	}
}
