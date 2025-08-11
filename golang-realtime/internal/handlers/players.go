package handlers

import (
	"context"
	"golang-realtime/internal/store"
	"golang-realtime/pkg/common/request"
	"golang-realtime/pkg/common/response"
	"net/http"
)

type CreatePlayerRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

// CreatePlayerHandler handles the creation of a new player.
func (hr *HandlerRepo) CreatePlayerHandler(w http.ResponseWriter, r *http.Request) {
	var req CreatePlayerRequest
	if err := request.DecodeJSON(w, r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, nil, true, err.Error())
		return
	}

	ctx := context.Background()

	// Create player with proper parameters
	createParams := store.CreatePlayerParams{
		Name:     req.Name,
		Password: req.Password,
	}

	newPlayer, err := hr.queries.CreatePlayer(ctx, createParams)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, nil, true, "Failed to create player: "+err.Error())
		return
	}

	hr.logger.Info("New player created", "player_id", newPlayer.ID, "name", newPlayer.Name)

	err = response.JSON(w, http.StatusCreated, newPlayer, false, "Player created successfully")
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, nil, true, err.Error())
	}
}

type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

// LoginHandler handles the login of an existing player.
func (hr *HandlerRepo) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := request.DecodeJSON(w, r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, nil, true, err.Error())
		return
	}

	hr.logger.Info("LoginHandler hit", "request", req)

	ctx := context.Background()
	player, err := hr.queries.GetPlayerByName(ctx, req.Name)
	if err != nil {
		hr.logger.Error("Failed to get player by name", "err", err)
		response.JSON(w, http.StatusNotFound, nil, true, "Player not found")
		return
	}

	if player.Password != req.Password {
		response.JSON(w, http.StatusUnauthorized, nil, true, "Invalid password")
		return
	}

	hr.logger.Info("login successful", "player", player)

	err = response.JSON(w, http.StatusOK, player, false, "Login successful")
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, nil, true, err.Error())
	}
}
