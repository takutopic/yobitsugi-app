// Package main
package main

import (
	"bytes"
	"encoding/json"
	"io"
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
	BugDescription string `json:"bugDescription"`
	OriginalCode  string `json:"originalCode"`
	PatchedCode   string `json:"patchedCode"`
}

// AssessmentResult defines the structure for the WebSocket broadcast
type AssessmentResult struct {
	Status        string `json:"status"`
	AssessedTruth bool   `json:"assessedTruth"`
	Confidence    int    `json:"confidence"`
	Reasoning     string `json:"reasoning"`
}

const pythonServiceURL = "http://localhost:8000/assess-kantei"

// assessHandler handles the POST request to /api/assess
func assessHandler(hub *Hub, c *gin.Context) {
	var request PatchRequest

	// Bind the incoming JSON from React to our struct.
	// If binding fails, return a 400 Bad Request.
	if err := c.BindJSON(&request); err != nil {
		log.Println("Error binding JSON:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Log that we received the data
	log.Printf("Received assessment request: BugDescription[len %d], OriginalCode[len %d], PatchedCode[len %d]",
			len(request.BugDescription), len(request.OriginalCode), len(request.PatchedCode))

	go func(req PatchRequest) {
		log.Println("[Goroutine] Starting READ Python call...")

		jsonData, err := json.Marshal(req)
		if err != nil {
			log.Printf("[Goroutine] Error marshaling request for Python: %v, err")
			return
		}

		httpReq, err := http.NewRequest("POST", pythonServiceURL, bytes.NewBuffer((jsonData)))
		if err != nil {
			log.Printf("[Goroutine] Error creating request for Python: %v", err)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 30* time.Second}
		httpResp, err := client.Do(httpReq)
		if err != nil {
			log.Printf("[Goroutine] Error sending request to Python: %v", err)
			return
		}
		defer httpResp.Body.Close()

		body, err := io.ReadAll(httpResp.Body)
		if err != nil {
			log.Printf("[Goroutine] Error reading response from Python: %v", err)
			return
		}

		if httpResp.StatusCode != http.StatusOK {
			log.Printf("[Goroutine] Python service returned non-200 status: %s, Body: %s", httpResp.Status, string(body))
			return
		}

		var result AssessmentResult
		if err := json.Unmarshal(body, &result); err != nil {
			log.Printf("[Goroutine] Error unmarshaling response from Python: %v")
			return
		}

		log.Println("[Goroutine] Real Python call FINISHED.")

		jsonResult, err := json.Marshal(result)
		if err != nil {
			log.Println("Error marshaling final result", err)
			return
		}

		hub.broadcast <- jsonResult
	}(request)

	// Send an immediate "Accepted" response to React
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
		ServeWs(hub, c)
	})

	// Run the server
	log.Println("Gin BFF server starting on http://localhost:8080")
	log.Fatal(router.Run(":8080"))
}

