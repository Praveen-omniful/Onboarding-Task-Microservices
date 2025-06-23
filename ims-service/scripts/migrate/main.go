package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"runtime"

	_ "github.com/lib/pq"
)

func main() {
	var (
		dbHost     = flag.String("db-host", "localhost", "Database host")
		dbPort     = flag.String("db-port", "5432", "Database port")
		dbUser     = flag.String("db-user", "postgres", "Database user")
		dbPassword = flag.String("db-password", "postgres", "Database password")
		dbName     = flag.String("db-name", "ims", "Database name")
	)

	flag.Parse()

	// Create database connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		*dbHost, *dbPort, *dbUser, *dbPassword, *dbName)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Test the connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}

	// Get the directory of the currently running file
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	// Path to the migration file
	migrationPath := filepath.Join(dir, "..", "..", "migrations", "000001_initial_schema.up.sql")

	// Read the migration file
	migrationSQL, err := ioutil.ReadFile(migrationPath)
	if err != nil {
		log.Fatalf("Error reading migration file: %v", err)
	}

	// Execute the migration
	_, err = db.Exec(string(migrationSQL))
	if err != nil {
		log.Fatalf("Error executing migration: %v", err)
	}

	log.Println("Migration completed successfully!")
}
