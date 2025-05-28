package graft

import (
	"context"
	"io"
)

// Mock implementations for testing

// MockEngineV2 provides a mock implementation of EngineV2 for testing
type MockEngineV2 struct {
	// Control behavior
	ParseYAMLFunc          func(data []byte) (DocumentV2, error)
	ParseJSONFunc          func(data []byte) (DocumentV2, error)
	ParseFileFunc          func(path string) (DocumentV2, error)
	ParseReaderFunc        func(reader io.Reader) (DocumentV2, error)
	MergeFunc              func(ctx context.Context, docs ...DocumentV2) MergeBuilder
	MergeFilesFunc         func(ctx context.Context, paths ...string) MergeBuilder
	MergeReadersFunc       func(ctx context.Context, readers ...io.Reader) MergeBuilder
	EvaluateFunc           func(ctx context.Context, doc DocumentV2) (DocumentV2, error)
	ToYAMLFunc             func(doc DocumentV2) ([]byte, error)
	ToJSONFunc             func(doc DocumentV2) ([]byte, error)
	ToJSONIndentFunc       func(doc DocumentV2, indent string) ([]byte, error)
	RegisterOperatorFunc   func(name string, op Operator) error
	UnregisterOperatorFunc func(name string) error
	ListOperatorsFunc      func() []string
	WithLoggerV2Func       func(logger LoggerV2) EngineV2
	WithVaultClientV2Func  func(client VaultClientV2) EngineV2
	WithAWSConfigFunc      func(config AWSConfig) EngineV2

	// Call tracking
	ParseYAMLCalls          [][]byte
	ParseJSONCalls          [][]byte
	ParseFileCalls          []string
	ParseReaderCalls        []io.Reader
	MergeCalls              [][]DocumentV2
	MergeFilesCalls         [][]string
	MergeReadersCalls       [][]io.Reader
	EvaluateCalls           []DocumentV2
	ToYAMLCalls             []DocumentV2
	ToJSONCalls             []DocumentV2
	ToJSONIndentCalls       []struct{ Doc DocumentV2; Indent string }
	RegisterOperatorCalls   []struct{ Name string; Op Operator }
	UnregisterOperatorCalls []string
	ListOperatorsCalls      int
	WithLoggerV2Calls       []LoggerV2
	WithVaultClientV2Calls  []VaultClientV2
	WithAWSConfigCalls      []AWSConfig
}

// NewMockEngineV2 creates a new mock engine with sensible defaults
func NewMockEngineV2() *MockEngineV2 {
	return &MockEngineV2{
		// Default implementations that do nothing or return empty values
		ParseYAMLFunc:          func(data []byte) (DocumentV2, error) { return &MockDocumentV2{}, nil },
		ParseJSONFunc:          func(data []byte) (DocumentV2, error) { return &MockDocumentV2{}, nil },
		ParseFileFunc:          func(path string) (DocumentV2, error) { return &MockDocumentV2{}, nil },
		ParseReaderFunc:        func(reader io.Reader) (DocumentV2, error) { return &MockDocumentV2{}, nil },
		MergeFunc:              func(ctx context.Context, docs ...DocumentV2) MergeBuilder { return &MockMergeBuilder{} },
		MergeFilesFunc:         func(ctx context.Context, paths ...string) MergeBuilder { return &MockMergeBuilder{} },
		MergeReadersFunc:       func(ctx context.Context, readers ...io.Reader) MergeBuilder { return &MockMergeBuilder{} },
		EvaluateFunc:           func(ctx context.Context, doc DocumentV2) (DocumentV2, error) { return doc, nil },
		ToYAMLFunc:             func(doc DocumentV2) ([]byte, error) { return []byte{}, nil },
		ToJSONFunc:             func(doc DocumentV2) ([]byte, error) { return []byte{}, nil },
		ToJSONIndentFunc:       func(doc DocumentV2, indent string) ([]byte, error) { return []byte{}, nil },
		RegisterOperatorFunc:   func(name string, op Operator) error { return nil },
		UnregisterOperatorFunc: func(name string) error { return nil },
		ListOperatorsFunc:      func() []string { return []string{} },
		WithLoggerV2Func:       func(logger LoggerV2) EngineV2 { return &MockEngineV2{} },
		WithVaultClientV2Func:  func(client VaultClientV2) EngineV2 { return &MockEngineV2{} },
		WithAWSConfigFunc:      func(config AWSConfig) EngineV2 { return &MockEngineV2{} },
	}
}

// EngineV2 interface implementations
func (m *MockEngineV2) ParseYAML(data []byte) (DocumentV2, error) {
	m.ParseYAMLCalls = append(m.ParseYAMLCalls, data)
	return m.ParseYAMLFunc(data)
}

func (m *MockEngineV2) ParseJSON(data []byte) (DocumentV2, error) {
	m.ParseJSONCalls = append(m.ParseJSONCalls, data)
	return m.ParseJSONFunc(data)
}

func (m *MockEngineV2) ParseFile(path string) (DocumentV2, error) {
	m.ParseFileCalls = append(m.ParseFileCalls, path)
	return m.ParseFileFunc(path)
}

func (m *MockEngineV2) ParseReader(reader io.Reader) (DocumentV2, error) {
	m.ParseReaderCalls = append(m.ParseReaderCalls, reader)
	return m.ParseReaderFunc(reader)
}

func (m *MockEngineV2) Merge(ctx context.Context, docs ...DocumentV2) MergeBuilder {
	m.MergeCalls = append(m.MergeCalls, docs)
	return m.MergeFunc(ctx, docs...)
}

func (m *MockEngineV2) MergeFiles(ctx context.Context, paths ...string) MergeBuilder {
	m.MergeFilesCalls = append(m.MergeFilesCalls, paths)
	return m.MergeFilesFunc(ctx, paths...)
}

func (m *MockEngineV2) MergeReaders(ctx context.Context, readers ...io.Reader) MergeBuilder {
	m.MergeReadersCalls = append(m.MergeReadersCalls, readers)
	return m.MergeReadersFunc(ctx, readers...)
}

func (m *MockEngineV2) Evaluate(ctx context.Context, doc DocumentV2) (DocumentV2, error) {
	m.EvaluateCalls = append(m.EvaluateCalls, doc)
	return m.EvaluateFunc(ctx, doc)
}

func (m *MockEngineV2) ToYAML(doc DocumentV2) ([]byte, error) {
	m.ToYAMLCalls = append(m.ToYAMLCalls, doc)
	return m.ToYAMLFunc(doc)
}

func (m *MockEngineV2) ToJSON(doc DocumentV2) ([]byte, error) {
	m.ToJSONCalls = append(m.ToJSONCalls, doc)
	return m.ToJSONFunc(doc)
}

func (m *MockEngineV2) ToJSONIndent(doc DocumentV2, indent string) ([]byte, error) {
	m.ToJSONIndentCalls = append(m.ToJSONIndentCalls, struct{ Doc DocumentV2; Indent string }{doc, indent})
	return m.ToJSONIndentFunc(doc, indent)
}

func (m *MockEngineV2) RegisterOperator(name string, op Operator) error {
	m.RegisterOperatorCalls = append(m.RegisterOperatorCalls, struct{ Name string; Op Operator }{name, op})
	return m.RegisterOperatorFunc(name, op)
}

func (m *MockEngineV2) UnregisterOperator(name string) error {
	m.UnregisterOperatorCalls = append(m.UnregisterOperatorCalls, name)
	return m.UnregisterOperatorFunc(name)
}

func (m *MockEngineV2) ListOperators() []string {
	m.ListOperatorsCalls++
	return m.ListOperatorsFunc()
}

func (m *MockEngineV2) WithLoggerV2(logger LoggerV2) EngineV2 {
	m.WithLoggerV2Calls = append(m.WithLoggerV2Calls, logger)
	return m.WithLoggerV2Func(logger)
}

func (m *MockEngineV2) WithVaultClientV2(client VaultClientV2) EngineV2 {
	m.WithVaultClientV2Calls = append(m.WithVaultClientV2Calls, client)
	return m.WithVaultClientV2Func(client)
}

func (m *MockEngineV2) WithAWSConfig(config AWSConfig) EngineV2 {
	m.WithAWSConfigCalls = append(m.WithAWSConfigCalls, config)
	return m.WithAWSConfigFunc(config)
}

// MockDocumentV2 provides a mock implementation of DocumentV2 for testing
type MockDocumentV2 struct {
	// Control behavior
	GetFunc        func(path string) (interface{}, error)
	GetStringFunc  func(path string) (string, error)
	GetIntFunc     func(path string) (int, error)
	GetBoolFunc    func(path string) (bool, error)
	GetSliceFunc   func(path string) ([]interface{}, error)
	GetMapFunc     func(path string) (map[interface{}]interface{}, error)
	SetFunc        func(path string, value interface{}) error
	DeleteFunc     func(path string) error
	KeysFunc       func() []string
	ToYAMLFunc     func() ([]byte, error)
	ToJSONFunc     func() ([]byte, error)
	RawDataFunc    func() interface{}
	DeepCopyFunc   func() DocumentV2
	CloneFunc      func() DocumentV2

	// Call tracking
	GetCalls        []string
	GetStringCalls  []string
	GetIntCalls     []string
	GetBoolCalls    []string
	GetSliceCalls   []string
	GetMapCalls     []string
	SetCalls        []struct{ Path string; Value interface{} }
	DeleteCalls     []string
	KeysCalls       int
	ToYAMLCalls     int
	ToJSONCalls     int
	RawDataCalls    int
	DeepCopyCalls   int
	CloneCalls      int

	// Test data
	TestData map[string]interface{}
}

// NewMockDocumentV2 creates a new mock document with sensible defaults
func NewMockDocumentV2() *MockDocumentV2 {
	return &MockDocumentV2{
		TestData: make(map[string]interface{}),
		// Default implementations
		GetFunc:        func(path string) (interface{}, error) { return nil, nil },
		GetStringFunc:  func(path string) (string, error) { return "", nil },
		GetIntFunc:     func(path string) (int, error) { return 0, nil },
		GetBoolFunc:    func(path string) (bool, error) { return false, nil },
		GetSliceFunc:   func(path string) ([]interface{}, error) { return []interface{}{}, nil },
		GetMapFunc:     func(path string) (map[interface{}]interface{}, error) { return make(map[interface{}]interface{}), nil },
		SetFunc:        func(path string, value interface{}) error { return nil },
		DeleteFunc:     func(path string) error { return nil },
		KeysFunc:       func() []string { return []string{} },
		ToYAMLFunc:     func() ([]byte, error) { return []byte{}, nil },
		ToJSONFunc:     func() ([]byte, error) { return []byte{}, nil },
		RawDataFunc:    func() interface{} { return make(map[interface{}]interface{}) },
		DeepCopyFunc:   func() DocumentV2 { return &MockDocumentV2{} },
		CloneFunc:      func() DocumentV2 { return &MockDocumentV2{} },
	}
}

// DocumentV2 interface implementations
func (m *MockDocumentV2) Get(path string) (interface{}, error) {
	m.GetCalls = append(m.GetCalls, path)
	return m.GetFunc(path)
}

func (m *MockDocumentV2) GetString(path string) (string, error) {
	m.GetStringCalls = append(m.GetStringCalls, path)
	return m.GetStringFunc(path)
}

func (m *MockDocumentV2) GetInt(path string) (int, error) {
	m.GetIntCalls = append(m.GetIntCalls, path)
	return m.GetIntFunc(path)
}

func (m *MockDocumentV2) GetBool(path string) (bool, error) {
	m.GetBoolCalls = append(m.GetBoolCalls, path)
	return m.GetBoolFunc(path)
}

func (m *MockDocumentV2) GetSlice(path string) ([]interface{}, error) {
	m.GetSliceCalls = append(m.GetSliceCalls, path)
	return m.GetSliceFunc(path)
}

func (m *MockDocumentV2) GetMap(path string) (map[interface{}]interface{}, error) {
	m.GetMapCalls = append(m.GetMapCalls, path)
	return m.GetMapFunc(path)
}

func (m *MockDocumentV2) Set(path string, value interface{}) error {
	m.SetCalls = append(m.SetCalls, struct{ Path string; Value interface{} }{path, value})
	return m.SetFunc(path, value)
}

func (m *MockDocumentV2) Delete(path string) error {
	m.DeleteCalls = append(m.DeleteCalls, path)
	return m.DeleteFunc(path)
}

func (m *MockDocumentV2) Keys() []string {
	m.KeysCalls++
	return m.KeysFunc()
}

func (m *MockDocumentV2) ToYAML() ([]byte, error) {
	m.ToYAMLCalls++
	return m.ToYAMLFunc()
}

func (m *MockDocumentV2) ToJSON() ([]byte, error) {
	m.ToJSONCalls++
	return m.ToJSONFunc()
}

func (m *MockDocumentV2) RawData() interface{} {
	m.RawDataCalls++
	return m.RawDataFunc()
}

func (m *MockDocumentV2) DeepCopy() DocumentV2 {
	m.DeepCopyCalls++
	return m.DeepCopyFunc()
}

func (m *MockDocumentV2) Clone() DocumentV2 {
	m.CloneCalls++
	return m.CloneFunc()
}

// MockMergeBuilder provides a mock implementation of MergeBuilder for testing
type MockMergeBuilder struct {
	// Control behavior
	WithPruneFunc        func(keys ...string) MergeBuilder
	WithCherryPickFunc   func(keys ...string) MergeBuilder
	SkipEvaluationFunc   func() MergeBuilder
	EnableGoPatchFunc    func() MergeBuilder
	FallbackAppendFunc   func() MergeBuilder
	ExecuteFunc          func() (DocumentV2, error)

	// Call tracking
	WithPruneCalls        [][]string
	WithCherryPickCalls   [][]string
	SkipEvaluationCalls   int
	EnableGoPatchCalls    int
	FallbackAppendCalls   int
	ExecuteCalls          int
}

// NewMockMergeBuilder creates a new mock merge builder with sensible defaults
func NewMockMergeBuilder() *MockMergeBuilder {
	mock := &MockMergeBuilder{
		ExecuteFunc: func() (DocumentV2, error) { return &MockDocumentV2{}, nil },
	}
	
	// Self-returning methods
	mock.WithPruneFunc = func(keys ...string) MergeBuilder { return mock }
	mock.WithCherryPickFunc = func(keys ...string) MergeBuilder { return mock }
	mock.SkipEvaluationFunc = func() MergeBuilder { return mock }
	mock.EnableGoPatchFunc = func() MergeBuilder { return mock }
	mock.FallbackAppendFunc = func() MergeBuilder { return mock }
	
	return mock
}

// MergeBuilder interface implementations
func (m *MockMergeBuilder) WithPrune(keys ...string) MergeBuilder {
	m.WithPruneCalls = append(m.WithPruneCalls, keys)
	return m.WithPruneFunc(keys...)
}

func (m *MockMergeBuilder) WithCherryPick(keys ...string) MergeBuilder {
	m.WithCherryPickCalls = append(m.WithCherryPickCalls, keys)
	return m.WithCherryPickFunc(keys...)
}

func (m *MockMergeBuilder) SkipEvaluation() MergeBuilder {
	m.SkipEvaluationCalls++
	return m.SkipEvaluationFunc()
}

func (m *MockMergeBuilder) EnableGoPatch() MergeBuilder {
	m.EnableGoPatchCalls++
	return m.EnableGoPatchFunc()
}

func (m *MockMergeBuilder) FallbackAppend() MergeBuilder {
	m.FallbackAppendCalls++
	return m.FallbackAppendFunc()
}

func (m *MockMergeBuilder) Execute() (DocumentV2, error) {
	m.ExecuteCalls++
	return m.ExecuteFunc()
}