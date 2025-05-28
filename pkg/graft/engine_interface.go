package graft

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	vaultkv "github.com/cloudfoundry-community/vaultkv"
)

// EngineContext provides access to engine state for operators
// This internal interface allows operators to access engine state without circular dependencies
type EngineContext interface {
	// Operator registry
	GetOperator(name string) (Operator, bool)
	
	// Vault operations
	GetVaultClient() *vaultkv.KV
	GetVaultCache() map[string]map[string]interface{}
	SetVaultCache(path string, data map[string]interface{})
	AddVaultRef(path string, keys []string)
	IsVaultSkipped() bool
	
	// AWS operations
	GetAWSSession() *session.Session
	GetSecretsManagerClient() secretsmanageriface.SecretsManagerAPI
	GetParameterStoreClient() ssmiface.SSMAPI
	GetAWSSecretsCache() map[string]string
	SetAWSSecretCache(key, value string)
	GetAWSParamsCache() map[string]string
	SetAWSParamCache(key, value string)
	IsAWSSkipped() bool
	
	// Static IPs
	GetUsedIPs() map[string]string
	SetUsedIP(key, ip string)
	
	// Prune operations
	AddKeyToPrune(key string)
	GetKeysToPrune() []string
	
	// Sort operations
	AddPathToSort(path, order string)
	GetPathsToSort() map[string]string
}

// GetEngine returns the engine context from an evaluator
func GetEngine(ev *Evaluator) EngineContext {
	if ev.engine != nil {
		if eng, ok := ev.engine.(EngineContext); ok {
			return eng
		}
	}
	// Return a default engine context that uses global state (for backward compatibility)
	return &defaultEngineContext{}
}

// defaultEngineContext provides backward compatibility with global state
type defaultEngineContext struct{}

func (d *defaultEngineContext) GetOperator(name string) (Operator, bool) {
	op, exists := OpRegistry[name]
	return op, exists
}

func (d *defaultEngineContext) GetVaultClient() *vaultkv.KV {
	// This will need to reference the global vault client
	// For now, return nil
	return nil
}

func (d *defaultEngineContext) GetVaultCache() map[string]map[string]interface{} {
	// Reference global vault cache
	return make(map[string]map[string]interface{})
}

func (d *defaultEngineContext) SetVaultCache(path string, data map[string]interface{}) {
	// Update global vault cache
}

func (d *defaultEngineContext) AddVaultRef(path string, keys []string) {
	VaultRefs[path] = keys
}

func (d *defaultEngineContext) IsVaultSkipped() bool {
	return SkipVault
}

func (d *defaultEngineContext) GetAWSSession() *session.Session {
	return nil
}

func (d *defaultEngineContext) GetSecretsManagerClient() secretsmanageriface.SecretsManagerAPI {
	return nil
}

func (d *defaultEngineContext) GetParameterStoreClient() ssmiface.SSMAPI {
	return nil
}

func (d *defaultEngineContext) GetAWSSecretsCache() map[string]string {
	return make(map[string]string)
}

func (d *defaultEngineContext) SetAWSSecretCache(key, value string) {
	// Update global AWS cache
}

func (d *defaultEngineContext) GetAWSParamsCache() map[string]string {
	return make(map[string]string)
}

func (d *defaultEngineContext) SetAWSParamCache(key, value string) {
	// Update global AWS cache
}

func (d *defaultEngineContext) IsAWSSkipped() bool {
	return SkipAws
}

func (d *defaultEngineContext) GetUsedIPs() map[string]string {
	// Reference global used IPs
	return make(map[string]string)
}

func (d *defaultEngineContext) SetUsedIP(key, ip string) {
	// Update global used IPs
}

func (d *defaultEngineContext) AddKeyToPrune(key string) {
	keysToPrune = append(keysToPrune, key)
}

func (d *defaultEngineContext) GetKeysToPrune() []string {
	return keysToPrune
}

func (d *defaultEngineContext) AddPathToSort(path, order string) {
	// Update global sort paths
}

func (d *defaultEngineContext) GetPathsToSort() map[string]string {
	return make(map[string]string)
}