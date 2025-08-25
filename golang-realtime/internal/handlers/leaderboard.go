package handlers

import (
	"golang-realtime/pkg/common/response"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type LeaderboardEntry struct {
	PlayerName string `json:"player_name"`
	Score      int    `json:"score"`
	Place      int    `json:"place"`
}

type LeaderboardResponse struct {
	Entries []LeaderboardEntry `json:"entries"`
}

func (hr *HandlerRepo) GetLeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	roomIdParam := chi.URLParam(r, "roomId")
	hr.logger.Info("GetLeaderboardHandler hit", "roomId", roomIdParam)
	roomId, err := strconv.ParseInt(roomIdParam, 10, 32)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, nil, true, "invalid room ID")
		return
	}

	// Use the new, safe store method to get the leaderboard.
	// This prevents the nil pointer panic.
	ctx := r.Context()
	dbEntries, err := hr.queries.GetLeaderboardForRoom(ctx, int32(roomId))
	if err != nil {
		response.JSON(w, http.StatusNotFound, nil, true, err.Error())
		return
	}

	// Convert database entries to response format
	var entries []LeaderboardEntry
	for _, dbEntry := range dbEntries {
		entry := LeaderboardEntry{
			PlayerName: dbEntry.Name,
			Score:      int(dbEntry.Score.Int32), // Handle pgtype.Int4
			Place:      int(dbEntry.Place.Int32), // Handle pgtype.Int4
		}
		entries = append(entries, entry)
	}

	res := LeaderboardResponse{
		Entries: entries,
	}

	response.JSON(w, http.StatusOK, res, false, "get leaderboard successfully")
}

func getRequestPlayerIdAndRoomId(r *http.Request, logger *slog.Logger) (int32, int32, error) {
	roomIdStr := r.URL.Query().Get("room_id")
	playerIdStr := r.URL.Query().Get("player_id")

	roomId, err := strconv.ParseInt(roomIdStr, 10, 32)
	if err != nil {
		logger.Error("failed to parse room_id", "room_id", roomIdStr)
		return 0, 0, err
	}

	playerId, err := strconv.ParseInt(playerIdStr, 10, 32)
	if err != nil {
		logger.Error("failed to parse player_id", "player_id", playerIdStr)
		return 0, 0, err
	}

	logger.Info("getRequestPlayerIdAndRoomId", "player_id", playerId, "room_id", roomId)
	return int32(playerId), int32(roomId), nil
}
