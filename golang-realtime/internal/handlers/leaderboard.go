package handlers

import (
	"golang-realtime/pkg/common/response"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// updateAndSortLeaderboard is a write operation that updates the leaderboard data and sorts it.
// This is called by the solution submission handler. It uses the store for thread-safe updates.
// func (hr *HandlerRepo) updateAndSortLeaderboard(submittedSolution events.SolutionSubmitted) {
// 	hr.store.UpdateRoomPlayersWithLock(submittedSolution.RoomId, func(roomPlayers []*store.RoomPlayer) []*store.RoomPlayer {
// 		if len(roomPlayers) == 0 {
// 			hr.logger.Info("creating new leaderboard for room on first submission", "room_id", submittedSolution.RoomId)
// 		}

// 		var found bool
// 		for i := range roomPlayers { // Iterate by index to modify slice element directly
// 			if roomPlayers[i].PlayerId == submittedSolution.PlayerId {
// 				roomPlayers[i].Score += submittedSolution.Score
// 				found = true
// 				break // Exit loop once player is found and updated
// 			}
// 		}

// 		if !found {
// 			// This case handles a player submitting a solution to a room they weren't previously in.
// 			// We'll add them to the leaderboard.
// 			roomPlayer := &store.RoomPlayer{
// 				RoomId:   submittedSolution.RoomId,
// 				PlayerId: submittedSolution.PlayerId,
// 				Score:    submittedSolution.Score,
// 			}
// 			roomPlayers = append(roomPlayers, roomPlayer)
// 		}

// 		// Re-sort the leaderboard based on the new scores.
// 		sort.Slice(roomPlayers, func(i, j int) bool {
// 			// Sort descending by score
// 			if roomPlayers[i].Score != roomPlayers[j].Score {
// 				return roomPlayers[i].Score > roomPlayers[j].Score
// 			}
// 			// If scores are equal, maintain the existing order (stable sort).
// 			// A real implementation would use submission timestamps for tie-breaking.
// 			return false
// 		})

// 		// Assign new places and return the updated slice.
// 		for i := range roomPlayers {
// 			roomPlayers[i].Place = i + 1
// 		}
// 		return roomPlayers
// 	})
// }

type LeaderboardEntry struct {
	PlayerId int `json:"player_id"`
	Score    int `json:"score"`
	Place    int `json:"place"`
}

type LeaderboardResponse struct {
	Entries []LeaderboardEntry `json:"entries"`
}

func (hr *HandlerRepo) GetLeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	roomIdParam := chi.URLParam(r, "roomId")
	hr.logger.Info("GetLeaderboardHandler hit", "roomId", roomIdParam)
	roomId, err := strconv.Atoi(roomIdParam)
	if err != nil {
		http.Error(w, "invalid room ID", http.StatusBadRequest)
		return
	}

	roomPlayers, ok := hr.store.GetRoomPlayers(roomId)
	if !ok {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	var res LeaderboardResponse
	for _, player := range roomPlayers {
		res.Entries = append(res.Entries, LeaderboardEntry{
			PlayerId: player.PlayerID,
			Score:    player.Score,
			Place:    player.Place,
		})
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
