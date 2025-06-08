package operators

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/cloudfoundry-community/vaultkv"
	"github.com/geofffranks/yaml"
	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// globalKV is the global vault client for backward compatibility
var globalKV *vaultkv.KV = nil

var vaultSecretCache = map[string]map[string]interface{}{}

// VaultRefs maps secret path to paths in YAML structure which call for it
var VaultRefs = map[string][]string{}

// VaultTarget represents a vault target configuration
type VaultTarget struct {
	URL        string `yaml:"url"`
	Token      string `yaml:"token"`
	Namespace  string `yaml:"namespace"`
	SkipVerify bool   `yaml:"skip_verify"`
}

// VaultClientPool manages vault clients for different targets
type VaultClientPool struct {
	clients map[string]*vaultkv.KV
	configs map[string]*VaultTarget
}

// Global client pool for target-aware vault clients
var vaultClientPool = &VaultClientPool{
	clients: make(map[string]*vaultkv.KV),
	configs: make(map[string]*VaultTarget),
}

// GetClient returns a vault client for the specified target
func (vcp *VaultClientPool) GetClient(targetName string, engine graft.Engine) (*vaultkv.KV, error) {
	// Return existing client if available
	if client, exists := vcp.clients[targetName]; exists {
		return client, nil
	}
	
	// Get target configuration
	config, err := vcp.getTargetConfig(targetName, engine)
	if err != nil {
		return nil, fmt.Errorf("vault target '%s' not found: %v", targetName, err)
	}
	
	// Create new client
	client, err := createVaultClientFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client for target '%s': %v", targetName, err)
	}
	
	// Store client for reuse
	vcp.clients[targetName] = client
	vcp.configs[targetName] = config
	
	return client, nil
}

// getTargetConfig retrieves target configuration from the engine or environment
func (vcp *VaultClientPool) getTargetConfig(targetName string, engine graft.Engine) (*VaultTarget, error) {
	// Check if we have cached config
	if config, exists := vcp.configs[targetName]; exists {
		return config, nil
	}
	
	// For now, try environment variables with target suffix
	// In a full implementation, this would query the engine's configuration
	envPrefix := fmt.Sprintf("VAULT_%s_", strings.ToUpper(targetName))
	
	config := &VaultTarget{
		URL:       os.Getenv(envPrefix + "ADDR"),
		Token:     os.Getenv(envPrefix + "TOKEN"),
		Namespace: os.Getenv(envPrefix + "NAMESPACE"),
	}
	
	// Check for skip verify
	if skipStr := os.Getenv(envPrefix + "SKIP_VERIFY"); skipStr == "true" || skipStr == "1" {
		config.SkipVerify = true
	}
	
	// If no environment variables found, return error
	if config.URL == "" || config.Token == "" {
		return nil, fmt.Errorf("vault target '%s' configuration not found (expected %sADDR and %sTOKEN environment variables)", 
			targetName, envPrefix, envPrefix)
	}
	
	return config, nil
}

// createVaultClientFromConfig creates a vault client from target configuration
func createVaultClientFromConfig(config *VaultTarget) (*vaultkv.KV, error) {
	// Expand environment variables in configuration
	addr := os.ExpandEnv(config.URL)
	token := os.ExpandEnv(config.Token)
	namespace := os.ExpandEnv(config.Namespace)
	
	parsedURL, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("could not parse Vault URL `%s': %s", addr, err)
	}
	
	// Port handling
	if parsedURL.Port() == "" {
		if parsedURL.Scheme == "http" {
			parsedURL.Host = parsedURL.Host + ":80"
		} else {
			parsedURL.Host = parsedURL.Host + ":443"
		}
	}
	
	// TLS configuration
	roots, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve system root certificate authorities: %s", err)
	}
	
	client := &vaultkv.Client{
		AuthToken: token,
		VaultURL:  parsedURL,
		Namespace: namespace,
		Client: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{
					RootCAs:            roots,
					InsecureSkipVerify: config.SkipVerify, // #nosec G402 - controlled by user configuration
				},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				req.Header.Add("X-Vault-Token", token)
				req.Header.Add("X-Vault-Namespace", namespace)
				return nil
			},
		},
	}
	
	// Enable tracing if debug is on
	if DebugOn() {
		client.Trace = os.Stderr
	}
	
	return client.NewKV(), nil
}

// extractTarget attempts to extract target information from the operator context
func (o VaultOperator) extractTarget(ev *Evaluator, args []*Expr) string {
	// For now, we'll implement a simple approach where target information
	// could be stored in the engine state or extracted from the evaluator context.
	// In a full implementation, this would access the parsed operator call's target field.
	
	// TODO: This is a placeholder implementation. In the complete implementation,
	// the target would be available from the operator call context.
	// For now, we'll return empty string (no target) to maintain backward compatibility.
	return ""
}

// getCacheKey generates a cache key that includes target information
func (o VaultOperator) getCacheKey(target, path string) string {
	if target == "" {
		return path
	}
	return fmt.Sprintf("%s@%s", target, path)
}

// SkipVault toggles whether calls to the Vault operator actually cause the
// Vault to be contacted and the keys substituted in.
var SkipVault bool

// vaultArgProcessor handles argument processing for vault operator with LogicalOr support
type vaultArgProcessor struct {
	args         []*Expr
	hasDefault   bool
	defaultExpr  *Expr
	defaultIndex int
	hasSubOps    bool // Track if sub-operators are used
}

// newVaultArgProcessor creates a processor that extracts defaults from any position
func newVaultArgProcessor(args []*Expr) *vaultArgProcessor {
	processor := &vaultArgProcessor{
		args:         make([]*Expr, len(args)),
		hasDefault:   false,
		defaultIndex: -1,
		hasSubOps:    false,
	}

	// Check for sub-operators and parse if needed
	parsedArgs, hasSubOps, err := ParseVaultArgs(args)
	if err != nil {
		// If parsing fails, fall back to original args
		parsedArgs = args
		hasSubOps = false
	}
	processor.hasSubOps = hasSubOps

	// Copy args and extract any LogicalOr
	for i, arg := range parsedArgs {
		if arg.Type == LogicalOr {
			processor.hasDefault = true
			processor.defaultExpr = arg.Right
			processor.defaultIndex = i
			// Use the left side of LogicalOr for vault path
			processor.args[i] = arg.Left
		} else {
			processor.args[i] = arg
		}
	}

	return processor
}

// isVaultPathString checks if an expression looks like a vault path (contains colon)
func isVaultPathString(ev *Evaluator, expr *Expr) bool {
	// Try to resolve to string without error propagation
	if expr.Type == Literal {
		if str, ok := expr.Literal.(string); ok {
			return strings.Contains(str, ":")
		}
	}
	return false
}

// detectMultiplePathArgs checks if we have multiple vault path arguments
func (p *vaultArgProcessor) detectMultiplePathArgs(ev *Evaluator) bool {
	// If we have LogicalOr, we're in classic mode
	if p.hasDefault {
		return false
	}
	
	// Check if we have multiple arguments that look like vault paths
	pathCount := 0
	for _, arg := range p.args {
		if isVaultPathString(ev, arg) {
			pathCount++
		}
	}
	
	return pathCount > 1
}

// resolveToString resolves an expression and converts it to a string
func (p *vaultArgProcessor) resolveToString(ev *Evaluator, expr *Expr) (string, error) {
	// Check if we need to handle sub-operators
	if p.hasSubOps {
		result, err := p.resolveWithSubOperators(ev, expr)
		if err != nil {
			return "", err
		}
		// Convert result to string
		return p.convertToString(result, expr)
	}

	// Use ResolveOperatorArgument to support nested expressions
	value, err := ResolveOperatorArgument(ev, expr)
	if err != nil {
		// Maintain backward compatibility with error messages
		if expr.Type == Reference {
			return "", fmt.Errorf("Unable to resolve `%s`: %s", expr.Reference, err)
		}
		return "", err
	}

	if value == nil {
		return "", fmt.Errorf("cannot use nil as vault path component")
	}

	// Convert resolved value to string with vault-specific error messages
	return p.convertToString(value, expr)
}

// convertToString converts a value to string with vault-specific error messages
func (p *vaultArgProcessor) convertToString(value interface{}, expr *Expr) (string, error) {
	if value == nil {
		return "", fmt.Errorf("cannot use nil as vault path component")
	}

	switch v := value.(type) {
	case string:
		return v, nil
	case int, int64, float32, float64, bool:
		return fmt.Sprintf("%v", v), nil
	case map[interface{}]interface{}, map[string]interface{}:
		if expr != nil && expr.Type == Reference {
			return "", fmt.Errorf("$.%s is a map; only scalars are supported for vault paths", expr.Reference)
		}
		return "", fmt.Errorf("value is a map; only scalars are supported for vault paths")
	case []interface{}:
		if expr != nil && expr.Type == Reference {
			return "", fmt.Errorf("$.%s is a list; only scalars are supported for vault paths", expr.Reference)
		}
		return "", fmt.Errorf("value is a list; only scalars are supported for vault paths")
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// resolveWithSubOperators resolves expressions with sub-operator support
func (p *vaultArgProcessor) resolveWithSubOperators(ev *Evaluator, expr *Expr) (interface{}, error) {
	if expr == nil {
		return nil, fmt.Errorf("cannot resolve nil expression")
	}

	switch expr.Type {
	case graft.VaultGroup:
		// Resolve grouped expression
		return p.resolveGroup(ev, expr)
	case graft.VaultChoice:
		// Resolve choice expression (try alternatives)
		return p.resolveChoice(ev, expr)
	default:
		// Fall back to standard resolution
		return ResolveOperatorArgument(ev, expr)
	}
}

// resolveGroup resolves a grouped expression
func (p *vaultArgProcessor) resolveGroup(ev *Evaluator, expr *Expr) (interface{}, error) {
	if expr.Left == nil {
		return nil, fmt.Errorf("empty group expression")
	}

	// Recursively resolve the inner expression
	return p.resolveWithSubOperators(ev, expr.Left)
}

// resolveChoice resolves a choice expression (try alternatives)
func (p *vaultArgProcessor) resolveChoice(ev *Evaluator, expr *Expr) (interface{}, error) {
	if expr.Left == nil && expr.Right == nil {
		return nil, fmt.Errorf("empty choice expression")
	}

	// Try left side first
	if expr.Left != nil {
		result, err := p.resolveWithSubOperators(ev, expr.Left)
		if err == nil && result != nil {
			// Left side succeeded
			return result, nil
		}
		// Left side failed or returned nil, try right side
		DEBUG("vault choice: left alternative failed (%v), trying right", err)
	}

	// Try right side
	if expr.Right != nil {
		result, err := p.resolveWithSubOperators(ev, expr.Right)
		if err == nil && result != nil {
			// Right side succeeded
			return result, nil
		}
		DEBUG("vault choice: right alternative failed (%v)", err)
	}

	// Both sides failed
	return nil, fmt.Errorf("all choice alternatives failed")
}

// buildVaultPath resolves all arguments and concatenates them into a vault path
func (p *vaultArgProcessor) buildVaultPath(ev *Evaluator) (string, error) {
	parts := make([]string, 0, len(p.args))

	for i, arg := range p.args {
		DEBUG("  processing arg[%d] for concatenation", i)

		part, err := p.resolveToString(ev, arg)
		if err != nil {
			DEBUG("    failed to resolve arg[%d]: %s", i, err)
			return "", err
		}

		DEBUG("    resolved to: '%s'", part)
		parts = append(parts, part)
	}

	path := strings.Join(parts, "")
	DEBUG("  final concatenated path: '%s'", path)

	return path, nil
}

// splitVaultPaths splits a path string by semicolons to support multiple vault paths
func (p *vaultArgProcessor) splitVaultPaths(path string) []string {
	// Check if the path contains semicolons for multiple paths
	if !strings.Contains(path, ";") {
		return []string{path}
	}

	// Split by semicolon and trim whitespace
	rawPaths := strings.Split(path, ";")
	paths := make([]string, 0, len(rawPaths))
	
	for _, p := range rawPaths {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			paths = append(paths, trimmed)
		}
	}
	
	return paths
}

// buildVaultPaths builds and returns all vault paths to try
func (p *vaultArgProcessor) buildVaultPaths(ev *Evaluator) ([]string, error) {
	// Check if we have multiple path arguments (vault-try style)
	if p.detectMultiplePathArgs(ev) && len(p.args) >= 2 {
		// Multiple arguments mode - each arg is a separate path
		// Last arg is the default unless there's a LogicalOr
		paths := make([]string, 0)
		
		argsToProcess := p.args
		if !p.hasDefault {
			// Last argument might be a default value, check if it's a path
			lastArg := p.args[len(p.args)-1]
			if !isVaultPathString(ev, lastArg) {
				// Last arg is not a path, treat it as default
				argsToProcess = p.args[:len(p.args)-1]
				p.hasDefault = true
				p.defaultExpr = lastArg
			}
		}
		
		// Process each argument as a separate path
		for i, arg := range argsToProcess {
			path, err := p.resolveToString(ev, arg)
			if err != nil {
				DEBUG("  failed to resolve path arg[%d]: %s", i, err)
				return nil, err
			}
			paths = append(paths, path)
		}
		
		DEBUG("  vault paths to try (multi-arg mode): %v", paths)
		return paths, nil
	}
	
	// Single concatenated path mode (classic)
	path, err := p.buildVaultPath(ev)
	if err != nil {
		return nil, err
	}
	
	// Then split it into multiple paths if needed (semicolon mode)
	paths := p.splitVaultPaths(path)
	
	DEBUG("  vault paths to try: %v", paths)
	return paths, nil
}

// evaluateDefault evaluates the default expression if one exists
func (p *vaultArgProcessor) evaluateDefault(ev *Evaluator) (interface{}, error) {
	if !p.hasDefault || p.defaultExpr == nil {
		return nil, fmt.Errorf("no default value available")
	}

	DEBUG("  evaluating default expression")
	// Use ResolveOperatorArgument to support nested expressions in defaults
	value, err := ResolveOperatorArgument(ev, p.defaultExpr)
	if err != nil {
		return nil, fmt.Errorf("unable to evaluate default value: %s", err)
	}

	return value, nil
}

// isVaultNotFound checks if an error indicates a missing secret
func isVaultNotFound(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "not found") ||
		strings.Contains(errMsg, "404") ||
		strings.Contains(errMsg, "secret not found")
}

// The VaultOperator provides a means of injecting credentials and
// other secrets from a Vault (vaultproject.io) Secure Key Storage
// instance.
type VaultOperator struct{}

// Setup ...
func (VaultOperator) Setup() error {
	return nil
}

// Phase identifies what phase of document management the vault
// operator should be evaluated in.  Vault lives in the Eval phase
func (VaultOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies collects implicit dependencies that a given `(( vault ... ))`
// call has. There are no dependencies other that those given as args to the
// command.
func (VaultOperator) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

func initializeVaultClient() error {
	addr := os.Getenv("VAULT_ADDR")
	token := os.Getenv("VAULT_TOKEN")
	namespace := os.Getenv("VAULT_NAMESPACE")
	skip := false

	if addr == "" || token == "" {
		svtoken := struct {
			Vault      string `yaml:"vault"`
			Token      string `yaml:"token"`
			Namespace  string `yaml:"namespace"`
			SkipVerify bool   `yaml:"skip_verify"`
		}{}
		b, err := os.ReadFile(os.ExpandEnv("${HOME}/.svtoken"))
		if err == nil {
			err = yaml.Unmarshal(b, &svtoken)
			if err == nil {
				addr = svtoken.Vault
				token = svtoken.Token
				namespace = svtoken.Namespace
				skip = svtoken.SkipVerify
			}
		}
	}

	if skipVaultVerify(os.Getenv("VAULT_SKIP_VERIFY")) {
		skip = true
	}

	if token == "" {
		b, err := os.ReadFile(fmt.Sprintf("%s/.vault-token", os.Getenv("HOME")))
		if err == nil {
			token = strings.TrimSuffix(string(b), "\n")
		}
	}

	if addr == "" || token == "" {
		return fmt.Errorf("Failed to determine Vault URL / token, and the $REDACT environment variable is not set.")
	}

	roots, err := x509.SystemCertPool()
	if err != nil {
		return fmt.Errorf("unable to retrieve system root certificate authorities: %s", err)
	}

	parsedURL, err := url.Parse(addr)
	if err != nil {
		return fmt.Errorf("Could not parse Vault URL `%s': %s", addr, err)
	}

	if parsedURL.Port() == "" {
		if parsedURL.Scheme == "http" {
			parsedURL.Host = parsedURL.Host + ":80"
		} else {
			parsedURL.Host = parsedURL.Host + ":443"
		}
	}

	client := &vaultkv.Client{
		AuthToken: token,
		VaultURL:  parsedURL,
		Namespace: namespace,
		Client: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{
					RootCAs:            roots,
					InsecureSkipVerify: skip, // #nosec G402 - skip is controlled by user configuration
				},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				req.Header.Add("X-Vault-Token", token)
				req.Header.Add("X-Vault-Namespace", token)
				return nil
			},
		},
	}
	if DebugOn() {
		client.Trace = os.Stderr
	}

	if err != nil {
		return fmt.Errorf("Error setting up Vault client: %s", err)
	}

	globalKV = client.NewKV()

	return nil
}

// Run executes the `(( vault ... ))` operator call, which entails
// interacting with the (unsealed) Vault instance to retrieve the
// given secrets.
func (o VaultOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( vault ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( vault ... )) operation at $.%s\n", ev.Here)

	// Get engine
	engine := graft.GetEngine(ev)

	// Extract target information if available
	targetName := o.extractTarget(ev, args)
	if targetName != "" {
		DEBUG("vault: using target '%s'", targetName)
	}

	// syntax: (( vault "secret/path:key" ))
	// syntax: (( vault@target "secret/path:key" ))
	// syntax: (( vault path.object "to concat with" other.object ))
	// syntax: (( vault "secret/path:key" || "default" ))
	// syntax: (( vault prefix "/" key ":password" || "default" ))
	// syntax: (( vault ( meta.vault_path meta.stub  ":" ("key1" | "key2" ) | meta.exodus_path "subpath:key1") || "default"))
	if len(args) < 1 {
		return nil, fmt.Errorf("vault operator requires at least one argument")
	}

	// Detect if we need enhanced parsing for sub-operators
	if o.needsEnhancedParsing(args) {
		DEBUG("vault: using enhanced parsing with sub-operators")
		return o.runWithSubOperators(ev, args, engine, targetName)
	}

	// Use classic implementation for backward compatibility
	DEBUG("vault: using classic parsing")
	return o.runClassic(ev, args, engine, targetName)
}

// needsEnhancedParsing checks if any arguments contain vault sub-operators
func (o VaultOperator) needsEnhancedParsing(args []*Expr) bool {
	for _, arg := range args {
		if arg == nil {
			continue
		}
		
		switch arg.Type {
		case graft.VaultGroup, graft.VaultChoice:
			return true
		case graft.Literal:
			// Check if literal contains sub-operator syntax
			if str, ok := arg.Literal.(string); ok && ContainsSubOperators(str) {
				return true
			}
		}
	}
	return false
}

// runClassic executes vault operator with classic logic (backward compatibility)
func (o VaultOperator) runClassic(ev *Evaluator, args []*Expr, engine graft.Engine, targetName string) (*Response, error) {
	// Use the existing argument processor
	processor := newVaultArgProcessor(args)

	// Build all vault paths from arguments
	paths, err := processor.buildVaultPaths(ev)
	if err != nil {
		// Failed to build paths, check if we have a default
		if processor.hasDefault {
			DEBUG("vault: failed to build paths (%s), evaluating default value", err)
			defaultValue, evalErr := processor.evaluateDefault(ev)
			if evalErr != nil {
				return nil, fmt.Errorf("unable to evaluate default value: %s", evalErr)
			}
			return &Response{
				Type:  Replace,
				Value: defaultValue,
			}, nil
		}
		return nil, err
	}

	return o.tryVaultPaths(ev, engine, paths, processor, targetName)
}

// runWithSubOperators executes vault operator with sub-operator support
func (o VaultOperator) runWithSubOperators(ev *Evaluator, args []*Expr, engine graft.Engine, targetName string) (*Response, error) {
	// Use enhanced argument processor
	processor := newVaultArgProcessor(args)

	// Build all vault paths from arguments (with sub-operator support)
	paths, err := processor.buildVaultPaths(ev)
	if err != nil {
		// Failed to build paths, check if we have a default
		if processor.hasDefault {
			DEBUG("vault: failed to build paths (%s), evaluating default value", err)
			defaultValue, evalErr := processor.evaluateDefault(ev)
			if evalErr != nil {
				return nil, fmt.Errorf("unable to evaluate default value: %s", evalErr)
			}
			return &Response{
				Type:  Replace,
				Value: defaultValue,
			}, nil
		}
		return nil, err
	}

	return o.tryVaultPaths(ev, engine, paths, processor, targetName)
}

// tryVaultPaths attempts to retrieve secrets from a list of vault paths
func (o VaultOperator) tryVaultPaths(ev *Evaluator, engine graft.Engine, paths []string, processor *vaultArgProcessor, targetName string) (*Response, error) {
	// Try each path in order
	var lastErr error
	for i, key := range paths {
		DEBUG("vault: trying path %d of %d: %s", i+1, len(paths), key)
		
		// Track vault references using engine context
		engine.GetOperatorState().AddVaultRef(key, []string{ev.Here.String()})

		// Perform the vault lookup
		secret, err := o.performVaultLookup(engine, key, targetName)
		if err == nil {
			// Success!
			DEBUG("vault: path %d succeeded", i+1)
			return &Response{
				Type:  Replace,
				Value: secret,
			}, nil
		}
		
		// Remember the last error
		lastErr = err
		DEBUG("vault: path %d failed: %s", i+1, err)
		
		// For non-404 errors on single path, fail immediately
		if len(paths) == 1 && !isVaultNotFound(err) {
			break
		}
	}

	// All paths failed, check if we should try the default
	if processor.hasDefault && (lastErr == nil || isVaultNotFound(lastErr)) {
		DEBUG("vault: all paths failed, evaluating default value")
		defaultValue, evalErr := processor.evaluateDefault(ev)
		if evalErr != nil {
			return nil, fmt.Errorf("unable to evaluate default value: %s", evalErr)
		}
		return &Response{
			Type:  Replace,
			Value: defaultValue,
		}, nil
	}

	// Return the last error
	if lastErr != nil {
		return nil, lastErr
	}
	
	// This shouldn't happen, but just in case
	return nil, fmt.Errorf("vault operator failed to retrieve secret")
}

// resolveVaultArgs handles the resolution of vault arguments
func (VaultOperator) resolveVaultArgs(ev *Evaluator, args []*Expr) (string, error) {
	var l []string
	for i, arg := range args {
		// Use ResolveOperatorArgument to support nested expressions
		val, err := ResolveOperatorArgument(ev, arg)
		if err != nil {
			DEBUG("  arg[%d]: failed to resolve expression to a concrete value", i)
			DEBUG("     [%d]: error was: %s", i, err)
			// Maintain backward compatibility with error messages
			if arg.Type == Reference {
				return "", fmt.Errorf("Unable to resolve `%s`: %s", arg.Reference, err)
			}
			return "", err
		}

		if val == nil {
			DEBUG("  arg[%d]: resolved to nil", i)
			return "", fmt.Errorf("vault operator argument cannot be nil")
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
			return "", ansi.Errorf("@R{vault operator argument is not a string scalar}")

		case []interface{}:
			DEBUG("  arg[%d]: %v is not a string scalar", i, v)
			return "", ansi.Errorf("@R{vault operator argument is not a string scalar}")

		default:
			DEBUG("  arg[%d]: using value of type %T as string", i, val)
			l = append(l, fmt.Sprintf("%v", val))
		}
	}
	key := strings.Join(l, "")
	DEBUG("     [0]: Using vault key '%s'\n", key)
	return key, nil
}

// performVaultLookup performs the actual vault lookup
func (o VaultOperator) performVaultLookup(engine graft.Engine, key string, targetName string) (string, error) {
	if engine.GetOperatorState().IsVaultSkipped() {
		return "REDACTED", nil
	}

	var kv *vaultkv.KV
	var err error
	
	if targetName != "" {
		// Use target-specific client
		kv, err = vaultClientPool.GetClient(targetName, engine)
		if err != nil {
			return "", fmt.Errorf("failed to get vault client for target '%s': %v", targetName, err)
		}
		DEBUG("vault: using target-specific client for '%s'", targetName)
	} else {
		// Fall back to default behavior (environment-based or global client)
		kv = engine.GetOperatorState().GetVaultClient()
		if kv == nil {
			// For backward compatibility, try to initialize from environment
			if SkipVault {
				return "REDACTED", nil
			}
			
			// Fall back to global initialization
			if globalKV == nil {
				err := initializeVaultClient()
				if err != nil {
					return "", fmt.Errorf("Error during Vault client initialization: %s", err)
				}
			}
			kv = globalKV
		}
		DEBUG("vault: using default client")
	}

	leftPart, rightPart := parsePath(key)
	if leftPart == "" || rightPart == "" {
		return "", ansi.Errorf("@R{invalid argument} @c{%s}@R{; must be in the form} @m{path/to/secret:key}", key)
	}

	// Check cache first (include target in cache key)
	cacheKey := o.getCacheKey(targetName, leftPart)
	vaultCache := engine.GetOperatorState().GetVaultCache()
	var fullSecret map[string]interface{}
	var found bool
	if fullSecret, found = vaultCache[cacheKey]; found {
		DEBUG("vault: Cache hit for `%s` (target: %s)", leftPart, targetName)
	} else {
		DEBUG("vault: Cache MISS for `%s` (target: %s)", leftPart, targetName)
		// Secret isn't cached. Grab it from the vault.
		var err error
		fullSecret, err = getVaultSecretWithClient(kv, leftPart)
		if err != nil {
			//Normalize the error messages
			switch err.(type) {
			case *vaultkv.ErrNotFound:
				err = fmt.Errorf("secret %s not found", key)
			}
			return "", err
		}
		engine.GetOperatorState().SetVaultCache(cacheKey, fullSecret)
	}

	secret, err := extractSubkey(fullSecret, leftPart, rightPart)
	if err != nil {
		return "", err
	}
	return secret, nil
}

// VaultTryOperator is a deprecated alias for VaultOperator
// It maintains backward compatibility but logs a deprecation warning
type VaultTryOperator struct{}

// Setup initializes the operator
func (VaultTryOperator) Setup() error {
	return nil
}

// Phase identifies when this operator runs
func (VaultTryOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies returns the dependencies for this operator
func (VaultTryOperator) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run executes vault-try by maintaining its original behavior
func (o VaultTryOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( vault-try ... )) operation at $.%s", ev.Here)
	DEBUG("WARNING: vault-try is deprecated. Consider using vault with semicolon-separated paths: (( vault \"path1:key; path2:key\" || \"default\" ))")
	defer DEBUG("done with (( vault-try ... )) operation at $.%s\n", ev.Here)

	// Minimum 2 arguments: at least one vault path and a default
	if len(args) < 2 {
		return nil, fmt.Errorf("vault-try operator requires at least 2 arguments (one or more vault paths, followed by a default value)")
	}

	// The last argument is always the default
	vaultPaths := args[:len(args)-1]
	defaultExpr := args[len(args)-1]

	// Get engine
	engine := graft.GetEngine(ev)

	// Try each vault path in order
	for i, pathExpr := range vaultPaths {
		DEBUG("vault-try: attempting path %d of %d", i+1, len(vaultPaths))

		// Resolve the path expression to a string
		val, err := ResolveOperatorArgument(ev, pathExpr)
		if err != nil {
			DEBUG("vault-try: path %d failed to resolve: %s", i+1, err)
			continue // Skip to next path
		}

		if val == nil {
			DEBUG("vault-try: path %d resolved to nil", i+1)
			continue // Skip to next path
		}

		// Convert to string
		path, err := AsString(val)
		if err != nil {
			DEBUG("vault-try: path %d is not a string: %s", i+1, err)
			continue // Skip to next path
		}

		if path == "" {
			DEBUG("vault-try: path %d is empty", i+1)
			continue // Skip to next path
		}

		// Validate path format (forgiving - just continue on malformed)
		if !strings.Contains(path, ":") {
			DEBUG("vault-try: path %d is malformed (no colon)", i+1)
			continue // Skip to next path
		}

		parts := strings.Split(path, ":")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			DEBUG("vault-try: path %d is malformed", i+1)
			continue // Skip to next path
		}

		// Track this vault reference
		engine.GetOperatorState().AddVaultRef(path, []string{ev.Here.String()})

		// Use the shared vault infrastructure
		vaultOp := VaultOperator{}
		secret, err := vaultOp.performVaultLookup(engine, path, "")
		if err == nil {
			// Success!
			DEBUG("vault-try: path %d succeeded", i+1)
			return &Response{
				Type:  Replace,
				Value: secret,
			}, nil
		}

		// Log the error but continue to next path
		DEBUG("vault-try: path %d failed: %s", i+1, err)
	}

	// All vault paths failed, use the default value
	DEBUG("vault-try: all paths failed, evaluating default value")
	defaultValue, err := ResolveOperatorArgument(ev, defaultExpr)
	if err != nil {
		return nil, fmt.Errorf("unable to evaluate default value: %s", err)
	}

	return &Response{
		Type:  Replace,
		Value: defaultValue,
	}, nil
}

func init() {
	RegisterOp("vault", VaultOperator{})
	RegisterOp("vault-try", VaultTryOperator{})
}

/****** VAULT INTEGRATION ***********************************/

func getVaultSecret(secret string) (map[string]interface{}, error) {
	ret := map[string]interface{}{}

	DEBUG("Fetching Vault secret at `%s'", secret)
	_, err := globalKV.Get(secret, &ret, nil)
	if err != nil {
		DEBUG(" failure.")
		return nil, err
	}

	DEBUG("  success.")
	return ret, nil
}

// getVaultSecretWithClient retrieves a secret using the provided client
func getVaultSecretWithClient(kvClient *vaultkv.KV, secret string) (map[string]interface{}, error) {
	ret := map[string]interface{}{}

	DEBUG("Fetching Vault secret at `%s'", secret)
	_, err := kvClient.Get(secret, &ret, nil)
	if err != nil {
		DEBUG(" failure.")
		return nil, err
	}

	DEBUG("  success.")
	return ret, nil
}

func extractSubkey(secretMap map[string]interface{}, secret, subkey string) (string, error) {
	DEBUG("  extracting the [%s] subkey from the secret", subkey)

	secretSubkeyPath := fmt.Sprintf("%s:%s", secret, subkey)
	v, ok := secretMap[subkey]
	if !ok {
		DEBUG("    !! %s not found!\n", secretSubkeyPath)
		return "", ansi.Errorf("@R{secret} @c{%s} @R{not found}", secretSubkeyPath)
	}
	if _, ok := v.(string); !ok {
		DEBUG("    !! %s is not a string!\n", secretSubkeyPath)
		return "", ansi.Errorf("@R{secret} @c{%s} @R{is not a string}", secretSubkeyPath)
	}
	DEBUG(" success.")
	return v.(string), nil
}

func parsePath(path string) (secret, key string) {
	secret = path
	if idx := strings.LastIndex(path, ":"); idx >= 0 {
		secret = path[:idx]
		key = path[idx+1:]
	}
	return
}

func skipVaultVerify(env string) bool {
	env = strings.ToLower(env)
	if env == "" || env == "no" || env == "false" || env == "0" || env == "off" {
		return false
	}
	return true
}
