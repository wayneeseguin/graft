package graft

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

// TestHelper provides utilities for testing graft operations
type TestHelper struct {
	engine EngineV2
	t      *testing.T
}

// NewTestHelper creates a new test helper with default engine configuration
func NewTestHelper(t *testing.T) *TestHelper {
	engine, err := NewEngineV2()
	if err != nil {
		t.Fatalf("Failed to create test engine: %v", err)
	}
	return &TestHelper{
		engine: engine,
		t:      t,
	}
}

// NewTestHelperWithOptions creates a new test helper with custom engine options
func NewTestHelperWithOptions(t *testing.T, opts ...EngineOption) *TestHelper {
	engine, err := NewEngineV2(opts...)
	if err != nil {
		t.Fatalf("Failed to create test engine with options: %v", err)
	}
	return &TestHelper{
		engine: engine,
		t:      t,
	}
}

// ParseYAMLString parses YAML from string and returns a document
func (h *TestHelper) ParseYAMLString(yamlStr string) DocumentV2 {
	doc, err := h.engine.ParseYAML([]byte(yamlStr))
	if err != nil {
		h.t.Fatalf("Failed to parse YAML: %v\nYAML:\n%s", err, yamlStr)
	}
	return doc
}

// ParseJSONString parses JSON from string and returns a document
func (h *TestHelper) ParseJSONString(jsonStr string) DocumentV2 {
	doc, err := h.engine.ParseJSON([]byte(jsonStr))
	if err != nil {
		h.t.Fatalf("Failed to parse JSON: %v\nJSON:\n%s", err, jsonStr)
	}
	return doc
}

// MustMerge merges documents and fails the test if there's an error
func (h *TestHelper) MustMerge(docs ...DocumentV2) DocumentV2 {
	ctx := context.Background()
	result, err := h.engine.Merge(ctx, docs...).Execute()
	if err != nil {
		h.t.Fatalf("Failed to merge documents: %v", err)
	}
	return result
}

// MustMergeWithPrune merges documents with pruning and fails the test if there's an error
func (h *TestHelper) MustMergeWithPrune(prune []string, docs ...DocumentV2) DocumentV2 {
	ctx := context.Background()
	builder := h.engine.Merge(ctx, docs...)
	for _, key := range prune {
		builder = builder.WithPrune(key)
	}
	result, err := builder.Execute()
	if err != nil {
		h.t.Fatalf("Failed to merge documents with prune: %v", err)
	}
	return result
}

// MustEvaluate evaluates a document and fails the test if there's an error
func (h *TestHelper) MustEvaluate(doc DocumentV2) DocumentV2 {
	ctx := context.Background()
	result, err := h.engine.Evaluate(ctx, doc)
	if err != nil {
		h.t.Fatalf("Failed to evaluate document: %v", err)
	}
	return result
}

// MustMergeAndEvaluate merges and evaluates documents in one step
func (h *TestHelper) MustMergeAndEvaluate(docs ...DocumentV2) DocumentV2 {
	merged := h.MustMerge(docs...)
	return h.MustEvaluate(merged)
}

// AssertPath asserts that a path exists and has the expected value
func (h *TestHelper) AssertPath(doc DocumentV2, path string, expected interface{}) {
	actual, err := doc.Get(path)
	if err != nil {
		h.t.Fatalf("Path '%s' not found: %v", path, err)
	}
	if !reflect.DeepEqual(actual, expected) {
		h.t.Fatalf("Path '%s': expected %v (%T), got %v (%T)", path, expected, expected, actual, actual)
	}
}

// AssertPathString asserts that a path exists and has the expected string value
func (h *TestHelper) AssertPathString(doc DocumentV2, path string, expected string) {
	actual, err := doc.GetString(path)
	if err != nil {
		h.t.Fatalf("Path '%s' string value error: %v", path, err)
	}
	if actual != expected {
		h.t.Fatalf("Path '%s': expected %q, got %q", path, expected, actual)
	}
}

// AssertPathInt asserts that a path exists and has the expected int value
func (h *TestHelper) AssertPathInt(doc DocumentV2, path string, expected int) {
	actual, err := doc.GetInt(path)
	if err != nil {
		h.t.Fatalf("Path '%s' int value error: %v", path, err)
	}
	if actual != expected {
		h.t.Fatalf("Path '%s': expected %d, got %d", path, expected, actual)
	}
}

// AssertPathBool asserts that a path exists and has the expected bool value
func (h *TestHelper) AssertPathBool(doc DocumentV2, path string, expected bool) {
	actual, err := doc.GetBool(path)
	if err != nil {
		h.t.Fatalf("Path '%s' bool value error: %v", path, err)
	}
	if actual != expected {
		h.t.Fatalf("Path '%s': expected %t, got %t", path, expected, actual)
	}
}

// AssertPathNotExists asserts that a path does not exist
func (h *TestHelper) AssertPathNotExists(doc DocumentV2, path string) {
	_, err := doc.Get(path)
	if err == nil {
		h.t.Fatalf("Path '%s' should not exist", path)
	}
}

// AssertError asserts that an error occurred and optionally checks the error type
func (h *TestHelper) AssertError(err error, expectedType ...ErrorType) {
	if err == nil {
		h.t.Fatal("Expected an error but got nil")
	}

	if len(expectedType) > 0 {
		graftErr, ok := err.(*GraftError)
		if !ok {
			h.t.Fatalf("Expected GraftError but got %T: %v", err, err)
		}
		if graftErr.Type != expectedType[0] {
			h.t.Fatalf("Expected error type %v but got %v: %v", expectedType[0], graftErr.Type, err)
		}
	}
}

// AssertNoError asserts that no error occurred
func (h *TestHelper) AssertNoError(err error) {
	if err != nil {
		h.t.Fatalf("Unexpected error: %v", err)
	}
}

// MockOperator creates a simple mock operator for testing
type MockOperator struct {
	Name        string
	ReturnValue interface{}
	ReturnError error
	CallCount   int
	LastArgs    []interface{}
}

// Type returns the operator name
func (m *MockOperator) Type() string {
	return m.Name
}

// Run executes the mock operator (placeholder implementation)
func (m *MockOperator) Run(ev *Evaluator, args []interface{}) (interface{}, error) {
	m.CallCount++
	m.LastArgs = args

	if m.ReturnError != nil {
		return nil, m.ReturnError
	}

	return m.ReturnValue, nil
}

// TestWithMockOperator provides a way to test with custom mock operators
func (h *TestHelper) TestWithMockOperator(name string, mock *MockOperator, testFunc func()) {
	// Note: This would require engine to support operator registration
	// For now, this is a placeholder for future implementation
	h.t.Log("Mock operator testing not yet implemented")
	testFunc()
}

// CompareDocuments compares two documents and returns detailed differences
func (h *TestHelper) CompareDocuments(doc1, doc2 DocumentV2) []string {
	var differences []string

	yaml1, err1 := doc1.ToYAML()
	yaml2, err2 := doc2.ToYAML()

	if err1 != nil || err2 != nil {
		differences = append(differences, fmt.Sprintf("YAML conversion error: doc1=%v, doc2=%v", err1, err2))
		return differences
	}

	if string(yaml1) != string(yaml2) {
		differences = append(differences, "Documents differ")
		differences = append(differences, fmt.Sprintf("Doc1:\n%s", string(yaml1)))
		differences = append(differences, fmt.Sprintf("Doc2:\n%s", string(yaml2)))
	}

	return differences
}

// AssertDocumentsEqual asserts that two documents are equal
func (h *TestHelper) AssertDocumentsEqual(doc1, doc2 DocumentV2) {
	differences := h.CompareDocuments(doc1, doc2)
	if len(differences) > 0 {
		h.t.Fatalf("Documents are not equal:\n%s", fmt.Sprintf("%v", differences))
	}
}

// CreateTestDocument creates a document from a map for testing
func (h *TestHelper) CreateTestDocument(data map[string]interface{}) DocumentV2 {
	// Convert string keys to interface{} keys to match graft's internal format
	converted := make(map[interface{}]interface{})
	h.convertMapKeys(data, converted)
	return &document{data: converted}
}

func (h *TestHelper) convertMapKeys(src map[string]interface{}, dst map[interface{}]interface{}) {
	for k, v := range src {
		switch val := v.(type) {
		case map[string]interface{}:
			nested := make(map[interface{}]interface{})
			h.convertMapKeys(val, nested)
			dst[k] = nested
		case []interface{}:
			newSlice := make([]interface{}, len(val))
			for i, item := range val {
				if itemMap, ok := item.(map[string]interface{}); ok {
					nested := make(map[interface{}]interface{})
					h.convertMapKeys(itemMap, nested)
					newSlice[i] = nested
				} else {
					newSlice[i] = item
				}
			}
			dst[k] = newSlice
		default:
			dst[k] = v
		}
	}
}