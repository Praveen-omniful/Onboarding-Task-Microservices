package config

import (
	"fmt"
	"log"

	"github.com/omniful/go_commons/db/sql/postgres"
)

// DBCluster is the global database cluster handle
var DBCluster *postgres.DbCluster

// InitDB initializes the database connection using your config
func InitDB(cfg *Config) error {
	// Use the configuration from environment variables (docker-compose setup)
	masterConfig := postgres.DBConfig{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		Username: cfg.Database.User,
		Password: cfg.Database.Password,
		Dbname:   cfg.Database.DBName,
	}

	slavesConfig := make([]postgres.DBConfig, 0) // for read replicas, if any

	log.Printf("Connecting to database: host=%s port=%s dbname=%s user=%s",
		masterConfig.Host, masterConfig.Port, masterConfig.Dbname, masterConfig.Username)

	db := postgres.InitializeDBInstance(masterConfig, &slavesConfig)
	if db == nil {
		log.Printf("Failed to connect to the database - DBCluster is nil")
		return fmt.Errorf("failed to connect to the database - DBCluster is nil")
	}
	DBCluster = db
	log.Println("Database connected successfully!")
	return nil
}
