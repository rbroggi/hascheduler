package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hascheduler/internal"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelInfo)
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://mongo:27017"
	}
	slog.Info("Connecting to MongoDB", "uri", mongoURI)

	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		fmt.Printf("Failed to connect to MongoDB: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err = client.Ping(ctx, nil); err != nil {
		fmt.Printf("MongoDB not available: %v\n", err)
		return

	}
	db := client.Database("demo")
	elector, err := internal.NewElector(db)
	if err != nil {
		slog.Error("Failed to create elector", "error", err)
		os.Exit(1)
	}
	store := internal.NewStore(db)
	scheduler, err := internal.NewScheduler(elector, store)
	if err != nil {
		slog.Error("Failed to create scheduler", "error", err)
		os.Exit(1)
	}

	server := internal.NewServer(store)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/schedules", server.Handle)
	http.HandleFunc("/", handler)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel = context.WithCancel(context.Background())
	go func() {
		sig := <-sigCh
		slog.Info("Shutdown signal received", "signal", sig)
		cancel() // Cancel the context when a signal is received
	}()
	go func() {
		if err := scheduler.Start(ctx); err != nil {
			slog.Error("Scheduler start", "error", err)
		}
	}()
	go func() {
		slog.Info("Server listening on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server ListenAndServe", "error", err)
		}
	}()
	ch := elector.Run(ctx)

	// Wait for a termination signal
	<-ch
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
