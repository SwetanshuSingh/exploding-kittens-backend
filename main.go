package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"os"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var ctx = context.Background()
var client *redis.Client

func main() {
	// Parse Redis URL
	redisURL := os.Getenv("URL")
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		panic(err)
	}

	// Create Redis client
	client = redis.NewClient(opt)

	// Check if the connection to Redis is successful
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		fmt.Println("Error connecting to Redis:", err)
		return
	}
	fmt.Println("Connected to Redis:", pong)

	// Initialize router
	router := mux.NewRouter()

	// Define API routes
	router.HandleFunc("/createUser/{username}", CreateUser).Methods("POST")
	router.HandleFunc("/winGame/{username}", WinGame).Methods("POST")
	router.HandleFunc("/leaderboard", GetLeaderBoard).Methods("GET")

	allowedOrigins := handlers.AllowedOrigins([]string{"http://localhost:5173", "https://exploding-kittens-eight.vercel.app"})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST"})
	allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type"})

	// Start the server
	fmt.Println("Server is running on :8080")
	http.ListenAndServe(":8080", handlers.CORS(allowedOrigins, allowedHeaders, allowedMethods)(router))
}

// CreateUser creates a new user with the specified username
func CreateUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	username := params["username"]

	// Check if the username already exists
	exists, err := client.Exists(ctx, username).Result()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if exists == 1 {
		// User already exists, return the existing username as JSON
		response := map[string]string{"message": "Username already exists", "username": username}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Initialize user with 0 wins
	err = client.Set(ctx, username, 0, 0).Err()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the username in the JSON response
	response := map[string]string{"message": "User created successfully", "username": username}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func WinGame(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	username := params["username"]

	// Increment user's win count
	newWinCount, err := client.Incr(ctx, username).Result()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with the updated win count
	response := map[string]interface{}{
		"message":   "Game won successfully",
		"win_count": newWinCount,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

type UserScore struct {
	Username string `json:"username"`
	Score    int    `json:"score"`
}

func GetLeaderBoard(w http.ResponseWriter, r *http.Request) {
	// Get all keys (usernames) from Redis
	keys, err := client.Keys(ctx, "*").Result()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Retrieve data for each user
	var usersData []UserScore
	for _, key := range keys {
		winCount, err := client.Get(ctx, key).Int()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		userScore := UserScore{Username: key, Score: winCount}
		usersData = append(usersData, userScore)
	}

	// Return user data as JSON array
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(usersData)
}
