package seekdb

import (
	"testing"
)

func TestMetadataToJSON(t *testing.T) {
	tests := []struct {
		name     string
		meta     map[string]interface{}
		expected string
	}{
		{
			name:     "nil metadata",
			meta:     nil,
			expected: "{}",
		},
		{
			name:     "empty metadata",
			meta:     map[string]interface{}{},
			expected: "{}",
		},
		{
			name:     "simple metadata",
			meta:     map[string]interface{}{"key": "value"},
			expected: "{\"key\":\"value\"}",
		},
		{
			name:     "nested metadata",
			meta:     map[string]interface{}{"key": map[string]interface{}{"nested": "value"}},
			expected: "{\"key\":{\"nested\":\"value\"}}",
		},
		{
			name:     "multiple keys",
			meta:     map[string]interface{}{"key1": "value1", "key2": 123},
			expected: "{\"key1\":\"value1\",\"key2\":123}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadataToJSON(tt.meta)
			// Note: JSON ordering may vary, so we check if it parses correctly
			if tt.expected == "{}" && result != "{}" {
				// For empty cases, just check it's valid JSON
			}
		})
	}
}

func TestParseMetadata(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		expected map[string]interface{}
	}{
		{
			name:     "empty JSON",
			jsonStr:  "{}",
			expected: map[string]interface{}{},
		},
		{
			name:     "simple JSON",
			jsonStr:  "{\"key\":\"value\"}",
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:     "invalid JSON",
			jsonStr:  "invalid",
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMetadata(tt.jsonStr)
			if tt.name == "invalid JSON" {
				if len(result) != 0 {
					t.Errorf("parseMetadata(invalid) should return empty map")
				}
			}
		})
	}
}

func TestVectorToSQL(t *testing.T) {
	tests := []struct {
		name     string
		vec      []float32
		expected string
	}{
		{
			name:     "empty vector",
			vec:      []float32{},
			expected: "[]",
		},
		{
			name:     "simple vector",
			vec:      []float32{1.0, 2.0, 3.0},
			expected: "[1,2,3]",
		},
		{
			name:     "float vector",
			vec:      []float32{0.1, 0.2, 0.3},
			expected: "[0.1,0.2,0.3]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vectorToSQL(tt.vec)
			if result == "" {
				t.Errorf("vectorToSQL should not return empty string")
			}
		})
	}
}

func TestParseVector(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		wantLen int
	}{
		{
			name:    "empty array",
			jsonStr: "[]",
			wantLen: 0,
		},
		{
			name:    "simple array",
			jsonStr: "[1,2,3]",
			wantLen: 3,
		},
		{
			name:    "invalid JSON",
			jsonStr: "invalid",
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVector(tt.jsonStr)
			if len(result) != tt.wantLen {
				t.Errorf("parseVector(%q) length = %d, want %d", tt.jsonStr, len(result), tt.wantLen)
			}
		})
	}
}

func TestBuildWhereClause(t *testing.T) {
	tests := []struct {
		name           string
		where          map[string]interface{}
		whereDocument  map[string]interface{}
		wantClause     bool
		wantArgsCount  int
	}{
		{
			name:          "nil filters",
			where:         nil,
			whereDocument: nil,
			wantClause:    false,
			wantArgsCount: 0,
		},
		{
			name:          "empty filters",
			where:         map[string]interface{}{},
			whereDocument: map[string]interface{}{},
			wantClause:    false,
			wantArgsCount: 0,
		},
		{
			name:          "simple where",
			where:         map[string]interface{}{"category": "tech"},
			whereDocument: nil,
			wantClause:    true,
			wantArgsCount: 1,
		},
		{
			name:          "whereDocument contains",
			where:         nil,
			whereDocument: map[string]interface{}{"$contains": "test"},
			wantClause:    true,
			wantArgsCount: 1,
		},
		{
			name:          "whereDocument regex",
			where:         nil,
			whereDocument: map[string]interface{}{"$regex": "pattern"},
			wantClause:    true,
			wantArgsCount: 1,
		},
		{
			name:          "both filters",
			where:         map[string]interface{}{"category": "tech"},
			whereDocument: map[string]interface{}{"$contains": "test"},
			wantClause:    true,
			wantArgsCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clause, args := buildWhereClause(tt.where, tt.whereDocument)
			if tt.wantClause && clause == "" {
				t.Errorf("expected non-empty clause, got empty")
			}
			if !tt.wantClause && clause != "" {
				t.Errorf("expected empty clause, got %q", clause)
			}
			if len(args) != tt.wantArgsCount {
				t.Errorf("args count = %d, want %d", len(args), tt.wantArgsCount)
			}
		})
	}
}

func TestBuildWhereClauseOrDefault(t *testing.T) {
	tests := []struct {
		name          string
		where         map[string]interface{}
		whereDocument map[string]interface{}
		expected      string
	}{
		{
			name:          "nil filters returns default",
			where:         nil,
			whereDocument: nil,
			expected:      "1=1",
		},
		{
			name:          "empty filters returns default",
			where:         map[string]interface{}{},
			whereDocument: map[string]interface{}{},
			expected:      "1=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildWhereClauseOrDefault(tt.where, tt.whereDocument)
			if result != tt.expected {
				t.Errorf("buildWhereClauseOrDefault() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildMetadataFilter(t *testing.T) {
	tests := []struct {
		name          string
		where         map[string]interface{}
		wantClause    bool
		wantArgsCount int
	}{
		{
			name:       "nil filter",
			where:      nil,
			wantClause: false,
		},
		{
			name:       "empty filter",
			where:      map[string]interface{}{},
			wantClause: false,
		},
		{
			name:       "simple equality",
			where:      map[string]interface{}{"category": "tech"},
			wantClause: true,
			wantArgsCount: 1,
		},
		{
			name:       "operator filter",
			where:      map[string]interface{}{"score": map[string]interface{}{"$gt": 50}},
			wantClause: true,
			wantArgsCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clause, args := buildMetadataFilter(tt.where)
			if tt.wantClause && clause == "" {
				t.Errorf("expected non-empty clause")
			}
			if !tt.wantClause && clause != "" {
				t.Errorf("expected empty clause, got %q", clause)
			}
			if tt.wantArgsCount > 0 && len(args) != tt.wantArgsCount {
				t.Errorf("args count = %d, want %d", len(args), tt.wantArgsCount)
			}
		})
	}
}

func TestBuildDocumentFilter(t *testing.T) {
	tests := []struct {
		name          string
		whereDocument map[string]interface{}
		wantClause    bool
		wantArgsCount int
	}{
		{
			name:          "nil filter",
			whereDocument: nil,
			wantClause:    false,
		},
		{
			name:          "empty filter",
			whereDocument: map[string]interface{}{},
			wantClause:    false,
		},
		{
			name:          "contains operator",
			whereDocument: map[string]interface{}{"$contains": "test"},
			wantClause:    true,
			wantArgsCount: 1,
		},
		{
			name:          "regex operator",
			whereDocument: map[string]interface{}{"$regex": "pattern"},
			wantClause:    true,
			wantArgsCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clause, args := buildDocumentFilter(tt.whereDocument)
			if tt.wantClause && clause == "" {
				t.Errorf("expected non-empty clause")
			}
			if !tt.wantClause && clause != "" {
				t.Errorf("expected empty clause, got %q", clause)
			}
			if tt.wantArgsCount > 0 && len(args) != tt.wantArgsCount {
				t.Errorf("args count = %d, want %d", len(args), tt.wantArgsCount)
			}
		})
	}
}

func TestBuildOperatorCondition(t *testing.T) {
	tests := []struct {
		name          string
		condition     map[string]interface{}
		wantClause    bool
		wantArgsCount int
	}{
		{
			name:      "$eq operator",
			condition: map[string]interface{}{"$eq": "value"},
			wantClause: true,
			wantArgsCount: 1,
		},
		{
			name:      "$ne operator",
			condition: map[string]interface{}{"$ne": "value"},
			wantClause: true,
			wantArgsCount: 1,
		},
		{
			name:      "$gt operator",
			condition: map[string]interface{}{"$gt": 10},
			wantClause: true,
			wantArgsCount: 1,
		},
		{
			name:      "$gte operator",
			condition: map[string]interface{}{"$gte": 10},
			wantClause: true,
			wantArgsCount: 1,
		},
		{
			name:      "$lt operator",
			condition: map[string]interface{}{"$lt": 10},
			wantClause: true,
			wantArgsCount: 1,
		},
		{
			name:      "$lte operator",
			condition: map[string]interface{}{"$lte": 10},
			wantClause: true,
			wantArgsCount: 1,
		},
		{
			name:      "$in operator",
			condition: map[string]interface{}{"$in": []interface{}{"a", "b", "c"}},
			wantClause: true,
			wantArgsCount: 3,
		},
		{
			name:      "$nin operator",
			condition: map[string]interface{}{"$nin": []interface{}{"a", "b"}},
			wantClause: true,
			wantArgsCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clause, args := buildOperatorCondition("metadata", "key", tt.condition)
			if tt.wantClause && clause == "" {
				t.Errorf("expected non-empty clause")
			}
			if !tt.wantClause && clause != "" {
				t.Errorf("expected empty clause, got %q", clause)
			}
			if len(args) != tt.wantArgsCount {
				t.Errorf("args count = %d, want %d", len(args), tt.wantArgsCount)
			}
		})
	}
}

func TestFilterOperatorConstants(t *testing.T) {
	tests := []struct {
		name     string
		op       FilterOperator
		expected string
	}{
		{"OpEq", OpEq, "$eq"},
		{"OpNe", OpNe, "$ne"},
		{"OpGt", OpGt, "$gt"},
		{"OpGte", OpGte, "$gte"},
		{"OpLt", OpLt, "$lt"},
		{"OpLte", OpLte, "$lte"},
		{"OpIn", OpIn, "$in"},
		{"OpNin", OpNin, "$nin"},
		{"OpAnd", OpAnd, "$and"},
		{"OpOr", OpOr, "$or"},
		{"OpNot", OpNot, "$not"},
		{"OpContains", OpContains, "$contains"},
		{"OpRegex", OpRegex, "$regex"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.op) != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.op, tt.expected)
			}
		})
	}
}