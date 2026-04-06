package seekdb

import (
	"testing"
)

func TestHNSWConfig(t *testing.T) {
	config := &HNSWConfig{
		Dimension:      128,
		DistanceMetric: DistanceCosine,
		M:              16,
		EfConstruction: 200,
		EfSearch:       50,
	}

	if config.Type() != "HNSW" {
		t.Errorf("Type() = %q, want %q", config.Type(), "HNSW")
	}
	if config.GetDimension() != 128 {
		t.Errorf("GetDimension() = %d, want %d", config.GetDimension(), 128)
	}
	if config.GetDistanceMetric() != DistanceCosine {
		t.Errorf("GetDistanceMetric() = %q, want %q", config.GetDistanceMetric(), DistanceCosine)
	}
}

func TestHNSWSQConfig(t *testing.T) {
	config := &HNSWSQConfig{
		Dimension:      256,
		DistanceMetric: DistanceL2,
		M:              32,
		EfConstruction: 400,
		EfSearch:       100,
	}

	if config.Type() != "HNSW_SQ" {
		t.Errorf("Type() = %q, want %q", config.Type(), "HNSW_SQ")
	}
	if config.GetDimension() != 256 {
		t.Errorf("GetDimension() = %d, want %d", config.GetDimension(), 256)
	}
	if config.GetDistanceMetric() != DistanceL2 {
		t.Errorf("GetDistanceMetric() = %q, want %q", config.GetDistanceMetric(), DistanceL2)
	}
}

func TestHNSWBQConfig(t *testing.T) {
	config := &HNSWBQConfig{
		Dimension:      512,
		DistanceMetric: DistanceIP,
		M:              24,
		EfConstruction: 300,
		EfSearch:       75,
	}

	if config.Type() != "HNSW_BQ" {
		t.Errorf("Type() = %q, want %q", config.Type(), "HNSW_BQ")
	}
	if config.GetDimension() != 512 {
		t.Errorf("GetDimension() = %d, want %d", config.GetDimension(), 512)
	}
	if config.GetDistanceMetric() != DistanceIP {
		t.Errorf("GetDistanceMetric() = %q, want %q", config.GetDistanceMetric(), DistanceIP)
	}
}

func TestIVFConfig(t *testing.T) {
	config := &IVFConfig{
		Dimension:      128,
		DistanceMetric: DistanceCosine,
		Nlist:          100,
	}

	if config.Type() != "IVF" {
		t.Errorf("Type() = %q, want %q", config.Type(), "IVF")
	}
	if config.GetDimension() != 128 {
		t.Errorf("GetDimension() = %d, want %d", config.GetDimension(), 128)
	}
	if config.GetDistanceMetric() != DistanceCosine {
		t.Errorf("GetDistanceMetric() = %q, want %q", config.GetDistanceMetric(), DistanceCosine)
	}
}

func TestIVFPQConfig(t *testing.T) {
	config := &IVFPQConfig{
		Dimension:      128,
		DistanceMetric: DistanceL2,
		Nlist:          100,
		M:              8,
		Nbits:          8,
	}

	if config.Type() != "IVF_PQ" {
		t.Errorf("Type() = %q, want %q", config.Type(), "IVF_PQ")
	}
	if config.GetDimension() != 128 {
		t.Errorf("GetDimension() = %d, want %d", config.GetDimension(), 128)
	}
	if config.GetDistanceMetric() != DistanceL2 {
		t.Errorf("GetDistanceMetric() = %q, want %q", config.GetDistanceMetric(), DistanceL2)
	}
}

func TestDefaultHNSWConfig(t *testing.T) {
	config := DefaultHNSWConfig(128, DistanceCosine)

	if config.Dimension != 128 {
		t.Errorf("Dimension = %d, want %d", config.Dimension, 128)
	}
	if config.DistanceMetric != DistanceCosine {
		t.Errorf("DistanceMetric = %q, want %q", config.DistanceMetric, DistanceCosine)
	}
	if config.M != 16 {
		t.Errorf("M = %d, want %d (default)", config.M, 16)
	}
	if config.EfConstruction != 200 {
		t.Errorf("EfConstruction = %d, want %d (default)", config.EfConstruction, 200)
	}
	if config.EfSearch != 50 {
		t.Errorf("EfSearch = %d, want %d (default)", config.EfSearch, 50)
	}
}

func TestDefaultHNSWConfigEmptyDistance(t *testing.T) {
	config := DefaultHNSWConfig(256, "")

	if config.DistanceMetric != DistanceCosine {
		t.Errorf("DistanceMetric should default to cosine when empty, got %q", config.DistanceMetric)
	}
}

func TestDefaultIVFConfig(t *testing.T) {
	config := DefaultIVFConfig(128, DistanceL2)

	if config.Dimension != 128 {
		t.Errorf("Dimension = %d, want %d", config.Dimension, 128)
	}
	if config.DistanceMetric != DistanceL2 {
		t.Errorf("DistanceMetric = %q, want %q", config.DistanceMetric, DistanceL2)
	}
	if config.Nlist != 100 {
		t.Errorf("Nlist = %d, want %d (default)", config.Nlist, 100)
	}
}

func TestDefaultIVFConfigEmptyDistance(t *testing.T) {
	config := DefaultIVFConfig(256, "")

	if config.DistanceMetric != DistanceCosine {
		t.Errorf("DistanceMetric should default to cosine when empty, got %q", config.DistanceMetric)
	}
}

func TestVectorIndexConfigInterface(t *testing.T) {
	// Test that all config types implement the interface
	var configs []VectorIndexConfig = []VectorIndexConfig{
		&HNSWConfig{Dimension: 128, DistanceMetric: DistanceCosine},
		&HNSWSQConfig{Dimension: 128, DistanceMetric: DistanceCosine},
		&HNSWBQConfig{Dimension: 128, DistanceMetric: DistanceCosine},
		&IVFConfig{Dimension: 128, DistanceMetric: DistanceCosine},
		&IVFPQConfig{Dimension: 128, DistanceMetric: DistanceCosine},
	}

	for i, config := range configs {
		if config.GetDimension() != 128 {
			t.Errorf("config[%d].GetDimension() = %d, want 128", i, config.GetDimension())
		}
		if config.GetDistanceMetric() != DistanceCosine {
			t.Errorf("config[%d].GetDistanceMetric() = %q, want %q", i, config.GetDistanceMetric(), DistanceCosine)
		}
	}
}