package handlers

import (
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

	// Note: Simple ID generation. In a real-world scenario, use UUIDs or a database sequence.
	newPlayer := &store.Player{
		ID:       hr.store.GetPlayersCount() + 1,
		Name:     req.Name,
		Password: req.Password,
	}

	hr.store.CreatePlayer(newPlayer)
	hr.logger.Info("New player created", "player_id", newPlayer.ID, "name", newPlayer.Name)
	hr.logger.Info("Player list now is", "playerList", hr.store.GetAllPlayers())

	err := response.JSON(w, http.StatusCreated, newPlayer, false, "Player created successfully")
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

	player, found := hr.store.GetPlayerByName(req.Name)
	if !found {
		response.JSON(w, http.StatusNotFound, nil, true, "Player not found")
		return
	}

	if player.Password != req.Password {
		response.JSON(w, http.StatusUnauthorized, nil, true, "Invalid password")
		return
	}

	err := response.JSON(w, http.StatusOK, player, false, "Login successful")
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, nil, true, err.Error())
	}
}
