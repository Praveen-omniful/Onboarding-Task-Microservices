package main

import (
	"log"
	"os"

	"github.com/omniful/go_commons/db/sql/migration"
)

func main() {
	// Set environment variables for PostgreSQL connection
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_NAME", "ims_db")
	os.Setenv("DB_USER", "ims_user")
	os.Setenv("DB_PASSWORD", "ims_password")
	os.Setenv("DB_SSLMODE", "disable")

	// Get DB config from environment variables
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	dbname := os.Getenv("DB_NAME")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	sslmode := os.Getenv("DB_SSLMODE")

	log.Printf("Database config: host=%s, port=%s, dbname=%s, user=%s, sslmode=%s", host, port, dbname, user, sslmode)

	// Build the database URL
	dbURL := migration.BuildSQLDBURL(host, port, dbname, user, password)
	if sslmode != "" {
		dbURL += "?sslmode=" + sslmode
	}

	log.Printf("Database URL: %s", dbURL)

	// Set migration path to absolute path
	migrationPath := "file:///d/Microservices/ims-service/migrations"
	log.Printf("Migration path: %s", migrationPath)

	// Initialize migrator
	migrator, err := migration.InitializeMigrate(migrationPath, dbURL)
	if err != nil {
		log.Printf("Failed to initialize migrator: %v", err)
		log.Println("Continuing without migrations (database will use mock mode)")
		return
	}

	// Apply migrations
	err = migrator.Up()
	if err != nil {
		log.Printf("Failed to apply migrations: %v", err)
		log.Println("Continuing without migrations (database will use mock mode)")
		return
	}

	log.Println("✅ Migrations applied successfully!")
}
