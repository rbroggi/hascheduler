package main

import (
	"context"
	"fmt"
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
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://mongo:27017"
	}
	fmt.Printf("Connecting to MongoDB at %s\n", mongoURI)

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
		fmt.Printf("Failed to create elector: %v\n", err)
		os.Exit(1)
	}
	store := internal.NewStore(db)
	scheduler, err := internal.NewScheduler(elector, store)
	if err != nil {
		fmt.Printf("Failed to create scheduler: %v\n", err)
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
		fmt.Println("\nReceived signal:", sig)
		cancel() // Cancel the context when a signal is received
	}()
	go scheduler.Start(ctx)
	go func() {
		fmt.Println("Server listening on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP server ListenAndServe: %v\n", err)
		}
	}()
	ch := elector.Run(ctx)

	// Wait for a termination signal
	<-sigCh
	fmt.Println("Shutdown signal received")
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
