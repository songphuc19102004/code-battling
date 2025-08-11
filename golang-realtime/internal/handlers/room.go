package handlers

import (
	"context"
	"golang-realtime/internal/events"
	"golang-realtime/pkg/common/request"
	"golang-realtime/pkg/common/response"
	"net/http"
	"strconv"

	"golang-realtime/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (hr *HandlerRepo) ListRoomsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	rooms, err := hr.queries.ListRooms(ctx)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, nil, true, "Failed to get rooms: "+err.Error())
		return
	}

	err = response.JSON(w, http.StatusOK, rooms, false, "get rooms successfully")
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

	ctx := context.Background()

	// Generate a random ID for the room

	// Convert description to pgtype.Text
	var description pgtype.Text
	if req.Description != "" {
		description = pgtype.Text{
			String: req.Description,
			Valid:  true,
		}
	}

	createParams := store.CreateRoomParams{
		Name:        req.Name,
		Description: description,
	}

	newRoom, err := hr.queries.CreateRoom(ctx, createParams)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, nil, true, "Failed to create room: "+err.Error())
		return
	}

	// Create a room manager for the new room
	roomManager := hr.gr.CreateRoom(newRoom.ID, hr.queries)
	go func() {
		roomManager.Start()
	}()

	err = response.JSON(w, http.StatusCreated, newRoom, false, "create room successfully")
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, nil, true, err.Error())
	}
}

func (hr *HandlerRepo) DeleteRoomHandler(w http.ResponseWriter, r *http.Request) {
	roomIdStr := chi.URLParam(r, "roomId") // Fixed parameter name
	roomId, err := strconv.ParseInt(roomIdStr, 10, 32)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, nil, true, "invalid room id")
		return
	}

	ctx := context.Background()

	// Check if room exists before attempting to delete
	_, err = hr.queries.GetRoom(ctx, int32(roomId))
	if err != nil {
		response.JSON(w, http.StatusNotFound, nil, true, "room not found")
		return
	}

	roomManager := hr.gr.GetRoomById(int32(roomId))
	if roomManager == nil {
		response.JSON(w, http.StatusNotFound, nil, true, "room not found")
		return
	}

	go func() {
		e := events.RoomDeleted{
			RoomId: int32(roomId),
		}
		roomManager.Events <- e
	}()

	response.JSON(w, http.StatusOK, nil, false, "delete room successfully")
}

func (hr *HandlerRepo) LeaveRoomHandler(w http.ResponseWriter, r *http.Request) {
	roomIdParam := chi.URLParam(r, "roomId")
	playerIdParam := chi.URLParam(r, "playerId")

	roomId, err := strconv.ParseInt(roomIdParam, 10, 32)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, nil, true, "invalid room id")
		return
	}

	playerId, err := strconv.ParseInt(playerIdParam, 10, 32)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, nil, true, "invalid player id")
		return
	}

	ctx := context.Background()

	// Check if room exists before attempting to leave
	roomManager := hr.gr.GetRoomById(int32(roomId))
	if roomManager == nil {
		response.JSON(w, http.StatusNotFound, nil, false, "room not found")
		return
	}

	// Check if player exists before attempting to leave
	_, err = hr.queries.GetPlayer(ctx, int32(playerId))
	if err != nil {
		response.JSON(w, http.StatusNotFound, nil, true, "player not found")
		return
	}

	go func() {
		e := events.PlayerLeft{
			PlayerId: int32(playerId),
			RoomId:   int32(roomId),
		}
		roomManager.Events <- e
	}()

	response.JSON(w, http.StatusOK, nil, false, "leave room successfully")
}
