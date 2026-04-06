package seekdb

// VectorIndexConfig defines the interface for vector index configurations.
type VectorIndexConfig interface {
	Type() string
	GetDimension() int
	GetDistanceMetric() DistanceMetric
}

// HNSWConfig holds HNSW (Hierarchical Navigable Small World) index configuration.
// HNSW is an in-memory index optimized for high recall and fast search.
type HNSWConfig struct {
	Dimension      int
	DistanceMetric DistanceMetric
	M              int // Number of connections per node (default: 16)
	EfConstruction int // Construction time/accuracy trade-off (default: 200)
	EfSearch       int // Search time/accuracy trade-off (default: 50)
}

func (c *HNSWConfig) Type() string {
	return "HNSW"
}

func (c *HNSWConfig) GetDimension() int {
	return c.Dimension
}

func (c *HNSWConfig) GetDistanceMetric() DistanceMetric {
	return c.DistanceMetric
}

// HNSWSQConfig holds HNSW with Scalar Quantization configuration.
// SQ reduces memory usage by quantizing vectors to 8-bit integers.
type HNSWSQConfig struct {
	Dimension      int
	DistanceMetric DistanceMetric
	M              int
	EfConstruction int
	EfSearch       int
}

func (c *HNSWSQConfig) Type() string {
	return "HNSW_SQ"
}

func (c *HNSWSQConfig) GetDimension() int {
	return c.Dimension
}

func (c *HNSWSQConfig) GetDistanceMetric() DistanceMetric {
	return c.DistanceMetric
}

// HNSWBQConfig holds HNSW with Binary Quantization configuration.
// BQ further reduces memory usage with binary quantization.
type HNSWBQConfig struct {
	Dimension      int
	DistanceMetric DistanceMetric
	M              int
	EfConstruction int
	EfSearch       int
}

func (c *HNSWBQConfig) Type() string {
	return "HNSW_BQ"
}

func (c *HNSWBQConfig) GetDimension() int {
	return c.Dimension
}

func (c *HNSWBQConfig) GetDistanceMetric() DistanceMetric {
	return c.DistanceMetric
}

// IVFConfig holds IVF (Inverted File) index configuration.
// IVF is a disk-based index suitable for large datasets.
type IVFConfig struct {
	Dimension      int
	DistanceMetric DistanceMetric
	Nlist          int // Number of clusters (default: 100)
}

func (c *IVFConfig) Type() string {
	return "IVF"
}

func (c *IVFConfig) GetDimension() int {
	return c.Dimension
}

func (c *IVFConfig) GetDistanceMetric() DistanceMetric {
	return c.DistanceMetric
}

// IVFPQConfig holds IVF with Product Quantization configuration.
// PQ further compresses vectors for lower memory footprint.
type IVFPQConfig struct {
	Dimension      int
	DistanceMetric DistanceMetric
	Nlist          int
	M              int // Number of subquantizers
	Nbits          int // Number of bits per subquantizer (default: 8)
}

func (c *IVFPQConfig) Type() string {
	return "IVF_PQ"
}

func (c *IVFPQConfig) GetDimension() int {
	return c.Dimension
}

func (c *IVFPQConfig) GetDistanceMetric() DistanceMetric {
	return c.DistanceMetric
}

// DefaultHNSWConfig returns an HNSW config with sensible defaults.
func DefaultHNSWConfig(dimension int, distance DistanceMetric) *HNSWConfig {
	if distance == "" {
		distance = DistanceCosine
	}
	return &HNSWConfig{
		Dimension:      dimension,
		DistanceMetric: distance,
		M:              16,
		EfConstruction: 200,
		EfSearch:       50,
	}
}

// DefaultIVFConfig returns an IVF config with sensible defaults.
func DefaultIVFConfig(dimension int, distance DistanceMetric) *IVFConfig {
	if distance == "" {
		distance = DistanceCosine
	}
	return &IVFConfig{
		Dimension:      dimension,
		DistanceMetric: distance,
		Nlist:          100,
	}
}
