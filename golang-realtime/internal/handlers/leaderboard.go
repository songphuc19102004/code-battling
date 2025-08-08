package handlers

import (
	"golang-realtime/internal/store"
	"golang-realtime/pkg/common/response"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type LeaderboardResponse struct {
	Entries []store.LeaderboardEntry `json:"entries"`
}

func (hr *HandlerRepo) GetLeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	roomIdParam := chi.URLParam(r, "roomId")
	hr.logger.Info("GetLeaderboardHandler hit", "roomId", roomIdParam)
	roomId, err := strconv.Atoi(roomIdParam)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, nil, true, "invalid room ID")
		return
	}

	// Use the new, safe store method to get the leaderboard.
	// This prevents the nil pointer panic.
	leaderboardEntries, err := hr.store.GetLeaderboardForRoom(roomId)
	if err != nil {
		response.JSON(w, http.StatusNotFound, nil, true, err.Error())
		return
	}

	res := LeaderboardResponse{
		Entries: leaderboardEntries,
	}

	response.JSON(w, http.StatusOK, res, false, "get leaderboard successfully")
}

func getRequestPlayerIdAndRoomId(r *http.Request, logger *slog.Logger) (int, int, error) {
	roomIdStr := r.URL.Query().Get("room_id")
	playerIdStr := r.URL.Query().Get("player_id")

	roomId, err := strconv.Atoi(roomIdStr)
	if err != nil {
		logger.Error("failed to parse room_id", "room_id", roomIdStr)
		return 0, 0, err
	}

	playerId, err := strconv.Atoi(playerIdStr)
	if err != nil {
		logger.Error("failed to parse player_id", "player_id", playerIdStr)
		return 0, 0, err
	}

	logger.Info("getRequestPlayerIdAndRoomId", "player_id", playerId, "room_id", roomId)
	return playerId, roomId, nil
}
