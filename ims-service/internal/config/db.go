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
	masterConfig := postgres.DBConfig{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		Username: cfg.Database.User,
		Password: cfg.Database.Password,
		Dbname:   cfg.Database.DBName,
		// Add these fields to your Config struct if not present:
		// MaxOpenConns:    cfg.Database.MaxOpenConns,
		// MaxIdleConns:    cfg.Database.MaxIdleConns,
		// ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		// DebugMode:       cfg.Database.DebugMode,
	}

	slavesConfig := make([]postgres.DBConfig, 0) // for read replicas, if any

	log.Println("Connecting to the database...")
	db := postgres.InitializeDBInstance(masterConfig, &slavesConfig)
	if db == nil {
		log.Panic("Failed to connect to the database")
		return fmt.Errorf("failed to connect to the database")
	}
	DBCluster = db
	log.Println("Database connected successfully!")
	return nil
}
