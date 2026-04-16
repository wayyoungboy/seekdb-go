# seekdb-go

> **Disclaimer**: This is a community-maintained project and is NOT officially affiliated with or endorsed by OceanBase. For the official SDKs, see [pyseekdb](https://github.com/oceanbase/pyseekdb) (Python) and [seekdb-js](https://github.com/oceanbase/seekdb-js) (JavaScript).

A Go SDK for OceanBase seekdb, an AI-native search database that unifies relational, vector, text, JSON, and GIS data models.

## Embedded Mode Status

Embedded mode uses CGo bindings to `libseekdb.so` (same approach as pyseekdb/seekdb-js). The CGo implementation is complete and aligned with pyseekdb's singleton pattern (single open, no close, reference counting).

**Blocker**: `libseekdb.so` is not currently obtainable — the S3 download URL returns 403, RPM/DEB packages contain only the server binary, and pyseekdb wheels bundle the engine statically. See the full Embedded Mode section below for details.

Individual embedded tests pass when run in isolation:

| Test | Status |
|------|--------|
| AdminClient (standalone) | PASS |
| Client (standalone) | PASS |
| AutoPort (standalone) | PASS |
| AdminClient → Client (same process) | HANGS (known issue) |

Server mode is fully functional and recommended for production use.

## Features

- **Unified API** - Single SDK for both embedded (Linux) and server modes
- **Vector Similarity Search** - Efficient HNSW and IVF index support
- **Hybrid Search** - Combine vector and full-text search with RRF ranking
- **Multiple Embedding Providers** - OpenAI, Azure, Cohere, HuggingFace, Qwen, DeepSeek, Jina AI
- **Collection Management** - Create, update, delete, and fork collections
- **Filter Support** - Rich filtering with metadata and document filters
- **Include Options** - Control which fields are returned in queries
- **QueryTexts** - Auto-embed text queries when using embedding functions
- **Lazy Connection** - Connections are established on first use
- **Environment Variables** - Configure via environment variables
- **Type-Safe** - Full Go type safety with idiomatic API design

## Installation

```bash
go get github.com/oceanbase/seekdb-go
```

## Quick Start

### Server Mode

```go
package main

import (
    "context"
    "fmt"
    "log"

    seekdb "github.com/oceanbase/seekdb-go"
)

func main() {
    ctx := context.Background()

    // Create admin client
    admin, err := seekdb.NewAdminClient(seekdb.AdminConfig{
        Host: "127.0.0.1",
        Port: 2881,
        User: "root",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer admin.Close()

    // Create database
    admin.CreateDatabase(ctx, "mydb")

    // Create data client
    client, err := seekdb.NewClient(seekdb.ClientConfig{
        Host:     "127.0.0.1",
        Port:     2881,
        Database: "mydb",
        User:     "root",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Create collection with HNSW index
    collection, err := client.CreateCollection(ctx, "documents", seekdb.CollectionConfig{
        Dimension:      128,
        DistanceMetric: seekdb.DistanceCosine,
        VectorIndex:    seekdb.DefaultHNSWConfig(128, seekdb.DistanceCosine),
    })
    if err != nil {
        log.Fatal(err)
    }

    // Add documents
    err = collection.Add(ctx, seekdb.AddParams{
        IDs:        []string{"doc1", "doc2"},
        Documents:  []string{"Hello world", "Vector search"},
        Embeddings: [][]float32{embed1, embed2},
        Metadatas: []map[string]interface{}{
            {"category": "greeting"},
            {"category": "tech"},
        },
    })

    // Query similar documents
    results, err := collection.Query(ctx, seekdb.QueryParams{
        QueryEmbeddings: [][]float32{queryEmb},
        NResults:        10,
    })

    for i, id := range results.IDs[0] {
        fmt.Printf("%d. %s (distance: %.4f)\n", i+1, id, results.Distances[0][i])
    }
}
```

### With Embedding Function

```go
// Create collection with embedding function
collection, err := client.CreateCollection(ctx, "docs", seekdb.CollectionConfig{
    Dimension:         1536,
    DistanceMetric:    seekdb.DistanceCosine,
    EmbeddingFunction: seekdb.NewOpenAIEmbeddingFunction("text-embedding-3-small", "your-api-key"),
})

// Add documents without embeddings (auto-generated)
collection.Add(ctx, seekdb.AddParams{
    IDs:       []string{"doc1"},
    Documents: []string{"This document will be auto-embedded"},
})
```

## API Reference

### AdminClient

```go
admin, _ := seekdb.NewAdminClient(seekdb.AdminConfig{
    Host: "127.0.0.1",
    Port: 2881,
    User: "root",
})

admin.CreateDatabase(ctx, "mydb")
admin.GetDatabase(ctx, "mydb")
admin.ListDatabases(ctx, 10, 0)
admin.DeleteDatabase(ctx, "mydb")
```

### Client

```go
client, _ := seekdb.NewClient(seekdb.ClientConfig{
    Host:     "127.0.0.1",
    Port:     2881,
    Database: "mydb",
    User:     "root",
})

client.CreateCollection(ctx, "mycollection", config)
client.GetCollection(ctx, "mycollection")
client.GetOrCreateCollection(ctx, "mycollection", config)
client.ListCollections(ctx)
client.HasCollection(ctx, "mycollection")
client.DeleteCollection(ctx, "mycollection")
client.ForkCollection(ctx, "source", seekdb.ForkOptions{Name: "clone"})
```

### Collection

```go
// Data operations
collection.Add(ctx, seekdb.AddParams{...})
collection.Update(ctx, seekdb.UpdateParams{...})
collection.Upsert(ctx, seekdb.UpsertParams{...})
collection.Delete(ctx, seekdb.DeleteParams{...})

// Query operations
collection.Query(ctx, seekdb.QueryParams{...})
collection.Get(ctx, seekdb.GetParams{...})
collection.HybridSearch(ctx, seekdb.HybridSearchParams{...})
collection.Count(ctx)
collection.Peek(ctx, 10)

// Collection management
collection.Fork(ctx, seekdb.ForkOptions{...})
collection.Modify(ctx, seekdb.ModifyOptions{...})
collection.Describe(ctx)
```

### Vector Index Configuration

```go
// HNSW (recommended for most use cases)
hnsw := seekdb.DefaultHNSWConfig(128, seekdb.DistanceCosine)
// Or custom:
hnsw := &seekdb.HNSWConfig{
    Dimension:      128,
    DistanceMetric: seekdb.DistanceCosine,
    M:              16,
    EfConstruction: 200,
    EfSearch:       50,
}

// IVF (for larger datasets)
ivf := seekdb.DefaultIVFConfig(128, seekdb.DistanceL2)
```

### Distance Metrics

- `seekdb.DistanceCosine` - Cosine similarity (default)
- `seekdb.DistanceL2` - Euclidean distance
- `seekdb.DistanceIP` - Inner product

### Filter Operators

```go
// Metadata filters
where := map[string]interface{}{
    "category": "tech",                    // equality
    "score":   map[string]interface{}{"$gt": 50},  // comparison
    "tags":    map[string]interface{}{"$in": []string{"a", "b"}},
}

// Document content filters
whereDocument := map[string]interface{}{
    "$contains": "search term",
    "$regex":    "pattern.*",
}
```

### Embedding Functions

```go
// OpenAI
openai := seekdb.NewOpenAIEmbeddingFunction("text-embedding-3-small", "api-key")

// Azure OpenAI
azure := seekdb.NewAzureOpenAIEmbeddingFunction(endpoint, deployment, "api-key", 1536)

// Cohere
cohere := seekdb.NewCohereEmbeddingFunction("embed-english-v3.0", "api-key")

// HuggingFace
hf := seekdb.NewHuggingFaceEmbeddingFunction("sentence-transformers/all-MiniLM-L6-v2", "api-key")

// Qwen/DashScope
qwen := seekdb.NewDashScopeEmbeddingFunction("text-embedding-v3", "api-key")

// DeepSeek
deepseek := seekdb.NewDeepSeekEmbeddingFunction("api-key")

// Jina AI
jina := seekdb.NewJinaEmbeddingFunction("jina-embeddings-v2-base-en", "api-key")

// Custom/Local
local := seekdb.NewLocalEmbeddingFunction(768, myEmbedFunc)
```

## Running Modes

## Embedded Mode (Linux only, CGo)

Embedded mode uses CGo bindings to `libseekdb.so` (same approach as pyseekdb/seekdb-js). The `libseekdb.so` binary is too large for GitHub (>100MB) and must be obtained separately.

### Current Status

The CGo implementation is complete and aligned with pyseekdb's singleton pattern. However, obtaining `libseekdb.so` is currently blocked:

- **S3 URL returns 403** — the official download URL is no longer publicly accessible
- **RPM/DEB packages only contain the server binary**, not the embedded library
- **pyseekdb wheels** bundle the engine statically as a Python C extension (not usable for CGo)

To use embedded mode, you must obtain `libseekdb.so` from the S3 URL (see below) and place it in `libseekdb/`.

### Setup (if you can download libseekdb.so)

1. Download and extract `libseekdb.so` + `seekdb.h`:

```bash
# Linux x64
wget -O libseekdb-linux-x64.zip "https://oceanbase-seekdb-builds.s3.ap-southeast-1.amazonaws.com/libseekdb/all_commits/c1a508a4efed701b88d369c7bdcf2aa2ea3480bd/libseekdb-linux-x64.zip"
unzip -o libseekdb-linux-x64.zip -d libseekdb/
rm libseekdb-linux-x64.zip
```

2. Ensure `libaio-dev` is installed:

```bash
sudo apt-get install -y libaio-dev
```

3. Build with CGo enabled:

```bash
CGO_ENABLED=1 go build ./...
```

### Usage

```go
client, _ := seekdb.NewClient(seekdb.ClientConfig{
    Path:     "./data/seekdb",
    Database: "mydb",
})
```

### Known Issues

- When `AdminClient` and `Client` are used sequentially in embedded mode within the same process, the second client creation may hang. This is a known issue with the C library's global state management. Each embedded instance works correctly in isolation.
- Individual test results (all PASS when run in isolation):

| Test | Status |
|------|--------|
| AdminClient (standalone) | PASS |
| Client (standalone) | PASS |
| AutoPort (standalone) | PASS |
| AdminClient → Client (same process) | HANGS |

Server mode is fully functional and recommended for production use.

### Server Mode (All platforms)

```go
client, _ := seekdb.NewClient(seekdb.ClientConfig{
    Host:     "127.0.0.1",
    Port:     2881,
    Database: "mydb",
    User:     "root",
    Password: "",
})
```

### OceanBase Database

```go
client, _ := seekdb.NewClient(seekdb.ClientConfig{
    Host:     "127.0.0.1",
    Port:     2881,
    Tenant:   "test",
    Database: "mydb",
    User:     "root",
})
```

## Examples

See the [examples](./examples) directory for more usage examples:

- [Simple Example](./examples/simple/main.go) - Basic vector search

## Development

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Format code
go fmt ./...

# Static analysis
go vet ./...
```

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.

## References

- [pyseekdb](https://github.com/oceanbase/pyseekdb) - Python SDK
- [seekdb-js](https://github.com/oceanbase/seekdb-js) - JavaScript SDK
- [OceanBase](https://oceanbase.com) - OceanBase Database