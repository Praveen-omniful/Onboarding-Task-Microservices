package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v7"
)

type Config struct {
	Env      string `env:"ENV" envDefault:"development"`
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
}

type ServerConfig struct {
	Port                    string        `env:"SERVER_PORT" envDefault:":8080"`
	GracefulShutdownTimeout time.Duration `env:"GRACEFUL_SHUTDOWN_TIMEOUT" envDefault:"10s"`
}

type DatabaseConfig struct {
	Host     string `env:"DB_HOST" envDefault:"localhost"`
	Port     string `env:"DB_PORT" envDefault:"5432"`
	User     string `env:"DB_USER" envDefault:"postgres"`
	Password string `env:"DB_PASSWORD" envDefault:"postgres"`
	DBName   string `env:"DB_NAME" envDefault:"ims"`
	SSLMode  string `env:"DB_SSLMODE" envDefault:"disable"`
}

type RedisConfig struct {
	Address  string `env:"REDIS_ADDRESS" envDefault:"localhost:6379"`
	Password string `env:"REDIS_PASSWORD" envDefault:""`
	DB       int    `env:"REDIS_DB" envDefault:"0"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	// Parse nested structs
	if err := env.Parse(&cfg.Server); err != nil {
		return nil, err
	}

	if err := env.Parse(&cfg.Database); err != nil {
		return nil, err
	}

	if err := env.Parse(&cfg.Redis); err != nil {
		return nil, err
	}

	return cfg, nil
}

// GetConnectionString returns the PostgreSQL connection string
func (d *DatabaseConfig) GetConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode)
}
