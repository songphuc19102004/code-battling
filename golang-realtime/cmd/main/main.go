package main

import (
	"golang-realtime/database"
	"golang-realtime/internal/channels"
	"golang-realtime/internal/executor"
	"golang-realtime/internal/handlers"
	"golang-realtime/internal/store"
	"golang-realtime/pkg/common/env"
	"log"
	"log/slog"
	"os"
	"runtime/debug"
	"sync"

	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
)

type Application struct {
	wg       sync.WaitGroup
	cfg      *Config
	logger   *slog.Logger
	queries  *store.Queries
	gr       *channels.GlobalRooms
	handlers *handlers.HandlerRepo
}

type Config struct {
	Port int
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	dburl := env.GetString("DATABASE_URL", "")
	if dburl == "" {
		log.Fatal("DATABASE_URL not found")
	}

	cfg := &Config{Port: 8080}

	// test area
	connStr := env.GetString("DATABASE_URL", "")
	if connStr == "" {
		panic("DATABASE_URL environment variable is not set")
	}

	db, err := database.NewPool(connStr)
	if err != nil {
		panic(err)
	}

	queries := store.New(db)

	// log to os standard output
	slogHandler := tint.NewHandler(os.Stdout, &tint.Options{Level: slog.LevelDebug, AddSource: true})
	logger := slog.New(slogHandler)
	slog.SetDefault(logger) // Set default for any library using slog's default logger

	worker, err := executor.NewWorkerPool(logger, queries, &executor.WorkerPoolOptions{
		MaxWorkers:       5,
		MemoryLimitBytes: 6,
		MaxJobCount:      3,
		CpuNanoLimit:     1000,
	})
	if err != nil {
		panic(err)
	}
	gr := channels.NewGlobalRooms(queries, logger, worker)

	handlerRepo := handlers.NewHandlerRepo(logger, gr, queries)

	app := &Application{
		cfg:      cfg,
		logger:   logger,
		queries:  queries,
		gr:       gr,
		handlers: handlerRepo,
	}

	// err := queue.InitSender()
	// if err != nil {
	// 	panic(err)
	// }

	err = app.run()
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
