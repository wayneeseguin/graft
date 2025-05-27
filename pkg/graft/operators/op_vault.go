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
)

var kv *vaultkv.KV = nil

var vaultSecretCache = map[string]map[string]interface{}{}

// VaultRefs maps secret path to paths in YAML structure which call for it
var VaultRefs = map[string][]string{}

// SkipVault toggles whether calls to the Vault operator actually cause the
// Vault to be contacted and the keys substituted in.
var SkipVault bool

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

	kv = client.NewKV()

	return nil
}

// Run executes the `(( vault ... ))` operator call, which entails
// interacting with the (unsealed) Vault instance to retrieve the
// given secrets.
func (o VaultOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( vault ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( vault ... )) operation at $.%s\n", ev.Here)

	// syntax: (( vault "secret/path:key" ))
	// syntax: (( vault path.object "to concat with" other.object ))
	// syntax: (( vault "secret/path:key" || "default" ))
	// syntax: (( vault prefix "/" key ":password" || "default" ))
	if len(args) < 1 {
		return nil, fmt.Errorf("vault operator requires at least one argument")
	}

	// Use the new argument processor
	processor := newVaultArgProcessor(args)

	// Build the vault path from all arguments
	key, err := processor.buildVaultPath(ev)
	if err != nil {
		// Failed to build path, check if we have a default
		if processor.hasDefault {
			DEBUG("vault: failed to build path (%s), evaluating default value", err)
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

	// Track vault references
	if refs, found := VaultRefs[key]; !found {
		VaultRefs[key] = []string{ev.Here.String()}
	} else {
		VaultRefs[key] = append(refs, ev.Here.String())
	}

	// Perform the vault lookup
	secret, err := o.performVaultLookup(key)
	if err != nil {
		// Check if we should try the default
		if processor.hasDefault && isVaultNotFound(err) {
			DEBUG("vault: secret not found, evaluating default value")
			defaultValue, evalErr := processor.evaluateDefault(ev)
			if evalErr != nil {
				return nil, fmt.Errorf("unable to evaluate default value: %s", evalErr)
			}
			return &Response{
				Type:  Replace,
				Value: defaultValue,
			}, nil
		}
		// No default or not a "not found" error
		return nil, err
	}

	// Success!
	return &Response{
		Type:  Replace,
		Value: secret,
	}, nil
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
func (VaultOperator) performVaultLookup(key string) (string, error) {
	if SkipVault {
		return "REDACTED", nil
	}

	if kv == nil {
		err := initializeVaultClient()
		if err != nil {
			return "", fmt.Errorf("Error during Vault client initialization: %s", err)
		}
	}

	leftPart, rightPart := parsePath(key)
	if leftPart == "" || rightPart == "" {
		return "", ansi.Errorf("@R{invalid argument} @c{%s}@R{; must be in the form} @m{path/to/secret:key}", key)
	}

	var fullSecret map[string]interface{}
	var found bool
	if fullSecret, found = vaultSecretCache[leftPart]; found {
		DEBUG("vault: Cache hit for `%s`", leftPart)
	} else {
		DEBUG("vault: Cache MISS for `%s`", leftPart)
		// Secret isn't cached. Grab it from the vault.
		var err error
		fullSecret, err = getVaultSecret(leftPart)
		if err != nil {
			//Normalize the error messages
			switch err.(type) {
			case *vaultkv.ErrNotFound:
				err = fmt.Errorf("secret %s not found", key)
			}
			return "", err
		}
		vaultSecretCache[leftPart] = fullSecret
	}

	secret, err := extractSubkey(fullSecret, leftPart, rightPart)
	if err != nil {
		return "", err
	}
	return secret, nil
}

func init() {
	RegisterOp("vault", VaultOperator{})
}

/****** VAULT INTEGRATION ***********************************/

func getVaultSecret(secret string) (map[string]interface{}, error) {
	ret := map[string]interface{}{}

	DEBUG("Fetching Vault secret at `%s'", secret)
	_, err := kv.Get(secret, &ret, nil)
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
