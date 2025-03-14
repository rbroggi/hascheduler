package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"hascheduler/internal"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	defaultMongoURI = "mongodb://mongo:27017/?connect=direct"
	serverAddress   = ":8080"
	dbName          = "demo"
)

func main() {
	setLogLevel(os.Getenv("LOG_LEVEL"))

	mongoURI := getMongoURI()
	client, db := connectToMongoDB(mongoURI)
	defer client.Disconnect(context.Background())

	elector := initializeElector()
	store := internal.NewStore(db)
	scheduler := initializeScheduler(elector, store)
	mux := initializeHttpServeMux(store)

	run(mux, scheduler, elector)
}

func getMongoURI() string {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = defaultMongoURI
	}
	slog.Info("Connecting to MongoDB", "uri", mongoURI)
	return mongoURI
}

func setLogLevel(logLevel string) {
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case "INFO":
		slog.SetLogLoggerLevel(slog.LevelInfo)
	case "WARN":
		slog.SetLogLoggerLevel(slog.LevelWarn)
	case "ERROR":
		slog.SetLogLoggerLevel(slog.LevelError)
	default:
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}
}

func connectToMongoDB(mongoURI string) (*mongo.Client, *mongo.Database) {
	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	handleError("Failed to connect to MongoDB", err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = client.Ping(ctx, nil)
	handleError("MongoDB not available", err)

	db := client.Database(dbName)
	return client, db
}

func initializeElector() *internal.Elector {
	elector, err := internal.NewElector()
	handleError("Failed to create elector", err)
	return elector
}

func initializeScheduler(elector *internal.Elector, store *internal.Store) *internal.Scheduler {
	scheduler, err := internal.NewScheduler(elector, store)
	handleError("Failed to create scheduler", err)
	return scheduler
}

func initializeHttpServeMux(store *internal.Store) *http.ServeMux {
	server := internal.NewService(store)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /schedules", server.List)
	mux.HandleFunc("POST /schedules", server.Create)
	mux.HandleFunc("PUT /schedules/{id}", server.Update)
	mux.HandleFunc("DELETE /schedules/{id}", server.Delete)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/", handler)
	return mux
}

func run(
	mux *http.ServeMux,
	scheduler *internal.Scheduler,
	elector *internal.Elector,
) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	s := &http.Server{Addr: serverAddress, Handler: mux}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		slog.Info("Server listening on " + serverAddress)
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server ListenAndServe", "error", err)
		}
	}()
	go func() {
		sig := <-sigCh
		slog.Info("Shutdown signal received", "signal", sig)
		cancel()
	}()

	go func() {
		if err := scheduler.Start(ctx); err != nil {
			slog.Error("Scheduler start", "error", err)
		}
	}()
	done := make(chan struct{}, 1)
	go func() {
		elector.Run(ctx)
		done <- struct{}{}
	}()
	<-done
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello from Go App!")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleError(message string, err error) {
	if err != nil {
		slog.Error(message, "error", err)
		os.Exit(1)
	}
}
