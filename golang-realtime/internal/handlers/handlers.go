package handlers

import (
	"golang-realtime/internal/channels"
	"golang-realtime/internal/store"
	"log/slog"
)

// HandlerRepo holds all the dependencies required by the handlers.
// This includes the application logger, services like the RoomManager,
// and the centralized store for data access.
type HandlerRepo struct {
	logger  *slog.Logger
	gr      *channels.GlobalRooms
	queries *store.Queries
}

// NewHandlerRepo creates a new HandlerRepo with the provided dependencies.
func NewHandlerRepo(logger *slog.Logger, gr *channels.GlobalRooms, queries *store.Queries) *HandlerRepo {
	return &HandlerRepo{
		logger:  logger,
		gr:      gr,
		queries: queries,
	}
}
