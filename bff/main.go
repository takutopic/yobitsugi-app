// Package main
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Message is a struct to define our JSON structure
type Message struct {
	Text string `json."text"`
}

// corsMiddleware is a middleware which takes a handler and returns a new one.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS Header Settings
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods","GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
	
}

// rootHandler handles requests to the / endpoint.
func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "This is the homepage! 🏠")
}

// helloHandler handles requests to the /api/hello endpoint.
func helloHandler(w http.ResponseWriter, r *http.Request) {
	msg := Message{Text: "Hello from Go BFF! 🚀 (Now with JSON!)"}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msg)
}

func main() {
	http.Handle("/", corsMiddleware(http.HandlerFunc(rootHandler)))
	http.Handle("/api/hello", corsMiddleware(http.HandlerFunc(helloHandler)))

	fmt.Println("Go BFF server starting on http://localhost:8080 (with CORS Middleware)")
	log.Fatal(http.ListenAndServe(":8080", nil))
}