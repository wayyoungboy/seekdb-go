# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

seekdb-go is a Go SDK for OceanBase seekdb, an AI-native search database that unifies relational, vector, text, JSON, and GIS data models. This SDK provides a Go idiomatic interface similar to pyseekdb (Python) and seekdb-js (JavaScript).

## Architecture

### Core Components

**Two Client Types:**
- `AdminClient`: Database management operations (create/get/list/delete databases)
- `Client`: Collection and data operations (collections, DML, DQL)

**Connection Modes:**
- **Embedded mode**: seekdb runs in-process (Linux only, glibc >= 2.28)
- **Server mode**: Connect to remote seekdb or OceanBase Database via MySQL protocol (all platforms)

### Package Structure

```
seekdb-go/
├── admin.go          # AdminClient for database management
├── client.go         # Client for collection/data operations
├── collection.go     # Collection operations (create/get/delete/fork)
├── dml.go            # Data manipulation (add/update/upsert/delete)
├── dql.go            # Data query (query/get/hybrid_search)
├── config.go         # Configuration types (HNSWConfig, VectorIndexConfig, etc.)
├── embedding.go      # Embedding function interface and providers
├── filter.go         # Filter operators and where clause handling
├── connection.go     # Connection handling (embedded/server modes)
├── errors.go         # Error types and handling
├── types.go          # Common types (Document, Metadata, QueryResult, etc.)
└── examples/         # Example programs
```

### API Structure (Go Idiomatic Naming)

**Database APIs (AdminClient):**
- `CreateDatabase(name string) error`
- `GetDatabase(name string) (*DatabaseInfo, error)`
- `ListDatabases(limit, offset int) ([]DatabaseInfo, error)`
- `DeleteDatabase(name string) error`

**Collection APIs (Client):**
- `CreateCollection(name string, config CollectionConfig) (*Collection, error)`
- `GetCollection(name string) (*Collection, error)`
- `GetOrCreateCollection(name string, config CollectionConfig) (*Collection, error)`
- `ListCollections() ([]CollectionInfo, error)`
- `CountCollections() (int, error)`
- `HasCollection(name string) (bool, error)`
- `DeleteCollection(name string) error`

**DML APIs (Collection):**
- `Add(ids, documents, embeddings, metadatas) error`
- `Update(ids, documents, embeddings, metadatas) error`
- `Upsert(ids, documents, embeddings, metadatas) error`
- `Delete(ids, where, whereDocument) error`

**DQL APIs (Collection):**
- `Query(queryEmbeddings, nResults, where, whereDocument, include) (*QueryResult, error)`
- `Get(ids, where, whereDocument, limit, offset, include) (*GetResult, error)`
- `HybridSearch(query, knn, rank, nResults, include) (*QueryResult, error)`
- `Count() (int, error)`
- `Peek(limit int) (*GetResult, error)`

## Build Commands

```bash
# Build the SDK
go build ./...

# Run all tests
go test ./...

# Run tests for a specific package
go test -v ./collection

# Run a single test
go test -v -run TestCreateCollection ./collection

# Run with coverage
go test -cover ./...

# Format code
go fmt ./...

# Static analysis
go vet ./...

# Lint (requires golangci-lint)
golangci-lint run
```

## Dependencies

- `github.com/go-sql-driver/mysql` - MySQL driver for server mode connections
- Embedded seekdb binary (Linux only) - bundled with the SDK

## Connection Patterns

### Embedded Mode (Linux)
```go
admin := seekdb.NewAdminClient(seekdb.AdminConfig{Path: "./seekdb"})
client := seekdb.NewClient(seekdb.ClientConfig{Path: "./seekdb", Database: "test"})
```

### Server Mode (All Platforms)
```go
admin := seekdb.NewAdminClient(seekdb.AdminConfig{
    Host:     "127.0.0.1",
    Port:     2881,
    User:     "root",
    Password: "",
})

client := seekdb.NewClient(seekdb.ClientConfig{
    Host:     "127.0.0.1",
    Port:     2881,
    Database: "test",
    User:     "root",
    Password: "",
})
```

### OceanBase Database
```go
client := seekdb.NewClient(seekdb.ClientConfig{
    Host:     "127.0.0.1",
    Port:     2881,
    Tenant:   "test",
    Database: "test",
    User:     "root",
    Password: "",
})
```

## Key Types

### Vector Index Configuration
- `HNSWConfig`: HNSW index with dimension, distance metric (cosine/l2/ip), M, efConstruction
- `IVFConfig`: IVF index with dimension, distance metric, nlist
- Distance metrics: `DistanceCosine`, `DistanceL2`, `DistanceIP`

### Filter Operators
- Comparison: `$eq`, `$ne`, `$gt`, `$gte`, `$lt`, `$lte`
- Membership: `$in`, `$nin`
- Logical: `$and`, `$or`, `$not`
- Text: `$contains`, `$regex`

### Query Results
```go
type QueryResult struct {
    IDs        [][]string
    Documents  [][]string
    Embeddings [][][]float32
    Metadatas  [][]map[string]interface{}
    Distances  [][]float32
}
```

## Reference Implementations

This SDK follows the API design of:
- **pyseekdb**: Python SDK at `pip install pyseekdb`
- **seekdb-js**: JavaScript SDK at `npm install seekdb`

Key differences for Go:
- PascalCase method names (Go convention)
- Strong typing with interfaces for embedding functions
- Error returns instead of exceptions
- Context support for cancellation and timeouts

## seekdb Protocol Notes

- seekdb is MySQL-compatible (port 2881 default)
- Uses MySQL protocol for server mode connections
- Supports vector columns with `VECTOR(dim)` type
- Built-in AI functions: `AI_EMBED`, `AI_COMPLETE`, `AI_RERANK`
- Hybrid search via SQL with BM25 + vector similarity + RRF