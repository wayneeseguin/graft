package graft

import (
	"context"
	"io"
	
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	vaultkv "github.com/cloudfoundry-community/vaultkv"
)

// Mock implementations for testing

// MockEngine provides a mock implementation of Engine for testing
type MockEngine struct {
	// Control behavior
	ParseYAMLFunc          func(data []byte) (Document, error)
	ParseJSONFunc          func(data []byte) (Document, error)
	ParseFileFunc          func(path string) (Document, error)
	ParseReaderFunc        func(reader io.Reader) (Document, error)
	MergeFunc              func(ctx context.Context, docs ...Document) MergeBuilder
	MergeFilesFunc         func(ctx context.Context, paths ...string) MergeBuilder
	MergeReadersFunc       func(ctx context.Context, readers ...io.Reader) MergeBuilder
	EvaluateFunc           func(ctx context.Context, doc Document) (Document, error)
	ToYAMLFunc             func(doc Document) ([]byte, error)
	ToJSONFunc             func(doc Document) ([]byte, error)
	ToJSONIndentFunc       func(doc Document, indent string) ([]byte, error)
	RegisterOperatorFunc   func(name string, op Operator) error
	UnregisterOperatorFunc func(name string) error
	ListOperatorsFunc      func() []string
	GetOperatorFunc        func(name string) (Operator, bool)
	WithLoggerFunc         func(logger Logger) Engine
	WithVaultClientFunc    func(client VaultClient) Engine
	WithAWSConfigFunc      func(config AWSConfig) Engine
	GetOperatorStateFunc   func() OperatorState

	// Call tracking
	ParseYAMLCalls          [][]byte
	ParseJSONCalls          [][]byte
	ParseFileCalls          []string
	ParseReaderCalls        []io.Reader
	MergeCalls              [][]Document
	MergeFilesCalls         [][]string
	MergeReadersCalls       [][]io.Reader
	EvaluateCalls           []Document
	ToYAMLCalls             []Document
	ToJSONCalls             []Document
	ToJSONIndentCalls       []struct{ Doc Document; Indent string }
	RegisterOperatorCalls   []struct{ Name string; Op Operator }
	UnregisterOperatorCalls []string
	ListOperatorsCalls      int
	GetOperatorCalls        []string
	WithLoggerCalls         []Logger
	WithVaultClientCalls    []VaultClient
	WithAWSConfigCalls      []AWSConfig
	GetOperatorStateCalls   int
}

// NewMockEngine creates a new mock engine with sensible defaults
func NewMockEngine() *MockEngine {
	return &MockEngine{
		// Default implementations that do nothing or return empty values
		ParseYAMLFunc:          func(data []byte) (Document, error) { return &MockDocument{}, nil },
		ParseJSONFunc:          func(data []byte) (Document, error) { return &MockDocument{}, nil },
		ParseFileFunc:          func(path string) (Document, error) { return &MockDocument{}, nil },
		ParseReaderFunc:        func(reader io.Reader) (Document, error) { return &MockDocument{}, nil },
		MergeFunc:              func(ctx context.Context, docs ...Document) MergeBuilder { return &MockMergeBuilder{} },
		MergeFilesFunc:         func(ctx context.Context, paths ...string) MergeBuilder { return &MockMergeBuilder{} },
		MergeReadersFunc:       func(ctx context.Context, readers ...io.Reader) MergeBuilder { return &MockMergeBuilder{} },
		EvaluateFunc:           func(ctx context.Context, doc Document) (Document, error) { return doc, nil },
		ToYAMLFunc:             func(doc Document) ([]byte, error) { return []byte{}, nil },
		ToJSONFunc:             func(doc Document) ([]byte, error) { return []byte{}, nil },
		ToJSONIndentFunc:       func(doc Document, indent string) ([]byte, error) { return []byte{}, nil },
		RegisterOperatorFunc:   func(name string, op Operator) error { return nil },
		UnregisterOperatorFunc: func(name string) error { return nil },
		ListOperatorsFunc:      func() []string { return []string{} },
		WithLoggerFunc:       func(logger Logger) Engine { return &MockEngine{} },
		WithVaultClientFunc:  func(client VaultClient) Engine { return &MockEngine{} },
		WithAWSConfigFunc:      func(config AWSConfig) Engine { return &MockEngine{} },
	}
}

// Engine interface implementations
func (m *MockEngine) ParseYAML(data []byte) (Document, error) {
	m.ParseYAMLCalls = append(m.ParseYAMLCalls, data)
	return m.ParseYAMLFunc(data)
}

func (m *MockEngine) ParseJSON(data []byte) (Document, error) {
	m.ParseJSONCalls = append(m.ParseJSONCalls, data)
	return m.ParseJSONFunc(data)
}

func (m *MockEngine) ParseFile(path string) (Document, error) {
	m.ParseFileCalls = append(m.ParseFileCalls, path)
	return m.ParseFileFunc(path)
}

func (m *MockEngine) ParseReader(reader io.Reader) (Document, error) {
	m.ParseReaderCalls = append(m.ParseReaderCalls, reader)
	return m.ParseReaderFunc(reader)
}

func (m *MockEngine) Merge(ctx context.Context, docs ...Document) MergeBuilder {
	m.MergeCalls = append(m.MergeCalls, docs)
	return m.MergeFunc(ctx, docs...)
}

func (m *MockEngine) MergeFiles(ctx context.Context, paths ...string) MergeBuilder {
	m.MergeFilesCalls = append(m.MergeFilesCalls, paths)
	return m.MergeFilesFunc(ctx, paths...)
}

func (m *MockEngine) MergeReaders(ctx context.Context, readers ...io.Reader) MergeBuilder {
	m.MergeReadersCalls = append(m.MergeReadersCalls, readers)
	return m.MergeReadersFunc(ctx, readers...)
}

func (m *MockEngine) Evaluate(ctx context.Context, doc Document) (Document, error) {
	m.EvaluateCalls = append(m.EvaluateCalls, doc)
	return m.EvaluateFunc(ctx, doc)
}

func (m *MockEngine) ToYAML(doc Document) ([]byte, error) {
	m.ToYAMLCalls = append(m.ToYAMLCalls, doc)
	return m.ToYAMLFunc(doc)
}

func (m *MockEngine) ToJSON(doc Document) ([]byte, error) {
	m.ToJSONCalls = append(m.ToJSONCalls, doc)
	return m.ToJSONFunc(doc)
}

func (m *MockEngine) ToJSONIndent(doc Document, indent string) ([]byte, error) {
	m.ToJSONIndentCalls = append(m.ToJSONIndentCalls, struct{ Doc Document; Indent string }{doc, indent})
	return m.ToJSONIndentFunc(doc, indent)
}

func (m *MockEngine) RegisterOperator(name string, op Operator) error {
	m.RegisterOperatorCalls = append(m.RegisterOperatorCalls, struct{ Name string; Op Operator }{name, op})
	return m.RegisterOperatorFunc(name, op)
}

func (m *MockEngine) UnregisterOperator(name string) error {
	m.UnregisterOperatorCalls = append(m.UnregisterOperatorCalls, name)
	return m.UnregisterOperatorFunc(name)
}

func (m *MockEngine) ListOperators() []string {
	m.ListOperatorsCalls++
	return m.ListOperatorsFunc()
}

func (m *MockEngine) GetOperator(name string) (Operator, bool) {
	m.GetOperatorCalls = append(m.GetOperatorCalls, name)
	return m.GetOperatorFunc(name)
}

func (m *MockEngine) WithLogger(logger Logger) Engine {
	m.WithLoggerCalls = append(m.WithLoggerCalls, logger)
	return m.WithLoggerFunc(logger)
}

func (m *MockEngine) WithVaultClient(client VaultClient) Engine {
	m.WithVaultClientCalls = append(m.WithVaultClientCalls, client)
	return m.WithVaultClientFunc(client)
}

func (m *MockEngine) WithAWSConfig(config AWSConfig) Engine {
	m.WithAWSConfigCalls = append(m.WithAWSConfigCalls, config)
	return m.WithAWSConfigFunc(config)
}

func (m *MockEngine) GetOperatorState() OperatorState {
	m.GetOperatorStateCalls++
	return m.GetOperatorStateFunc()
}

// MockDocument provides a mock implementation of Document for testing
type MockDocument struct {
	// Control behavior
	GetFunc              func(path string) (interface{}, error)
	GetStringFunc        func(path string) (string, error)
	GetIntFunc           func(path string) (int, error)
	GetInt64Func         func(path string) (int64, error)
	GetFloat64Func       func(path string) (float64, error)
	GetBoolFunc          func(path string) (bool, error)
	GetMapFunc           func(path string) (map[string]interface{}, error)
	GetSliceFunc         func(path string) ([]interface{}, error)
	GetStringSliceFunc   func(path string) ([]string, error)
	GetMapStringStringFunc func(path string) (map[string]string, error)
	SetFunc              func(path string, value interface{}) error
	DeleteFunc           func(path string) error
	KeysFunc             func() []string
	ToYAMLFunc           func() ([]byte, error)
	ToJSONFunc           func() ([]byte, error)
	RawDataFunc          func() interface{}
	DeepCopyFunc         func() Document
	CloneFunc            func() Document
	PruneFunc            func(key string) Document
	CherryPickFunc       func(keys ...string) Document
	GetDataFunc          func() interface{}

	// Call tracking
	GetCalls               []string
	GetStringCalls         []string
	GetIntCalls            []string
	GetInt64Calls          []string
	GetFloat64Calls        []string
	GetBoolCalls           []string
	GetMapCalls            []string
	GetSliceCalls          []string
	GetStringSliceCalls    []string
	GetMapStringStringCalls []string
	SetCalls               []struct{ Path string; Value interface{} }
	DeleteCalls            []string
	KeysCalls              int
	ToYAMLCalls            int
	ToJSONCalls            int
	RawDataCalls           int
	DeepCopyCalls          int
	CloneCalls             int
	PruneCalls             []string
	CherryPickCalls        [][]string
	GetDataCalls           int

	// Test data
	TestData map[string]interface{}
}

// NewMockDocument creates a new mock document with sensible defaults
func NewMockDocument() *MockDocument {
	m := &MockDocument{
		TestData: make(map[string]interface{}),
		// Default implementations
		GetFunc:              func(path string) (interface{}, error) { return nil, nil },
		GetStringFunc:        func(path string) (string, error) { return "", nil },
		GetIntFunc:           func(path string) (int, error) { return 0, nil },
		GetInt64Func:         func(path string) (int64, error) { return 0, nil },
		GetFloat64Func:       func(path string) (float64, error) { return 0, nil },
		GetBoolFunc:          func(path string) (bool, error) { return false, nil },
		GetMapFunc:           func(path string) (map[string]interface{}, error) { return make(map[string]interface{}), nil },
		GetSliceFunc:         func(path string) ([]interface{}, error) { return []interface{}{}, nil },
		GetStringSliceFunc:   func(path string) ([]string, error) { return []string{}, nil },
		GetMapStringStringFunc: func(path string) (map[string]string, error) { return make(map[string]string), nil },
		SetFunc:              func(path string, value interface{}) error { return nil },
		DeleteFunc:           func(path string) error { return nil },
		KeysFunc:             func() []string { return []string{} },
		ToYAMLFunc:           func() ([]byte, error) { return []byte{}, nil },
		ToJSONFunc:           func() ([]byte, error) { return []byte{}, nil },
		RawDataFunc:          func() interface{} { return make(map[interface{}]interface{}) },
		GetDataFunc:          func() interface{} { return make(map[interface{}]interface{}) },
	}
	// Self-referential functions need to be set after creation
	m.DeepCopyFunc = func() Document { return NewMockDocument() }
	m.CloneFunc = func() Document { return NewMockDocument() }
	m.PruneFunc = func(key string) Document { return NewMockDocument() }
	m.CherryPickFunc = func(keys ...string) Document { return NewMockDocument() }
	return m
}

// Document interface implementations
func (m *MockDocument) Get(path string) (interface{}, error) {
	m.GetCalls = append(m.GetCalls, path)
	return m.GetFunc(path)
}

func (m *MockDocument) GetString(path string) (string, error) {
	m.GetStringCalls = append(m.GetStringCalls, path)
	return m.GetStringFunc(path)
}

func (m *MockDocument) GetInt(path string) (int, error) {
	m.GetIntCalls = append(m.GetIntCalls, path)
	return m.GetIntFunc(path)
}

func (m *MockDocument) GetBool(path string) (bool, error) {
	m.GetBoolCalls = append(m.GetBoolCalls, path)
	return m.GetBoolFunc(path)
}

func (m *MockDocument) GetInt64(path string) (int64, error) {
	m.GetInt64Calls = append(m.GetInt64Calls, path)
	return m.GetInt64Func(path)
}

func (m *MockDocument) GetFloat64(path string) (float64, error) {
	m.GetFloat64Calls = append(m.GetFloat64Calls, path)
	return m.GetFloat64Func(path)
}

func (m *MockDocument) GetSlice(path string) ([]interface{}, error) {
	m.GetSliceCalls = append(m.GetSliceCalls, path)
	return m.GetSliceFunc(path)
}

func (m *MockDocument) GetMap(path string) (map[string]interface{}, error) {
	m.GetMapCalls = append(m.GetMapCalls, path)
	return m.GetMapFunc(path)
}

func (m *MockDocument) GetStringSlice(path string) ([]string, error) {
	m.GetStringSliceCalls = append(m.GetStringSliceCalls, path)
	return m.GetStringSliceFunc(path)
}

func (m *MockDocument) GetMapStringString(path string) (map[string]string, error) {
	m.GetMapStringStringCalls = append(m.GetMapStringStringCalls, path)
	return m.GetMapStringStringFunc(path)
}

func (m *MockDocument) Set(path string, value interface{}) error {
	m.SetCalls = append(m.SetCalls, struct{ Path string; Value interface{} }{path, value})
	return m.SetFunc(path, value)
}

func (m *MockDocument) Delete(path string) error {
	m.DeleteCalls = append(m.DeleteCalls, path)
	return m.DeleteFunc(path)
}

func (m *MockDocument) Keys() []string {
	m.KeysCalls++
	return m.KeysFunc()
}

func (m *MockDocument) ToYAML() ([]byte, error) {
	m.ToYAMLCalls++
	return m.ToYAMLFunc()
}

func (m *MockDocument) ToJSON() ([]byte, error) {
	m.ToJSONCalls++
	return m.ToJSONFunc()
}

func (m *MockDocument) RawData() interface{} {
	m.RawDataCalls++
	return m.RawDataFunc()
}

func (m *MockDocument) DeepCopy() Document {
	m.DeepCopyCalls++
	return m.DeepCopyFunc()
}

func (m *MockDocument) Clone() Document {
	m.CloneCalls++
	return m.CloneFunc()
}

func (m *MockDocument) Prune(key string) Document {
	m.PruneCalls = append(m.PruneCalls, key)
	return m.PruneFunc(key)
}

func (m *MockDocument) CherryPick(keys ...string) Document {
	m.CherryPickCalls = append(m.CherryPickCalls, keys)
	return m.CherryPickFunc(keys...)
}

func (m *MockDocument) GetData() interface{} {
	m.GetDataCalls++
	return m.GetDataFunc()
}

// MockMergeBuilder provides a mock implementation of MergeBuilder for testing
type MockMergeBuilder struct {
	// Control behavior
	WithPruneFunc              func(keys ...string) MergeBuilder
	WithCherryPickFunc         func(keys ...string) MergeBuilder
	WithArrayMergeStrategyFunc func(strategy ArrayMergeStrategy) MergeBuilder
	SkipEvaluationFunc         func() MergeBuilder
	EnableGoPatchFunc          func() MergeBuilder
	FallbackAppendFunc         func() MergeBuilder
	ExecuteFunc                func() (Document, error)

	// Call tracking
	WithPruneCalls              [][]string
	WithArrayMergeStrategyCalls []ArrayMergeStrategy
	WithCherryPickCalls   [][]string
	SkipEvaluationCalls   int
	EnableGoPatchCalls    int
	FallbackAppendCalls   int
	ExecuteCalls          int
}

// NewMockMergeBuilder creates a new mock merge builder with sensible defaults
func NewMockMergeBuilder() *MockMergeBuilder {
	mock := &MockMergeBuilder{
		ExecuteFunc: func() (Document, error) { return &MockDocument{}, nil },
	}
	
	// Self-returning methods
	mock.WithPruneFunc = func(keys ...string) MergeBuilder { return mock }
	mock.WithCherryPickFunc = func(keys ...string) MergeBuilder { return mock }
	mock.WithArrayMergeStrategyFunc = func(strategy ArrayMergeStrategy) MergeBuilder { return mock }
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

func (m *MockMergeBuilder) WithArrayMergeStrategy(strategy ArrayMergeStrategy) MergeBuilder {
	m.WithArrayMergeStrategyCalls = append(m.WithArrayMergeStrategyCalls, strategy)
	return m.WithArrayMergeStrategyFunc(strategy)
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

func (m *MockMergeBuilder) Execute() (Document, error) {
	m.ExecuteCalls++
	return m.ExecuteFunc()
}

// MockOperatorState provides a mock implementation of OperatorState for testing
type MockOperatorState struct {
	// Vault operations
	GetVaultClientFunc    func() *vaultkv.KV
	GetVaultCacheFunc     func() map[string]map[string]interface{}
	SetVaultCacheFunc     func(path string, data map[string]interface{})
	AddVaultRefFunc       func(path string, keys []string)
	IsVaultSkippedFunc    func() bool
	
	// AWS operations
	GetAWSSessionFunc         func() *session.Session
	GetSecretsManagerClientFunc func() secretsmanageriface.SecretsManagerAPI
	GetParameterStoreClientFunc func() ssmiface.SSMAPI
	GetAWSSecretsCacheFunc    func() map[string]string
	SetAWSSecretCacheFunc     func(key, value string)
	GetAWSParamsCacheFunc     func() map[string]string
	SetAWSParamCacheFunc      func(key, value string)
	IsAWSSkippedFunc          func() bool
	
	// Static IPs
	GetUsedIPsFunc func() map[string]string
	SetUsedIPFunc  func(key, ip string)
	
	// Prune operations
	AddKeyToPruneFunc   func(key string)
	GetKeysToPruneFunc  func() []string
	
	// Sort operations
	AddPathToSortFunc   func(path, order string)
	GetPathsToSortFunc  func() map[string]string
}

func (m *MockOperatorState) GetVaultClient() *vaultkv.KV {
	return m.GetVaultClientFunc()
}

func (m *MockOperatorState) GetVaultCache() map[string]map[string]interface{} {
	return m.GetVaultCacheFunc()
}

func (m *MockOperatorState) SetVaultCache(path string, data map[string]interface{}) {
	m.SetVaultCacheFunc(path, data)
}

func (m *MockOperatorState) AddVaultRef(path string, keys []string) {
	m.AddVaultRefFunc(path, keys)
}

func (m *MockOperatorState) IsVaultSkipped() bool {
	return m.IsVaultSkippedFunc()
}

func (m *MockOperatorState) GetAWSSession() *session.Session {
	return m.GetAWSSessionFunc()
}

func (m *MockOperatorState) GetSecretsManagerClient() secretsmanageriface.SecretsManagerAPI {
	return m.GetSecretsManagerClientFunc()
}

func (m *MockOperatorState) GetParameterStoreClient() ssmiface.SSMAPI {
	return m.GetParameterStoreClientFunc()
}

func (m *MockOperatorState) GetAWSSecretsCache() map[string]string {
	return m.GetAWSSecretsCacheFunc()
}

func (m *MockOperatorState) SetAWSSecretCache(key, value string) {
	m.SetAWSSecretCacheFunc(key, value)
}

func (m *MockOperatorState) GetAWSParamsCache() map[string]string {
	return m.GetAWSParamsCacheFunc()
}

func (m *MockOperatorState) SetAWSParamCache(key, value string) {
	m.SetAWSParamCacheFunc(key, value)
}

func (m *MockOperatorState) IsAWSSkipped() bool {
	return m.IsAWSSkippedFunc()
}

func (m *MockOperatorState) GetUsedIPs() map[string]string {
	return m.GetUsedIPsFunc()
}

func (m *MockOperatorState) SetUsedIP(key, ip string) {
	m.SetUsedIPFunc(key, ip)
}

func (m *MockOperatorState) AddKeyToPrune(key string) {
	m.AddKeyToPruneFunc(key)
}

func (m *MockOperatorState) GetKeysToPrune() []string {
	return m.GetKeysToPruneFunc()
}

func (m *MockOperatorState) AddPathToSort(path, order string) {
	m.AddPathToSortFunc(path, order)
}

func (m *MockOperatorState) GetPathsToSort() map[string]string {
	return m.GetPathsToSortFunc()
}