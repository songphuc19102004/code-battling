package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func (app *Application) routes() http.Handler {
	mux := chi.NewRouter()

	mux.Use(cors.AllowAll().Handler)

	mux.Route("/events", func(r chi.Router) {
		r.Get("/", app.handlers.EventHandler)
	})

	mux.Route("/submission", func(r chi.Router) {
		r.Post("/", app.handlers.SubmitSolutionHandler)
	})

	mux.Route("/rooms", func(r chi.Router) {
		r.Get("/", app.handlers.ListRoomsHandler)
		r.Post("/", app.handlers.CreateRoomHandler)
		r.Delete("/{roomId}", app.handlers.DeleteRoomHandler)

		r.Get("/{roomId}/leaderboard", app.handlers.GetLeaderboardHandler)

		r.Delete("/{roomId}/players/{playerId}", app.handlers.LeaveRoomHandler)
	})

	mux.Route("/players", func(r chi.Router) {
		r.Post("/", app.handlers.CreatePlayerHandler)
		r.Post("/login", app.handlers.LoginHandler)
	})

	mux.Route("/questions", func(r chi.Router) {
		r.Get("/", app.handlers.ListQuestionsHandler)
	})

	mux.Route("/isolate", func(r chi.Router) {
		r.Get("/test/{room_id}", app.handlers.GetIsolateTestHandler)
	})

	return mux
}
