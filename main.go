package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

func main() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://mongo:27017"
	}
	fmt.Printf("Connecting to MongoDB at %s\n", mongoURI)

	clientOptions := options.Client().ApplyURI(mongoURI)
	client, _ = mongo.Connect(context.TODO(), clientOptions)

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/", handler)

	fmt.Println("Server listening on :8080")
	http.ListenAndServe(":8080", nil)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := client.Ping(ctx, nil)
	if err != nil {
		fmt.Printf("MongoDB not available: %v\n", err)
		http.Error(w, "MongoDB not available", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello from Go App!")
}
