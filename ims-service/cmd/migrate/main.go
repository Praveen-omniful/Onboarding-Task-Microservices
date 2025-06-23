package main

import (
	"log"
	"os"

	"github.com/omniful/go_commons/db/sql/migration"
)

func main() {
	// Get DB config from environment variables or use defaults
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "ims"
	}
	user := os.Getenv("DB_USER")
	// No default, allow empty
	password := os.Getenv("DB_PASSWORD")
	// No default, allow empty

	dbURL := migration.BuildSQLDBURL(host, port, dbname, user, password)
	migrationPath := "file://../../migrations"

	migrator, err := migration.InitializeMigrate(migrationPath, dbURL)
	if err != nil {
		log.Fatalf("Failed to initialize migrator: %v", err)
	}

	err = migrator.Up()
	if err != nil {
		log.Fatalf("Failed to apply migrations: %v", err)
	}

	log.Println("Migrations applied successfully!")
}
