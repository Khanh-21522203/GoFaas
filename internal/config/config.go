package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Storage  StorageConfig
	Worker   WorkerConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Addr string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	Type    string // "local" or "s3"
	BaseDir string // For local storage
}

// WorkerConfig holds worker configuration
type WorkerConfig struct {
	ID           string
	WorkDir      string
	RuntimeType  string // "simple" or "container"
	UseContainer bool   // Enable container-based execution
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Addr: getEnv("SERVER_ADDR", ":8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Database: getEnv("DB_NAME", "faas"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Storage: StorageConfig{
			Type:    getEnv("STORAGE_TYPE", "local"),
			BaseDir: getEnv("STORAGE_BASE_DIR", "./storage/functions"),
		},
		Worker: WorkerConfig{
			ID:           getEnv("WORKER_ID", fmt.Sprintf("worker-%d", time.Now().Unix())),
			WorkDir:      getEnv("WORKER_WORK_DIR", "./storage/work"),
			RuntimeType:  getEnv("WORKER_RUNTIME_TYPE", "container"),
			UseContainer: getEnvBool("WORKER_USE_CONTAINER", true),
		},
	}

	return cfg, nil
}

// GetDSN returns the database connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an integer environment variable or returns a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBool gets a boolean environment variable or returns a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

type SecurityConfig struct {
	JWTSecret     string        `env:"JWT_SECRET" envDefault:"change-me-in-production"`
	TokenDuration time.Duration `env:"TOKEN_DURATION" envDefault:"24h"`
}

type RateLimitConfig struct {
	Enabled           bool          `env:"RATE_LIMIT_ENABLED" envDefault:"true"`
	RequestsPerWindow int           `env:"RATE_LIMIT_REQUESTS" envDefault:"100"`
	WindowDuration    time.Duration `env:"RATE_LIMIT_WINDOW" envDefault:"1m"`
}

type CORSConfig struct {
	Enabled          bool     `env:"CORS_ENABLED" envDefault:"true"`
	AllowedOrigins   []string `env:"CORS_ALLOWED_ORIGINS" envDefault:"*"`
	AllowCredentials bool     `env:"CORS_ALLOW_CREDENTIALS" envDefault:"true"`
	MaxAge           int      `env:"CORS_MAX_AGE" envDefault:"3600"`
}
