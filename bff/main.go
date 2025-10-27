// Package main
package main

import (
	"fmt"
	"log"
	"net/http"
)

// helloHandler handles requests to the /api/hello endpoint
func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello from Go BFF! 🚀")
}

func main() {
	// Register the handler function for the /api/hello path
	http.HandleFunc("/api/hello", helloHandler)

	// Print a message to the console that the server is starting
	fmt.Println("Go BFF server starting on http://localhost:8080")

	// Start the server on port 8080
	// log.Fatal will print any errors and exit the program
	log.Fatal(http.ListenAndServe(":8080", nil))
}