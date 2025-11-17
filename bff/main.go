// Package main
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
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
	ID			  uint 	 `json:"id"`
	Status        string `json:"status"`
	AssessedTruth bool   `json:"assessedTruth"`
	Confidence    int    `json:"confidence"`
	Reasoning     string `json:"reasoning"`
}

var pythonServiceURL = getPythonServiceURL()

func getPythonServiceURL() string {
	if url := os.Getenv("PYTHON_SERVICE_URL"); url != "" {
		return url
	}
	return "http://localhost:8000/assess-kantei"
}

// assessHandler handles the POST request to /api/assess
func assessHandler(hub *Hub, db *gorm.DB, c *gin.Context) {
	var request PatchRequest

	// Bind the incoming JSON from React to our struct.
	// If binding fails, return a 400 Bad Request.
	if err := c.BindJSON(&request); err != nil {
		log.Println("Error binding JSON:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Create the initial job to the database
	job := AssessmentJob{
		ClientID: 		request.ClientID,
		Status: 		"PENDING",
		OriginalCode: 	request.OriginalCode,
		PatchedCode: 	request.PatchedCode,
		BugDescription: request.BugDescription,
	}

	// Save the PENDING job to the DB
	if result := db.Create(&job); result.Error != nil {
		log.Printf("Failed to create job in DB: %v", result.Error)
		// Job is not saved.
	}
	log.Printf("Created job with ID: %d", job.ID)


	// Log that we received the data
	log.Printf("Received assessment request: BugDescription[len %d], OriginalCode[len %d], PatchedCode[len %d]",
			len(request.BugDescription), len(request.OriginalCode), len(request.PatchedCode))

	go func(req PatchRequest, currentJob AssessmentJob) {
		log.Printf("[Goroutine %s] Starting READ Python call for JobID %d", req.ClientID, currentJob.ID)

		sendErrorToClient := func(reason string, err error) {
			log.Printf("[Goroutine %s] Error: %s. %v", req.ClientID, reason, err)
			errorResult := AssessmentResult{
				ID: currentJob.ID,
				Status:	"ERROR",
				AssessedTruth: false,
				Confidence: 0,
				Reasoning: fmt.Sprintf("%s (detail: %v)", reason, err),
			}
			
			// Update the job in DB to ERROR
			currentJob.Status = "ERROR"
			currentJob.ResultReasoning = errorResult.Reasoning
			if err := db.Save(&currentJob).Error; err != nil {
				log.Printf("Failed to update job status: %v", err)
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

		result.ID = currentJob.ID

		log.Printf("[Goroutine %s] Real Python call FINISHED.", req.ClientID)

		// Update the job in DB to COMPLETE
		currentJob.Status = "COMPLETE"
		currentJob.ResultStatus = result.Status
		currentJob.ResultAssessmentTruth = result.AssessedTruth
		currentJob.ResultConfidence = result.Confidence
		currentJob.ResultReasoning = result.Reasoning
		if err := db.Save(&currentJob).Error; err != nil {
			log.Printf("Failed to update job status: %v", err)
		}

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
	}(request, job)

	// Send an immediate "Accepted" response to React
	// This tells React "We got your job, and we are working on it."
	c.JSON(http.StatusAccepted, gin.H{"status": "Job accepted and processing", "jobID": job.ID})
}

// getJobByIDHandler fetches a single job by its ID, ensuring it belongs to the correct client.
func getJobByIDHandler(db *gorm.DB, c *gin.Context) {
	jobID := c.Param("id")

	clientID := c.Query("clientId")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "clientId query parameter is required"})
		return
	}

	var job AssessmentJob

	result := db.Where("id = ? AND client_id = ?", jobID, clientID).First(&job)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found or you do not have permission"})
			return
		}
		log.Printf("Failed to fetch job from DB: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch job"})
		return
	}
	
	c.JSON(http.StatusOK, job)
}

// getJobsHandler fetches all jobs for a specific client.
func getJobsHandler(db *gorm.DB, c *gin.Context) {
	clientID := c.Query("clientId")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "clientId query parameter is required"})
		return
	}

	var jobs []AssessmentJob

	result := db.Where("client_id = ?", clientID).Order("created_at desc").Find(&jobs)
	if result.Error != nil {
		log.Printf("Failed to fetch jobs from DB: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch job history"})
		return
	}

	c.JSON(http.StatusOK, jobs)
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
	// Load local overrides when available; containers already receive env vars via docker-compose
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, relying on existing environment variables")
	}
	pythonServiceURL = getPythonServiceURL()

	// Create a default Gin router
	router := gin.Default()

	// Setup CORS Middleware
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	config.AllowMethods = []string{"GET", "POST", "OPTIONS"}
	router.Use(cors.New(config))

	// Init DB
	db := InitDatabase()

	// Create and run the Hub
	hub := newHub()
	go hub.run() // Start the hub's main loop in a goroutine

	// Define the routes
	router.GET("/", rootHandler)
	router.GET("/api/hello", helloHandler)
	router.GET("/api/jobs", func(c *gin.Context) {
		getJobsHandler(db, c)
	})

	router.GET("/api/job/:id", func(c *gin.Context) {
		getJobByIDHandler(db, c)
	})

	router.POST("/api/assess-kantei", func(c *gin.Context) {
		assessHandler(hub, db, c)
	})

	// Add the new WebSocket route
	router.GET("/ws", func(c *gin.Context) {
		ServeWs(hub, c)
	})

	// Run the server
	log.Println("Gin BFF server starting on http://localhost:8080")
	log.Fatal(router.Run(":8080"))
}

