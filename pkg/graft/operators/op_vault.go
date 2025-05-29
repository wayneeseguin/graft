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

// SkipVault toggles whether calls to the Vault operator actually cause the
// Vault to be contacted and the keys substituted in.
var SkipVault bool

// vaultArgProcessor handles argument processing for vault operator with LogicalOr support
type vaultArgProcessor struct {
	args         []*Expr
	hasDefault   bool
	defaultExpr  *Expr
	defaultIndex int
}

// newVaultArgProcessor creates a processor that extracts defaults from any position
func newVaultArgProcessor(args []*Expr) *vaultArgProcessor {
	processor := &vaultArgProcessor{
		args:         make([]*Expr, len(args)),
		hasDefault:   false,
		defaultIndex: -1,
	}

	// Copy args and extract any LogicalOr
	for i, arg := range args {
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
	switch v := value.(type) {
	case string:
		return v, nil
	case int, int64, float32, float64, bool:
		return fmt.Sprintf("%v", v), nil
	case map[interface{}]interface{}, map[string]interface{}:
		if expr.Type == Reference {
			return "", fmt.Errorf("$.%s is a map; only scalars are supported for vault paths", expr.Reference)
		}
		return "", fmt.Errorf("value is a map; only scalars are supported for vault paths")
	case []interface{}:
		if expr.Type == Reference {
			return "", fmt.Errorf("$.%s is a list; only scalars are supported for vault paths", expr.Reference)
		}
		return "", fmt.Errorf("value is a list; only scalars are supported for vault paths")
	default:
		return fmt.Sprintf("%v", v), nil
	}
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
					InsecureSkipVerify: skip,
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

	// syntax: (( vault "secret/path:key" ))
	// syntax: (( vault path.object "to concat with" other.object ))
	// syntax: (( vault "secret/path:key" || "default" ))
	// syntax: (( vault prefix "/" key ":password" || "default" ))
	if len(args) < 1 {
		return nil, fmt.Errorf("vault operator requires at least one argument")
	}

	// Use the new argument processor
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

	// Try each path in order
	var lastErr error
	for i, key := range paths {
		DEBUG("vault: trying path %d of %d: %s", i+1, len(paths), key)
		
		// Track vault references using engine context
		engine.GetOperatorState().AddVaultRef(key, []string{ev.Here.String()})

		// Perform the vault lookup
		secret, err := o.performVaultLookup(engine, key)
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
func (VaultOperator) performVaultLookup(engine graft.Engine, key string) (string, error) {
	if engine.GetOperatorState().IsVaultSkipped() {
		return "REDACTED", nil
	}

	kv := engine.GetOperatorState().GetVaultClient()
	if kv == nil {
		// For backward compatibility, try to initialize from environment
		if SkipVault {
			return "REDACTED", nil
		}
		
		// Fall back to global initialization for now
		if globalKV == nil {
			err := initializeVaultClient()
			if err != nil {
				return "", fmt.Errorf("Error during Vault client initialization: %s", err)
			}
		}
		kv = globalKV
	}

	leftPart, rightPart := parsePath(key)
	if leftPart == "" || rightPart == "" {
		return "", ansi.Errorf("@R{invalid argument} @c{%s}@R{; must be in the form} @m{path/to/secret:key}", key)
	}

	// Check cache first
	vaultCache := engine.GetOperatorState().GetVaultCache()
	var fullSecret map[string]interface{}
	var found bool
	if fullSecret, found = vaultCache[leftPart]; found {
		DEBUG("vault: Cache hit for `%s`", leftPart)
	} else {
		DEBUG("vault: Cache MISS for `%s`", leftPart)
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
		engine.GetOperatorState().SetVaultCache(leftPart, fullSecret)
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
		secret, err := vaultOp.performVaultLookup(engine, path)
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
