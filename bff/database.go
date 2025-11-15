package main

import (
	"log"

	"gorm.io/driver/postgres" // <-- ADD Postgres driver
	"gorm.io/driver/sqlite"   // <-- Keep SQLite driver
	"gorm.io/gorm"
)

// AssessmentJob is the GORM model for the database table.
type AssessmentJob struct {
	gorm.Model
	
	ClientID 		string `gorm:"index"`
	Status 	 		string // "PENDING", "COMPLETE", "ERROR"
	
	OriginalCode	string
	PatchedCode 	string
	BugDescription  string

	ResultStatus		  string
	ResultAssessmentTruth bool
	ResultConfidence	  int
	ResultReasoning		  string
}

// InitDatabase initializes the SQLite database connection and runs AutoMigrate.
func InitDatabase() *gorm.DB {
	var db *gorm.DB
	var err error

	dsn := os.Getenv("DATABASE_URL")
	if dsn != "" {
		log.Println("DATABASE_URL not set, failling back to SQLite.")
		db, err = gorm.Open(sqlite.Open("yobitsugi.db"), &gorm.Config{})
		if err != nil {
			log.Fatal("Failed to connect to SQLite database:", err)
		}
		log.Println("SQLite database connection established.")
	} else {
		log.Println("DATABASE_URL set, connecting to PostgreSQL...")
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatal("Failed to connect to PostgreSQL database:", err)
		}
		log.Println("PostgreSQL database connection established.")
	}	

	log.Println("Running database auto-migration...")
	if err := db.AutoMigrate(&AssessmentJob{}); err != nil {
		log.Fatal("Failed to auto-migrate database:", err)
	}
	return db
}