// Package seekdb provides a Go SDK for OceanBase seekdb, an AI-native search database.
// It supports both embedded mode (Linux) and server mode (all platforms) connections.
package seekdb

// Version information
const (
	Version = "1.0.0"
	Name    = "seekdb-go"
)

// DistanceMetric defines the distance metric for vector similarity search.
type DistanceMetric string

const (
	DistanceCosine DistanceMetric = "cosine"
	DistanceL2     DistanceMetric = "l2"
	DistanceIP     DistanceMetric = "ip" // inner product
)

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
}
