// Package seekdb provides a Go SDK for OceanBase seekdb, an AI-native search database.
// It supports both embedded mode (Linux) and server mode (all platforms) connections.
package seekdb

import (
	"os"
	"strconv"
	"time"
)

// Version information
const (
	Version = "1.0.0"
	Name    = "seekdb-go"
)

// Environment variable names
const (
	EnvHost     = "SEEKDB_HOST"
	EnvPort     = "SEEKDB_PORT"
	EnvUser     = "SEEKDB_USER"
	EnvPassword = "SEEKDB_PASSWORD"
	EnvTenant   = "SEEKDB_TENANT"
	EnvDatabase = "SEEKDB_DATABASE"

	// Embedding API keys
	EnvOpenAIAPIKey   = "OPENAI_API_KEY"
	EnvCohereAPIKey   = "COHERE_API_KEY"
	EnvHuggingFaceKey = "HUGGINGFACE_API_KEY"
	EnvQwenAPIKey     = "QWEN_API_KEY"
	EnvDeepSeekAPIKey = "DEEPSEEK_API_KEY"
	EnvJinaAPIKey     = "JINA_API_KEY"
)

// DistanceMetric defines the distance metric for vector similarity search.
type DistanceMetric string

const (
	DistanceCosine DistanceMetric = "cosine"
	DistanceL2     DistanceMetric = "l2"
	DistanceIP     DistanceMetric = "ip" // inner product
)

// ConnectionPoolConfig holds connection pool settings.
type ConnectionPoolConfig struct {
	MaxOpenConns    int           // Maximum number of open connections (default: 25)
	MaxIdleConns    int           // Maximum number of idle connections (default: 5)
	ConnMaxLifetime time.Duration // Maximum lifetime of a connection (default: 5 minutes)
	ConnMaxIdleTime time.Duration // Maximum idle time of a connection (default: 10 minutes)
}

// DefaultConnectionPoolConfig returns the default connection pool configuration.
func DefaultConnectionPoolConfig() ConnectionPoolConfig {
	return ConnectionPoolConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}
}

// ClientConfig holds the configuration for connecting to seekdb.
type ClientConfig struct {
	// Embedded mode parameters
	Path string // Path to seekdb data directory (embedded mode)

	// Server mode parameters
	Host     string // Server host address
	Port     int    // Server port (default: 2881)
	User     string // Username (default: "root")
	Password string // Password (can be from SEEKDB_PASSWORD env var)
	Tenant   string // Tenant name (OceanBase Database only)
	Database string // Database name

	// Connection pool settings
	PoolConfig ConnectionPoolConfig
}

// AdminConfig holds the configuration for AdminClient connections.
type AdminConfig struct {
	// Embedded mode parameters
	Path string // Path to seekdb data directory (embedded mode)

	// Server mode parameters
	Host     string // Server host address
	Port     int    // Server port (default: 2881)
	User     string // Username (default: "root")
	Password string // Password (can be from SEEKDB_PASSWORD env var)
	Tenant   string // Tenant name (OceanBase Database only)

	// Connection pool settings
	PoolConfig ConnectionPoolConfig
}

// ClientConfigFromEnv creates a ClientConfig from environment variables.
func ClientConfigFromEnv() ClientConfig {
	port, _ := strconv.Atoi(os.Getenv(EnvPort))
	if port == 0 {
		port = 2881
	}

	return ClientConfig{
		Host:       os.Getenv(EnvHost),
		Port:       port,
		User:       getEnvWithDefault(EnvUser, "root"),
		Password:   os.Getenv(EnvPassword),
		Tenant:     os.Getenv(EnvTenant),
		Database:   getEnvWithDefault(EnvDatabase, "test"),
		PoolConfig: DefaultConnectionPoolConfig(),
	}
}

// AdminConfigFromEnv creates an AdminConfig from environment variables.
func AdminConfigFromEnv() AdminConfig {
	port, _ := strconv.Atoi(os.Getenv(EnvPort))
	if port == 0 {
		port = 2881
	}

	return AdminConfig{
		Host:       os.Getenv(EnvHost),
		Port:       port,
		User:       getEnvWithDefault(EnvUser, "root"),
		Password:   os.Getenv(EnvPassword),
		Tenant:     os.Getenv(EnvTenant),
		PoolConfig: DefaultConnectionPoolConfig(),
	}
}

// getEnvWithDefault returns the environment variable value or a default.
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetAPIKeyFromEnv returns an API key from environment variable with fallback.
func GetAPIKeyFromEnv(envKey, fallback string) string {
	if key := os.Getenv(envKey); key != "" {
		return key
	}
	return fallback
}
