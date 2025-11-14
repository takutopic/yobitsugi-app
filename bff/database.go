package main

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/firestore"
)

// InitFirestore initializes the Firestore client.
// It relies on the GOOGLE_APPLICATION_CREDENTIALS environment variable
// being set to find your service account JSON key.
func InitFirestore(ctx context.Context) *firestore.Client {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		log.Fatal("GCP_PROJECT_ID environment variable is not set.")
	}

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}

	log.Println("Firestore client initialized successfully.")
	return client
}