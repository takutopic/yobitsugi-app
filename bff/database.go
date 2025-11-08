package main

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// AssessmentJob is the GORM model for the database table.
type AssessmentJob struct {
	gorm.Model
	
	ClientID string `gorm:"index"`
	Status 	 string // "PENDING", "COMPLETE", "ERROR"
	
	OriginalCode	string
	PatchedCode 	string
	BugDescription  string

	ResultStatus		  string
	ResultAssessmentTruth bool
	ResutlConfidence	  int
	ResultReasoning		  string
}

// InitDatabase initializes the SQLite database connection and runs AutoMigrate.
func InitDatabase() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("yobitsugi.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connection established.")

	log.Println("Running database auto-migration...")
	if err := db.AutoMigrate(&AssessmentJob{}); err != nil {
		log.Fatal("Failed to auto-migrate database:", err)
	}
	return db
}