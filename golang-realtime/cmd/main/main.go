package main

import (
	"golang-realtime/internal/channels"
	"golang-realtime/internal/crunner"
	"golang-realtime/internal/handlers"

	"golang-realtime/internal/store"
	"log"
	"log/slog"
	"os"
	"runtime/debug"
	"sync"

	"github.com/lmittmann/tint"
)

type Application struct {
	wg       sync.WaitGroup
	cfg      *Config
	logger   *slog.Logger
	store    *store.Store
	gr       *channels.GlobalRooms
	handlers *handlers.HandlerRepo
}

type Config struct {
	Port int
}

func main() {

	cfg := &Config{Port: 8080}

	slogHandler := tint.NewHandler(os.Stdout, &tint.Options{Level: slog.LevelDebug, AddSource: true})
	logger := slog.New(slogHandler)
	slog.SetDefault(logger) // Set default for any library using slog's default logger

	store := store.NewStore()

	dockerClient := crunner.NewDockerClient()
	drunner := crunner.NewDockerRunner(dockerClient)
	gr := channels.NewGlobalRooms(store, drunner)

	handlerRepo := handlers.NewHandlerRepo(logger, gr, store)

	app := &Application{
		cfg:      cfg,
		logger:   logger,
		store:    store,
		gr:       gr,
		handlers: handlerRepo,
	}

	// err := queue.InitSender()
	// if err != nil {
	// 	panic(err)
	// }

	err := app.run()
	if err != nil {
		// Using standard log here to be absolutely sure it prints if slog itself had an issue
		log.Printf("CRITICAL ERROR from run(): %v\n", err)
		currentTrace := string(debug.Stack())
		log.Printf("Trace: %s\n", currentTrace)
		// Also log with slog if it's available
		slog.Error("CRITICAL ERROR from run()", "error", err.Error(), "trace", currentTrace)
		os.Exit(1)
	}
}
