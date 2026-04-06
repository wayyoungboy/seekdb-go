package seekdb

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FilterOperator represents filter operators for where clauses.
type FilterOperator string

const (
	OpEq       FilterOperator = "$eq"
	OpNe       FilterOperator = "$ne"
	OpGt       FilterOperator = "$gt"
	OpGte      FilterOperator = "$gte"
	OpLt       FilterOperator = "$lt"
	OpLte      FilterOperator = "$lte"
	OpIn       FilterOperator = "$in"
	OpNin      FilterOperator = "$nin"
	OpAnd      FilterOperator = "$and"
	OpOr       FilterOperator = "$or"
	OpNot      FilterOperator = "$not"
	OpContains FilterOperator = "$contains"
	OpRegex    FilterOperator = "$regex"
)

// buildWhereClause builds a SQL WHERE clause from filter conditions.
func buildWhereClause(where, whereDocument map[string]interface{}) (string, []interface{}) {
	var clauses []string
	var args []interface{}

	if where != nil {
		clause, clauseArgs := buildMetadataFilter(where)
		if clause != "" {
			clauses = append(clauses, clause)
			args = append(args, clauseArgs...)
		}
	}

	if whereDocument != nil {
		clause, clauseArgs := buildDocumentFilter(whereDocument)
		if clause != "" {
			clauses = append(clauses, clause)
			args = append(args, clauseArgs...)
		}
	}

	if len(clauses) == 0 {
		return "", args
	}

	return strings.Join(clauses, " AND "), args
}

// buildWhereClauseOrDefault builds a WHERE clause or returns "1=1" as default.
func buildWhereClauseOrDefault(where, whereDocument map[string]interface{}) string {
	clause, _ := buildWhereClause(where, whereDocument)
	if clause == "" {
		return "1=1"
	}
	return clause
}

// buildMetadataFilter builds a filter clause for metadata fields.
func buildMetadataFilter(where map[string]interface{}) (string, []interface{}) {
	if where == nil {
		return "", nil
	}

	var clauses []string
	var args []interface{}

	for key, value := range where {
		clause, clauseArgs := buildCondition("metadata", key, value)
		if clause != "" {
			clauses = append(clauses, clause)
			args = append(args, clauseArgs...)
		}
	}

	if len(clauses) == 0 {
		return "", nil
	}

	return strings.Join(clauses, " AND "), args
}

// buildDocumentFilter builds a filter clause for document content.
func buildDocumentFilter(whereDocument map[string]interface{}) (string, []interface{}) {
	if whereDocument == nil {
		return "", nil
	}

	var clauses []string
	var args []interface{}

	for op, value := range whereDocument {
		switch op {
		case "$contains":
			clauses = append(clauses, "document LIKE ?")
			args = append(args, fmt.Sprintf("%%%v%%", value))
		case "$regex":
			clauses = append(clauses, "document REGEXP ?")
			args = append(args, value)
		}
	}

	if len(clauses) == 0 {
		return "", nil
	}

	return strings.Join(clauses, " AND "), args
}

// buildCondition builds a single condition clause.
func buildCondition(column, key string, value interface{}) (string, []interface{}) {
	// Check if value is a map (operator-based condition)
	if m, ok := value.(map[string]interface{}); ok {
		return buildOperatorCondition(column, key, m)
	}

	// Simple equality condition
	return fmt.Sprintf("JSON_EXTRACT(%s, '$.%s') = ?", column, key), []interface{}{value}
}

// buildOperatorCondition builds a condition with an explicit operator.
func buildOperatorCondition(column, key string, condition map[string]interface{}) (string, []interface{}) {
	var clauses []string
	var args []interface{}

	for op, val := range condition {
		jsonPath := fmt.Sprintf("JSON_EXTRACT(%s, '$.%s')", column, key)

		switch op {
		case "$eq":
			clauses = append(clauses, fmt.Sprintf("%s = ?", jsonPath))
			args = append(args, val)
		case "$ne":
			clauses = append(clauses, fmt.Sprintf("%s != ?", jsonPath))
			args = append(args, val)
		case "$gt":
			clauses = append(clauses, fmt.Sprintf("%s > ?", jsonPath))
			args = append(args, val)
		case "$gte":
			clauses = append(clauses, fmt.Sprintf("%s >= ?", jsonPath))
			args = append(args, val)
		case "$lt":
			clauses = append(clauses, fmt.Sprintf("%s < ?", jsonPath))
			args = append(args, val)
		case "$lte":
			clauses = append(clauses, fmt.Sprintf("%s <= ?", jsonPath))
			args = append(args, val)
		case "$in":
			if arr, ok := val.([]interface{}); ok {
				placeholders := make([]string, len(arr))
				for i, v := range arr {
					placeholders[i] = "?"
					args = append(args, v)
				}
				clauses = append(clauses, fmt.Sprintf("%s IN (%s)", jsonPath, strings.Join(placeholders, ", ")))
			}
		case "$nin":
			if arr, ok := val.([]interface{}); ok {
				placeholders := make([]string, len(arr))
				for i, v := range arr {
					placeholders[i] = "?"
					args = append(args, v)
				}
				clauses = append(clauses, fmt.Sprintf("%s NOT IN (%s)", jsonPath, strings.Join(placeholders, ", ")))
			}
		}
	}

	if len(clauses) == 0 {
		return "", nil
	}

	return strings.Join(clauses, " AND "), args
}

// Helper functions for JSON conversion

func metadataToJSON(meta map[string]interface{}) string {
	if meta == nil {
		return "{}"
	}
	bytes, err := json.Marshal(meta)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

func parseMetadata(jsonStr string) map[string]interface{} {
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &meta); err != nil {
		return make(map[string]interface{})
	}
	return meta
}

func vectorToSQL(vec []float32) string {
	bytes, err := json.Marshal(vec)
	if err != nil {
		return "[]"
	}
	return string(bytes)
}

func parseVector(jsonStr string) []float32 {
	var vec []float32
	if err := json.Unmarshal([]byte(jsonStr), &vec); err != nil {
		return []float32{}
	}
	return vec
}
