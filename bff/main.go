// Package main
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	OriginalCode   string `json:"originalCode"`
	PatchedCode    string `json:"patchedCode"`
	ClientID	   string `json:"clientId"`
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
		log.Printf("[Goroutine %s] Starting READ Python call...", req.ClientID)

		sendErrorToClient := func(reason string, err error) {
			log.Printf("[Goroutine %s] Error: %s. %v", req.ClientID, reason, err)
			errorResult := AssessmentResult{
				Status:	"ERROR",
				AssessedTruth: false,
				Confidence: 0,
				Reasoning: fmt.Sprintf("%s (detail: %v)", reason, err),
			}
			jsonResult, _ := json.Marshal(errorResult)
			privateMsg := &PrivateMessage{
				ClientID: req.ClientID,
				Payload:  jsonResult,
			}
			hub.sendPrivate <- privateMsg
		}
		
		// Marshal the request data
		jsonData, err := json.Marshal(req)
		if err != nil {
			sendErrorToClient("Failed to create request for Kantei", err)
			return
		}

		// Create the HTTP request
		httpReq, err := http.NewRequest("POST", pythonServiceURL, bytes.NewBuffer((jsonData)))
		if err != nil {
			sendErrorToClient("Failed to create HTTP request", err)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		// Send the request
		client := &http.Client{Timeout: 30* time.Second}
		httpResp, err := client.Do(httpReq)
		if err != nil {
			sendErrorToClient("Failed to connect to Kantei service", err)
			return
		}
		defer httpResp.Body.Close()

		// Read the response
		body, err := io.ReadAll(httpResp.Body)
		if err != nil {
			sendErrorToClient("Failed to read response from Kantei", err)
			return
		}

		// Check for non-200 status
		if httpResp.StatusCode != http.StatusOK {
			reason := fmt.Sprintf("Kantei service returned non-200 status: %s", httpResp.Status)
			sendErrorToClient(reason, fmt.Errorf(string(body)))
			return
		}

		// Unmarshal the Python response
		var result AssessmentResult
		if err := json.Unmarshal(body, &result); err != nil {
			sendErrorToClient("Failed to parse Kantei response", err)
			return
		}

		log.Printf("[Goroutine %s] Real Python call FINISHED.", req.ClientID)

		// Marshal the *real* result for broadcasting
		jsonResult, err := json.Marshal(result)
		if err != nil {
			sendErrorToClient("Failed to marshal final result", err)
			return
		}

		// Send the real (successful) result to the client
		privateMsg := &PrivateMessage{
			ClientID: req.ClientID,
			Payload:  jsonResult,
		}
		hub.sendPrivate <- privateMsg
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

