// Package main
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Message is a struct to define our JSON structure
type Message struct {
	Text string `json:"text"`
}

// PatchRequest defines the structure for the assessment request JSON
type PatchRequest struct {
	OriginalCode string `json:"originalCode"`
	PatchedCode  string `json:"patchedCode"`
}

// AssessmentResult defines the structure for the WebSocket broadcast
type AssessmentResult struct {
	Status 		  string 	`json:"status"`
	AssessedTruth bool 	`json:"assessedTruth"`
	Confidence 	  int 	`json:"confidence"`
	Reasoning 	  string 	`json:"reasoning"`
}

// assessHandler handles the POST request to /api/assess
func assessHandler(hub *Hub, c *gin.Context) {
	var request PatchRequest

	// 1. Bind the incoming JSON from React to our struct.
	// If binding fails, return a 400 Bad Request.
	if err := c.BindJSON(&request); err != nil {
		log.Println("Error binding JSON:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 2. Log that we received the data
	log.Printf("Recieved assessment request: OriginalCode length %d, PatchedCode length %d",
			len(request.OriginalCode), len(request.PatchedCode))

	// 3. Async Part
	// Launch a goroutine to handle the "slow" work (simulating Python call).
	// This allows us to send an immediate response to React
	go func() {
		log.Println("[Goroutine] Starting simulated Python call...")

		time.Sleep(5 * time.Second)

		log.Println("[Goroutine] Simulated Python call FINISHED.")

		// Create the new, detailed result
		result := AssessmentResult{
			Status: 		"PASS",
			AssessedTruth: 	true,
			Confidence: 	95,
			Reasoning: 		"The patch logic correctly addresses the off-by-one error (simulated).",
		}

		// Marshal the result struct into JSON bytes
		jsonResult, err := json.Marshal(result)
		if err != nil {
			log.Println("Error marshaling result:", err)
			return
		}

		// Send the JSON bytes to the hub's broadcast channel
		hub.broadcast <- jsonResult
	}()

	// 4. Send an immediate "Accepted" response to React
	// This tells React "We got your job, and we are working on it."
	c.JSON(http.StatusAccepted, gin.H{"status": "Job accepted and processing"})
}

// rootHandler handles requests to the / endpoint.
func rootHandler(c *gin.Context) {
	c.String(http.StatusOK, "This is the homepage! 🏠 (Now with Gin!)")
}

// helloHandler handles requests to the /api/hello endpoint.
func helloHandler(c *gin.Context) {
	msg := Message{Text: "Hello from Go BFF! 🚀 (Now with JSON and Gin!)"}

	c.JSON(http.StatusOK, msg)
}

// wsHandler handles the WebSocket connection request.
func wsHandler(hub *Hub, c *gin.Context) {
	// Upgrade the HTTP connection to a WebSocket connection.
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}

	// Create a new Client
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
	// Register the new client with the hub
	client.hub.register <- client

	log.Println("Client successfully upgraded to WebSocket.")

	// Start the client's goroutines
	// Allow the client to read messages and write messges
	go client.writePump()
	go client.readPump()
}

func main() {
	// Create a default Gin router
	router := gin.Default()

	// Setup CORS Middleware
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	router.Use(cors.New(config))

	// Create and run the Hub
	hub := newHub()
	go hub.run() // Start the hub's main loop in a goroutine

	// Define the routes
	router.GET("/", rootHandler)
	router.GET("/api/hello", helloHandler)
	router.POST("/api/assess", func(c *gin.Context) {
		assessHandler(hub, c)
	})

	// Add the new WebSocket route
	router.GET("/ws", func(c *gin.Context) {
		wsHandler(hub, c)
	})

	// Run the server
	log.Println("Gin BFF server starting on http://localhost:8080")
	log.Fatal(router.Run(":8080"))
}

