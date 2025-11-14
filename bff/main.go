// Package main
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
	
	"cloud.google.com/go/firestore"
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
	ID			  string `json:"id"`
	Status        string `json:"status"`
	AssessedTruth bool   `json:"assessedTruth"`
	Confidence    int    `json:"confidence"`
	Reasoning     string `json:"reasoning"`
}

// AssessmentJob defines the structure for the client job
type AssessmentJob struct {
	ID 					string  `firestore:"-" json:"ID"`
	CreatedAt 		 time.Time	`firestore:"createdAt" json:"CreatedAt"`
	UpdatedAt 		 time.Time	`firestore:"updatedAt" json:"UpdatedAt"`

	ClientID 			string	`firestore:"clientId" json:"ClientID"`
	Status 				string	`firestore:"status" json:"Status"`

	OriginalCode 		string	`firestore:"originalCode" json:"OriginalCode"`
	PatchedCode 		string	`firestore:"patchedCode" json:"PatchedCode"`
	BugDescription 		string	`firestore:"bugDescription" json:"BugDescription"`

	ResultStatus 		string	`firestore:"resultStatus,omitempty" json:"ResultStatus"`
	ResultAssessedTruth bool	`firestore:"resultAssessedTruth,omitempty" json:"ResultAssessedTruth"`
	ResultConfidence 	int		`firestore:"resultConfidence,omitempty" json:"ResultConfidence"`
	ResultReasoning 	string	`firestore:"resultReasoning,omitempty" json:"ResultReasoning"`
}

const pythonServiceURL = "http://localhost:8000/assess-kantei"
const jobsCollection = "jobs"

// assessHandler handles the POST request to /api/assess
func assessHandler(hub *Hub, fsClient *firestore.Client, c *gin.Context) {
	var request PatchRequest
	if err := c.BindJSON(&request); err != nil {
		log.Println("Error binding JSON:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	log.Printf("Received assessment request from ClientID: %s", request.ClientID)
	ctx := context.Background()

	// Create the initial job to the database
	job := AssessmentJob{
		CreatedAt:		time.Now(),
		UpdatedAt:		time.Now(),
		ClientID: 		request.ClientID,
		Status: 		"PENDING",
		OriginalCode: 	request.OriginalCode,
		PatchedCode: 	request.PatchedCode,
		BugDescription: request.BugDescription,
	}

	// Add the PENDING job to Firestore with unique ID.
	docRef, _, err := fsClient.Collection(jobsCollection).Add(ctx, job)
	if err != nil {
		log.Printf("Failed to create job in Firestore: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
		return
	}
	job.ID = docRef.ID
	log.Printf("Created job with ID: %d", job.ID)


	// Log that we received the data
	log.Printf("Received assessment request: BugDescription[len %d], OriginalCode[len %d], PatchedCode[len %d]",
			len(request.BugDescription), len(request.OriginalCode), len(request.PatchedCode))

	go func(req PatchRequest, currentJobDocID string) {
		log.Printf("[Goroutine %s] Starting REAL Python call for JobID %d", req.ClientID, currentJobDocID)
		jobDoc := fsClient.Collection(jobsCollection).Doc(currentJobDocID)

		sendErrorToClient := func(reason string, err error) {
			log.Printf("[Goroutine %s] Error: %s. %v", req.ClientID, reason, err)

			errorReasoning := fmt.Sprintf("%s (detail: %v)", reason, err)
			errorResult := AssessmentResult{
				ID: currentJobDocID,
				Status:	"ERROR",
				Reasoning: errorReasoning,
			}
			
			// Update the job in Firestore to ERROR
			_, updateErr := jobDoc.Set(ctx, map[string]interface{}{
				"status":		   "ERROR",
				"resultReasoning": errorReasoning,
				"updatedAt":	   time.Now(),
			}, firestore.MergeAll)
			if updateErr != nil {
				log.Printf("[Goroutine %s] Failed to update job to ERROR: %v", req.ClientID, updateErr)
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
		result.ID = currentJobDocID

		log.Printf("[Goroutine %s] Real Python call FINISHED.", req.ClientID)

		// Update the job in DB to COMPLETE
		_, updateErr := jobDoc.Set(ctx, map[string]interface{}{
			"status":			   "COMPLETE",
			"resultStatus":		   result.Status,
			"resultAssessedTruth": result.AssessedTruth,
			"resultConfidence":	   result.Confidence,
			"resultReasoning":	   result.Reasoning,
			"updatedAt":		   time.Now(),
		}, firestore.MergeAll)
		if updateErr != nil {
			log.Printf("[Goroutine %s] Failed to update job to COMPLETE: %v", req.ClientID, updateErr)
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
	}(request, job.ID)

	// Send an immediate "Accepted" response to React
	c.JSON(http.StatusAccepted, gin.H{"status": "Job accepted and processing", "jobID": job.ID})
}

// getJobByIDHandler fetches a single job by its ID, ensuring it belongs to the correct client.
func getJobByIDHandler(fsClient *firestore.Client, c *gin.Context) {
	jobID := c.Param("id")

	clientID := c.Query("clientId")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "clientId query parameter is required"})
		return
	}

	ctx := context.Background()
	doc, err := fsClient.Collection(jobsCollection).Doc(jobID).Get(ctx)
	if err != nil {
		log.Printf("Failed to fetch job from Firestore: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	var job AssessmentJob
	if err := doc.DataTo(&job); err != nil {
		log.Printf("Failed to map job data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse job data"})
		return
	}
	job.ID = doc.Ref.ID

	if job.ClientID != clientID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found or you do not have permission"})
		return
	}
	
	c.JSON(http.StatusOK, job)
}

// getJobsHandler fetches all jobs for a specific client.
func getJobsHandler(fsClient *firestore.Client, c *gin.Context) {
	clientID := c.Query("clientId")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "clientId query parameter is required"})
		return
	}

	var jobs []AssessmentJob
	ctx := context.Background()

	iter := fsClient.Collection(jobsCollection).
		Where("ClientId", "==", clientID).
		OrderBy("createdAt", firestore.Desc).
		Documents(ctx)

	docs, err := iter.GetAll()
	if err != nil {
		log.Printf("Failed to fetch jobs from Firestore: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch job history"})
		return
	}

	for _, doc := range docs {
		var job AssessmentJob
		if err := doc.DataTo(&job); err != nil {
			log.Printf("Failed to map job data: %v", err)
			continue
		}
		job.ID = doc.Ref.ID
		jobs = append(jobs, job)
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
	ctx := context.Background()

	// Create a default Gin router
	router := gin.Default()

	// Setup CORS Middleware
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	config.AllowMethods = []string{"GET", "POST", "OPTIONS"}
	router.Use(cors.New(config))

	// Init Firestore
	fsClient := InitFirestore(ctx)

	// Create and run the Hub
	hub := newHub()
	go hub.run() // Start the hub's main loop in a goroutine

	// Define the routes
	router.GET("/", rootHandler)
	router.GET("/api/jobs", func(c *gin.Context) {
		getJobsHandler(fsClient, c)
	})

	router.GET("/api/job/:id", func(c *gin.Context) {
		getJobByIDHandler(fsClient, c)
	})

	router.POST("/api/assess-kantei", func(c *gin.Context) {
		assessHandler(hub, fsClient, c)
	})

	// Add the new WebSocket route
	router.GET("/ws", func(c *gin.Context) {
		ServeWs(hub, c)
	})

	// Run the server
	log.Println("Gin BFF server starting on http://localhost:8080")
	log.Fatal(router.Run(":8080"))
}

