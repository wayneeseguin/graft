package graft

import (
	"fmt"
	
	"github.com/cppforlife/go-patch/patch"
)

// goPatchDocument is a special document type that holds go-patch operations
type goPatchDocument struct {
	ops patch.Ops
}

// Implement Document interface
func (g *goPatchDocument) Get(path string) (interface{}, error) {
	return nil, fmt.Errorf("go-patch documents do not support Get operations")
}

func (g *goPatchDocument) GetString(path string) (string, error) {
	return "", fmt.Errorf("go-patch documents do not support GetString operations")
}

func (g *goPatchDocument) GetInt(path string) (int, error) {
	return 0, fmt.Errorf("go-patch documents do not support GetInt operations")
}

func (g *goPatchDocument) GetBool(path string) (bool, error) {
	return false, fmt.Errorf("go-patch documents do not support GetBool operations")
}

func (g *goPatchDocument) GetSlice(path string) ([]interface{}, error) {
	return nil, fmt.Errorf("go-patch documents do not support GetSlice operations")
}

func (g *goPatchDocument) GetMap(path string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("go-patch documents do not support GetMap operations")
}

func (g *goPatchDocument) Set(path string, value interface{}) error {
	return fmt.Errorf("go-patch documents do not support Set operations")
}

func (g *goPatchDocument) Delete(path string) error {
	return fmt.Errorf("go-patch documents do not support Delete operations")
}

func (g *goPatchDocument) Keys() []string {
	return []string{}
}

func (g *goPatchDocument) ToYAML() ([]byte, error) {
	return nil, fmt.Errorf("go-patch documents do not support ToYAML operations")
}

func (g *goPatchDocument) ToJSON() ([]byte, error) {
	return nil, fmt.Errorf("go-patch documents do not support ToJSON operations")
}

func (g *goPatchDocument) RawData() interface{} {
	// Return the ops as the raw data
	return g.ops
}

func (g *goPatchDocument) Prune(key string) Document {
	return g // No-op for go-patch documents
}

func (g *goPatchDocument) Clone() Document {
	return &goPatchDocument{ops: g.ops}
}

func (g *goPatchDocument) CherryPick(keys ...string) Document {
	return g // No-op for go-patch documents
}

func (g *goPatchDocument) GetData() interface{} {
	return g.ops
}

func (g *goPatchDocument) GetInt64(path string) (int64, error) {
	return 0, fmt.Errorf("go-patch documents do not support GetInt64 operations")
}

func (g *goPatchDocument) GetFloat64(path string) (float64, error) {
	return 0, fmt.Errorf("go-patch documents do not support GetFloat64 operations")
}

func (g *goPatchDocument) GetStringSlice(path string) ([]string, error) {
	return nil, fmt.Errorf("go-patch documents do not support GetStringSlice operations")
}

func (g *goPatchDocument) GetMapStringString(path string) (map[string]string, error) {
	return nil, fmt.Errorf("go-patch documents do not support GetMapStringString operations")
}

// IsGoPatchDocument checks if a document is a go-patch document
func IsGoPatchDocument(doc Document) bool {
	_, ok := doc.(*goPatchDocument)
	return ok
}

// GetGoPatchOps extracts go-patch operations from a document
func GetGoPatchOps(doc Document) (patch.Ops, bool) {
	if gpDoc, ok := doc.(*goPatchDocument); ok {
		return gpDoc.ops, true
	}
	return nil, false
}

// NewGoPatchDocument creates a new go-patch document
func NewGoPatchDocument(ops patch.Ops) Document {
	return &goPatchDocument{ops: ops}
}