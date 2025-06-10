package operators

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/geofffranks/yaml"
	"github.com/wayneeseguin/graft/internal/utils/ansi"
	"github.com/wayneeseguin/graft/internal/utils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// awsSession holds a shared AWS session struct
var awsSession *session.Session

// secretsManagerClient holds a secretsmanager client configured with a session
// We use secretsmanageriface.SecretsManagerAPI to be able to provide mocks in testing
var secretsManagerClient secretsmanageriface.SecretsManagerAPI

// parameterstoreClient holds a parameterstore client configured with a session
// We use ssmiface.SSMAPI to be able to provide mocks in testing
var parameterstoreClient ssmiface.SSMAPI

// awsSecretsCache caches values from AWS Secretsmanager
var awsSecretsCache = make(map[string]string)

// awsParamsCache caches values from AWS Parameterstore
var awsParamsCache = make(map[string]string)

// SkipAws toggles whether AwsOperator will attempt to query AWS for any value
// When true will always return "REDACTED"
var SkipAws bool

// AwsTarget represents an AWS target configuration
type AwsTarget struct {
	Region             string        `yaml:"region"`
	Profile            string        `yaml:"profile"`
	Role               string        `yaml:"role"`
	AccessKeyID        string        `yaml:"access_key_id"`
	SecretAccessKey    string        `yaml:"secret_access_key"`
	SessionToken       string        `yaml:"session_token"`
	Endpoint           string        `yaml:"endpoint"`
	S3ForcePathStyle   bool          `yaml:"s3_force_path_style"`
	DisableSSL         bool          `yaml:"disable_ssl"`
	MaxRetries         int           `yaml:"max_retries"`
	HTTPTimeout        time.Duration `yaml:"http_timeout"`
	CacheTTL           time.Duration `yaml:"cache_ttl"`
	AssumeRoleDuration time.Duration `yaml:"assume_role_duration"`
	ExternalID         string        `yaml:"external_id"`
	SessionName        string        `yaml:"session_name"`
	MfaSerial          string        `yaml:"mfa_serial"`
	AuditLogging       bool          `yaml:"audit_logging"`
}

// AwsClientPool manages AWS sessions and clients for different targets
type AwsClientPool struct {
	mu                    sync.RWMutex
	sessions              map[string]*session.Session
	secretsManagerClients map[string]secretsmanageriface.SecretsManagerAPI
	parameterStoreClients map[string]ssmiface.SSMAPI
	configs               map[string]*AwsTarget
	secretsCache          map[string]map[string]string // target -> secret -> value
	paramsCache           map[string]map[string]string // target -> param -> value
}

// Global client pool for target-aware AWS connections
var awsTargetPool = &AwsClientPool{
	sessions:              make(map[string]*session.Session),
	secretsManagerClients: make(map[string]secretsmanageriface.SecretsManagerAPI),
	parameterStoreClients: make(map[string]ssmiface.SSMAPI),
	configs:               make(map[string]*AwsTarget),
	secretsCache:          make(map[string]map[string]string),
	paramsCache:           make(map[string]map[string]string),
}

// GetSession returns an AWS session for the specified target
func (acp *AwsClientPool) GetSession(targetName string) (*session.Session, error) {
	acp.mu.RLock()
	if session, exists := acp.sessions[targetName]; exists {
		acp.mu.RUnlock()
		return session, nil
	}
	acp.mu.RUnlock()

	// Get target configuration
	config, err := acp.getTargetConfig(targetName)
	if err != nil {
		return nil, fmt.Errorf("AWS target '%s' not found: %v", targetName, err)
	}

	// Create AWS session from target config
	session, err := acp.createSessionFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session for target '%s': %v", targetName, err)
	}

	// Store session for reuse
	acp.mu.Lock()
	acp.sessions[targetName] = session
	acp.configs[targetName] = config
	acp.mu.Unlock()

	return session, nil
}

// GetSecretsManagerClient returns a Secrets Manager client for the specified target
func (acp *AwsClientPool) GetSecretsManagerClient(targetName string) (secretsmanageriface.SecretsManagerAPI, error) {
	acp.mu.RLock()
	if client, exists := acp.secretsManagerClients[targetName]; exists {
		acp.mu.RUnlock()
		return client, nil
	}
	acp.mu.RUnlock()

	// Get session for this target
	session, err := acp.GetSession(targetName)
	if err != nil {
		return nil, err
	}

	// Create Secrets Manager client
	client := secretsmanager.New(session)

	// Store client for reuse
	acp.mu.Lock()
	acp.secretsManagerClients[targetName] = client
	acp.mu.Unlock()

	return client, nil
}

// GetParameterStoreClient returns a Parameter Store client for the specified target
func (acp *AwsClientPool) GetParameterStoreClient(targetName string) (ssmiface.SSMAPI, error) {
	acp.mu.RLock()
	if client, exists := acp.parameterStoreClients[targetName]; exists {
		acp.mu.RUnlock()
		return client, nil
	}
	acp.mu.RUnlock()

	// Get session for this target
	session, err := acp.GetSession(targetName)
	if err != nil {
		return nil, err
	}

	// Create Parameter Store client
	client := ssm.New(session)

	// Store client for reuse
	acp.mu.Lock()
	acp.parameterStoreClients[targetName] = client
	acp.mu.Unlock()

	return client, nil
}

// getTargetConfig retrieves target configuration from environment variables
func (acp *AwsClientPool) getTargetConfig(targetName string) (*AwsTarget, error) {
	// Check if we have cached config
	acp.mu.RLock()
	if config, exists := acp.configs[targetName]; exists {
		acp.mu.RUnlock()
		return config, nil
	}
	acp.mu.RUnlock()

	// Use environment variables with target suffix
	envPrefix := fmt.Sprintf("AWS_%s_", strings.ToUpper(targetName))

	// Check if any AWS target-specific environment variables are set
	region := os.Getenv(envPrefix + "REGION")
	profile := os.Getenv(envPrefix + "PROFILE")
	role := os.Getenv(envPrefix + "ROLE")
	accessKeyID := os.Getenv(envPrefix + "ACCESS_KEY_ID")

	// Require at least one target-specific configuration
	if region == "" && profile == "" && role == "" && accessKeyID == "" {
		return nil, fmt.Errorf("AWS target '%s' configuration incomplete (expected %sREGION, %sPROFILE, %sROLE, or %sACCESS_KEY_ID environment variable)",
			targetName, envPrefix, envPrefix, envPrefix, envPrefix)
	}

	config := &AwsTarget{
		Region:             getEnvOrDefault(envPrefix+"REGION", ""),
		Profile:            getEnvOrDefault(envPrefix+"PROFILE", ""),
		Role:               getEnvOrDefault(envPrefix+"ROLE", ""),
		AccessKeyID:        getEnvOrDefault(envPrefix+"ACCESS_KEY_ID", ""),
		SecretAccessKey:    getEnvOrDefault(envPrefix+"SECRET_ACCESS_KEY", ""),
		SessionToken:       getEnvOrDefault(envPrefix+"SESSION_TOKEN", ""),
		Endpoint:           getEnvOrDefault(envPrefix+"ENDPOINT", ""),
		S3ForcePathStyle:   parseBoolOrDefault(getEnvOrDefault(envPrefix+"S3_FORCE_PATH_STYLE", "false"), false),
		DisableSSL:         parseBoolOrDefault(getEnvOrDefault(envPrefix+"DISABLE_SSL", "false"), false),
		MaxRetries:         parseIntOrDefault(getEnvOrDefault(envPrefix+"MAX_RETRIES", "3"), 3),
		HTTPTimeout:        parseDurationOrDefault(getEnvOrDefault(envPrefix+"HTTP_TIMEOUT", "30s"), 30*time.Second),
		CacheTTL:           parseDurationOrDefault(getEnvOrDefault(envPrefix+"CACHE_TTL", "5m"), 5*time.Minute),
		AssumeRoleDuration: parseDurationOrDefault(getEnvOrDefault(envPrefix+"ASSUME_ROLE_DURATION", "1h"), 1*time.Hour),
		ExternalID:         getEnvOrDefault(envPrefix+"EXTERNAL_ID", ""),
		SessionName:        getEnvOrDefault(envPrefix+"SESSION_NAME", "graft-"+targetName),
		MfaSerial:          getEnvOrDefault(envPrefix+"MFA_SERIAL", ""),
		AuditLogging:       parseBoolOrDefault(getEnvOrDefault(envPrefix+"AUDIT_LOGGING", "false"), false),
	}

	return config, nil
}

// createSessionFromConfig creates an AWS session from target configuration
func (acp *AwsClientPool) createSessionFromConfig(config *AwsTarget) (*session.Session, error) {
	options := session.Options{
		Config:            aws.Config{},
		SharedConfigState: session.SharedConfigEnable,
	}

	// Configure region
	if config.Region != "" {
		options.Config.Region = aws.String(config.Region)
	}

	// Configure profile
	if config.Profile != "" {
		options.Profile = config.Profile
	}

	// Configure endpoint (for testing or custom endpoints)
	if config.Endpoint != "" {
		options.Config.Endpoint = aws.String(config.Endpoint)
	}

	// Configure S3 path style
	if config.S3ForcePathStyle {
		options.Config.S3ForcePathStyle = aws.Bool(true)
	}

	// Configure SSL
	if config.DisableSSL {
		options.Config.DisableSSL = aws.Bool(true)
	}

	// Configure retries
	if config.MaxRetries > 0 {
		options.Config.MaxRetries = aws.Int(config.MaxRetries)
	}

	// Configure HTTP timeout (this would require additional configuration in practice)
	// HTTPTimeout is not directly available in aws.Config but would be handled by custom transport

	// Configure credentials if provided
	if config.AccessKeyID != "" && config.SecretAccessKey != "" {
		options.Config.Credentials = credentials.NewStaticCredentials(
			config.AccessKeyID,
			config.SecretAccessKey,
			config.SessionToken,
		)
	}

	// Create base session
	sess, err := session.NewSessionWithOptions(options)
	if err != nil {
		return nil, err
	}

	// Configure role assumption if provided
	if config.Role != "" {
		assumeRoleFunc := func(p *stscreds.AssumeRoleProvider) {
			if config.AssumeRoleDuration > 0 {
				p.Duration = config.AssumeRoleDuration
			}
			if config.ExternalID != "" {
				p.ExternalID = aws.String(config.ExternalID)
			}
			if config.SessionName != "" {
				p.RoleSessionName = config.SessionName
			}
			if config.MfaSerial != "" {
				p.SerialNumber = aws.String(config.MfaSerial)
				// Note: MFA token input would need to be handled separately
			}
		}

		creds := stscreds.NewCredentials(sess, config.Role, assumeRoleFunc)
		roleConfig := aws.Config{Credentials: creds}
		if config.Region != "" {
			roleConfig.Region = aws.String(config.Region)
		}
		sess, err = session.NewSession(&roleConfig)
		if err != nil {
			return nil, err
		}
	}

	return sess, nil
}

// GetSecretCache returns the secrets cache for a target
func (acp *AwsClientPool) GetSecretCache(targetName string) map[string]string {
	acp.mu.RLock()
	defer acp.mu.RUnlock()

	if cache, exists := acp.secretsCache[targetName]; exists {
		return cache
	}

	acp.mu.RUnlock()
	acp.mu.Lock()
	acp.secretsCache[targetName] = make(map[string]string)
	cache := acp.secretsCache[targetName]
	acp.mu.Unlock()
	acp.mu.RLock()

	return cache
}

// GetParamCache returns the parameters cache for a target
func (acp *AwsClientPool) GetParamCache(targetName string) map[string]string {
	acp.mu.RLock()
	defer acp.mu.RUnlock()

	if cache, exists := acp.paramsCache[targetName]; exists {
		return cache
	}

	acp.mu.RUnlock()
	acp.mu.Lock()
	acp.paramsCache[targetName] = make(map[string]string)
	cache := acp.paramsCache[targetName]
	acp.mu.Unlock()
	acp.mu.RLock()

	return cache
}

// SetSecretCache sets a secret value in the cache for a target
func (acp *AwsClientPool) SetSecretCache(targetName, secret, value string) {
	acp.mu.Lock()
	defer acp.mu.Unlock()

	if _, exists := acp.secretsCache[targetName]; !exists {
		acp.secretsCache[targetName] = make(map[string]string)
	}
	acp.secretsCache[targetName][secret] = value
}

// SetParamCache sets a parameter value in the cache for a target
func (acp *AwsClientPool) SetParamCache(targetName, param, value string) {
	acp.mu.Lock()
	defer acp.mu.Unlock()

	if _, exists := acp.paramsCache[targetName]; !exists {
		acp.paramsCache[targetName] = make(map[string]string)
	}
	acp.paramsCache[targetName][param] = value
}

// AwsOperator provides two operators;  (( awsparam "path" )) and (( awssecret "name_or_arn" ))
// It will fetch parameters / secrets from the respective AWS service
type AwsOperator struct {
	variant string
}

// extractTarget extracts target name from operator call (placeholder)
func (o AwsOperator) extractTarget(ev *Evaluator, args []*Expr) string {
	// TODO: Extract target from parsed expression when parser supports it
	// For now, return empty string to use default configuration
	return ""
}

// getCacheKey generates a cache key that includes target information
func (o AwsOperator) getCacheKey(target, variant, key string) string {
	if target == "" {
		return fmt.Sprintf("%s:%s", variant, key)
	}
	return fmt.Sprintf("%s@%s:%s", target, variant, key)
}

// initializeAwsSession will configure an AWS session with profile, region and role assume including loading shared config (e.g. ~/.aws/credentials)
func initializeAwsSession(profile string, region string, role string) (s *session.Session, err error) {
	options := session.Options{
		Config:            aws.Config{},
		SharedConfigState: session.SharedConfigEnable,
	}

	if region != "" {
		options.Config.Region = aws.String(region)
	}

	if profile != "" {
		options.Profile = profile
	}

	s, err = session.NewSessionWithOptions(options)
	if err != nil {
		return nil, err
	}

	if role != "" {
		options.Config.Credentials = stscreds.NewCredentials(s, role, func(p *stscreds.AssumeRoleProvider) {})
		s, err = session.NewSession(&options.Config)
	}

	return s, err
}

// getAwsSecret will fetch the specified secret from AWS Secretsmanager at the specified (if provided) stage / version
func getAwsSecret(awsSession *session.Session, secret string, params url.Values) (string, error) {
	val, cached := awsSecretsCache[secret]
	if cached {
		return val, nil
	}

	if secretsManagerClient == nil {
		secretsManagerClient = secretsmanager.New(awsSession)
	}

	input := secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secret),
	}

	if params.Get("stage") != "" {
		input.VersionStage = aws.String(params.Get("stage"))
	} else if params.Get("version") != "" {
		input.VersionId = aws.String(params.Get("version"))
	}

	output, err := secretsManagerClient.GetSecretValue(&input)
	if err != nil {
		return "", err
	}

	awsSecretsCache[secret] = aws.StringValue(output.SecretString)

	return awsSecretsCache[secret], nil
}

// getAwsParam will fetch the specified parameter from AWS SSM Parameterstore
func getAwsParam(awsSession *session.Session, param string) (string, error) {
	val, cached := awsParamsCache[param]
	if cached {
		return val, nil
	}

	if parameterstoreClient == nil {
		parameterstoreClient = ssm.New(awsSession)
	}

	input := ssm.GetParameterInput{
		Name:           aws.String(param),
		WithDecryption: aws.Bool(true),
	}

	output, err := parameterstoreClient.GetParameter(&input)
	if err != nil {
		return "", err
	}

	awsParamsCache[param] = aws.StringValue(output.Parameter.Value)

	return awsParamsCache[param], nil
}

// Setup ...
func (AwsOperator) Setup() error {
	return nil
}

// Phase ...
func (AwsOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies is not used by AwsOperator
func (AwsOperator) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run will invoke the appropriate getAws* function for each instance of the AwsOperator
// and extract the specified key (if provided).
func (o AwsOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	var err error
	DEBUG("running (( %s ... )) operation at $.%s", o.variant, ev.Here)
	defer DEBUG("done with (( %s ... )) operation at $.%s\n", o.variant, ev.Here)

	if len(args) < 1 {
		return nil, fmt.Errorf("%s operator requires at least one argument", o.variant)
	}

	var l []string
	for i, arg := range args {
		// Use ResolveOperatorArgument to support nested expressions
		val, err := ResolveOperatorArgument(ev, arg)
		if err != nil {
			DEBUG("  arg[%d]: failed to resolve expression to a concrete value", i)
			DEBUG("     [%d]: error was: %s", i, err)
			return nil, err
		}

		if val == nil {
			DEBUG("  arg[%d]: resolved to nil", i)
			return nil, fmt.Errorf("%s operator argument cannot be nil", o.variant)
		}

		switch v := val.(type) {
		case string:
			DEBUG("  arg[%d]: using string value '%v'", i, v)
			l = append(l, v)

		case int, int64, float64, bool:
			DEBUG("  arg[%d]: converting %T to string", i, v)
			l = append(l, fmt.Sprintf("%v", v))

		case map[interface{}]interface{}, map[string]interface{}:
			DEBUG("  arg[%d]: %v is not a string scalar", i, v)
			return nil, ansi.Errorf("@R{%s operator argument is a map; only scalars are supported here}", o.variant)

		case []interface{}:
			DEBUG("  arg[%d]: %v is not a string scalar", i, v)
			return nil, ansi.Errorf("@R{%s operator argument is a list; only scalars are supported here}", o.variant)

		default:
			DEBUG("  arg[%d]: using value of type %T as string", i, val)
			l = append(l, fmt.Sprintf("%v", val))
		}
	}

	key, params, err := parseAwsOpKey(strings.Join(l, ""))
	if err != nil {
		return nil, err
	}

	DEBUG("     [0]: Using %s key '%s'\n", o.variant, key)

	// Extract target information (placeholder for now)
	targetName := o.extractTarget(ev, args)

	var value string
	if !SkipAws {
		if targetName != "" {
			// Use target-aware client pool
			value, err = o.getValueFromTarget(targetName, key, params)
		} else {
			// Use default behavior
			engine := graft.GetEngine(ev)
			session := engine.GetOperatorState().GetAWSSession()
			if session == nil {
				session, err = initializeAwsSession(os.Getenv("AWS_PROFILE"), os.Getenv("AWS_REGION"), os.Getenv("AWS_ROLE"))
				if err != nil {
					return nil, fmt.Errorf("error during AWS session initialization: %s", err)
				}
			}

			if o.variant == "awsparam" {
				value, err = o.getAwsParamFromEngine(engine, session, key)
			} else if o.variant == "awssecret" {
				value, err = o.getAwsSecretFromEngine(engine, session, key, params)
			}
		}

		if err != nil {
			return nil, fmt.Errorf("$.%s error fetching %s: %s", key, o.variant, err)
		}

		subkey := params.Get("key")
		if subkey != "" {
			tmp := make(map[string]interface{})
			err := yaml.Unmarshal([]byte(value), &tmp)

			if err != nil {
				return nil, fmt.Errorf("$.%s error extracting key: %s", key, err)
			}

			if _, ok := tmp[subkey]; !ok {
				return nil, fmt.Errorf("$.%s invalid key '%s'", key, subkey)
			}

			value = fmt.Sprintf("%v", tmp[subkey])
		}
	} else {
		// Return skip message when AWS is skipped
		if targetName != "" {
			value = fmt.Sprintf("<skipped for %s@%s[%s]>", o.variant, targetName, key)
		} else {
			value = fmt.Sprintf("<skipped for %s[%s]>", o.variant, key)
		}
	}

	return &Response{
		Type:  Replace,
		Value: value,
	}, nil
}

// getValueFromTarget retrieves a value from AWS using target-specific clients
func (o AwsOperator) getValueFromTarget(targetName, key string, params url.Values) (string, error) {
	config, err := awsTargetPool.getTargetConfig(targetName)
	if err != nil {
		return "", err
	}

	// Audit logging
	if config.AuditLogging {
		DEBUG("AUDIT: Accessing AWS %s: %s (target: %s)", o.variant, key, targetName)
	}

	// Check cache first with target-aware key
	_ = o.getCacheKey(targetName, o.variant, key)

	if o.variant == "awsparam" {
		cache := awsTargetPool.GetParamCache(targetName)
		if val, cached := cache[key]; cached {
			if config.AuditLogging {
				DEBUG("AUDIT: Cache hit for %s parameter: %s (target: %s)", o.variant, key, targetName)
			}
			return val, nil
		}

		// Get Parameter Store client for this target
		client, err := awsTargetPool.GetParameterStoreClient(targetName)
		if err != nil {
			return "", err
		}

		input := &ssm.GetParameterInput{
			Name:           aws.String(key),
			WithDecryption: aws.Bool(true),
		}

		output, err := client.GetParameter(input)
		if err != nil {
			if config.AuditLogging {
				DEBUG("AUDIT: Failed to retrieve parameter: %s (target: %s) - %v", key, targetName, err)
			}
			return "", err
		}

		value := aws.StringValue(output.Parameter.Value)
		awsTargetPool.SetParamCache(targetName, key, value)

		if config.AuditLogging {
			DEBUG("AUDIT: Successfully retrieved parameter: %s (target: %s)", key, targetName)
		}

		return value, nil

	} else if o.variant == "awssecret" {
		cache := awsTargetPool.GetSecretCache(targetName)
		if val, cached := cache[key]; cached {
			if config.AuditLogging {
				DEBUG("AUDIT: Cache hit for %s secret: %s (target: %s)", o.variant, key, targetName)
			}
			return val, nil
		}

		// Get Secrets Manager client for this target
		client, err := awsTargetPool.GetSecretsManagerClient(targetName)
		if err != nil {
			return "", err
		}

		input := &secretsmanager.GetSecretValueInput{
			SecretId: aws.String(key),
		}

		if params.Get("stage") != "" {
			input.VersionStage = aws.String(params.Get("stage"))
		} else if params.Get("version") != "" {
			input.VersionId = aws.String(params.Get("version"))
		}

		output, err := client.GetSecretValue(input)
		if err != nil {
			if config.AuditLogging {
				DEBUG("AUDIT: Failed to retrieve secret: %s (target: %s) - %v", key, targetName, err)
			}
			return "", err
		}

		value := aws.StringValue(output.SecretString)
		awsTargetPool.SetSecretCache(targetName, key, value)

		if config.AuditLogging {
			DEBUG("AUDIT: Successfully retrieved secret: %s (target: %s)", key, targetName)
		}

		return value, nil
	}

	return "", fmt.Errorf("unknown AWS operator variant: %s", o.variant)
}

// parseAwsOpKey parsed the parameters passed to AwsOperator.
// Primarily it splits the key from the extra arguments (specified as a query string)
func parseAwsOpKey(key string) (string, url.Values, error) {
	split := strings.SplitN(key, "?", 2)
	if len(split) == 1 {
		split = append(split, "")
	}

	values, err := url.ParseQuery(split[1])
	if err != nil {
		return "", values, fmt.Errorf("invalid argument string: %s", err)
	}

	return split[0], values, nil
}

// getAwsSecretFromEngine fetches a secret using the engine
func (o AwsOperator) getAwsSecretFromEngine(engine graft.Engine, awsSession *session.Session, secret string, params url.Values) (string, error) {
	cache := engine.GetOperatorState().GetAWSSecretsCache()
	if val, cached := cache[secret]; cached {
		return val, nil
	}

	client := engine.GetOperatorState().GetSecretsManagerClient()
	if client == nil {
		client = secretsmanager.New(awsSession)
	}

	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secret),
	}

	if params.Get("stage") != "" {
		input.VersionStage = aws.String(params.Get("stage"))
	} else if params.Get("version") != "" {
		input.VersionId = aws.String(params.Get("version"))
	}

	output, err := client.GetSecretValue(input)
	if err != nil {
		return "", err
	}

	value := aws.StringValue(output.SecretString)
	engine.GetOperatorState().SetAWSSecretCache(secret, value)
	return value, nil
}

// getAwsParamFromEngine fetches a parameter using the engine context
func (o AwsOperator) getAwsParamFromEngine(engine graft.Engine, awsSession *session.Session, param string) (string, error) {
	cache := engine.GetOperatorState().GetAWSParamsCache()
	if val, cached := cache[param]; cached {
		return val, nil
	}

	client := engine.GetOperatorState().GetParameterStoreClient()
	if client == nil {
		client = ssm.New(awsSession)
	}

	input := &ssm.GetParameterInput{
		Name:           aws.String(param),
		WithDecryption: aws.Bool(true),
	}

	output, err := client.GetParameter(input)
	if err != nil {
		return "", err
	}

	value := aws.StringValue(output.Parameter.Value)
	engine.GetOperatorState().SetAWSParamCache(param, value)
	return value, nil
}

// NewAwsParamOperator creates a new AWS Parameter Store operator
func NewAwsParamOperator() *AwsOperator {
	return &AwsOperator{variant: "awsparam"}
}

// NewAwsSecretOperator creates a new AWS Secrets Manager operator
func NewAwsSecretOperator() *AwsOperator {
	return &AwsOperator{variant: "awssecret"}
}

// Helper functions for environment variable parsing

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseDurationOrDefault parses duration string or returns default
func parseDurationOrDefault(value string, defaultValue time.Duration) time.Duration {
	if d, err := time.ParseDuration(value); err == nil {
		return d
	}
	return defaultValue
}

// parseIntOrDefault parses integer string or returns default
func parseIntOrDefault(value string, defaultValue int) int {
	if i, err := strconv.Atoi(value); err == nil {
		return i
	}
	return defaultValue
}

// parseBoolOrDefault parses boolean string or returns default
func parseBoolOrDefault(value string, defaultValue bool) bool {
	if b, err := strconv.ParseBool(value); err == nil {
		return b
	}
	return defaultValue
}

// init registers the two variants of the AwsOperator
func init() {
	RegisterOp("awsparam", AwsOperator{variant: "awsparam"})
	RegisterOp("awssecret", AwsOperator{variant: "awssecret"})
}
