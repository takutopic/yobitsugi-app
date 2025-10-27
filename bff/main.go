// Package main
package main

import (
	"log"
	"net/http"

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

// assessHandler handles the POST request to /api/assess
func assessHandler(c *gin.Context) {
	var request PatchRequest

	// 1. Bind the incoming JSON from React to our struct.
	// If binding fails, return a 400 Bad Request.
	if err := c.BindJSON(&request); err != nil {
		log.Println("Error binding JSON:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 2. Log that we recieved the data
	log.Println("Recieved assessment request: OriginalCode length %d, PatchedCode length %d",
			len(request.OriginalCode), len(request.PatchedCode))

	// 3. Async Part
	// Launch a goroutine to handle the "slow" work (simulating Python call).
	// This allows us to send an immediate response to React
	go func() {
		log.Println("[Goroutine] Starting simulated Python call...")

		time.Sleep(5 * time.Second)

		log.Println("[Goroutine] Simulated Python call FINISHED.")
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

func main() {
	// 1. Create a default Gin router
	router := gin.Default()

	// 2. Setup CORS Middleware
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:5173"}
	router.Use(cors.New(config))

	// 3. Define the routes
	router.GET("/", rootHandler)
	router.GET("/api/hello", helloHandler)
	router.POST("/api/assess", assessHandler)

	// 4. Run the server
	log.Println("Gin BFF server starting on http://localhost:8080")
	log.Fatal(router.Run(":8080"))
}