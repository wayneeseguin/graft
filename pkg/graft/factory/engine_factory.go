package factory

import (
	"os"

	"github.com/wayneeseguin/graft/pkg/graft"
	"github.com/wayneeseguin/graft/pkg/graft/operators"
)

// NewDefaultEngine creates an engine with all default operators registered
func NewDefaultEngine() *graft.DefaultEngine {
	config := graft.DefaultEngineConfig()

	// Configure from environment
	config.VaultAddr = os.Getenv("VAULT_ADDR")
	config.VaultToken = os.Getenv("VAULT_TOKEN")
	config.SkipVault = os.Getenv("REDACT") != ""

	config.AWSRegion = os.Getenv("AWS_REGION")
	if config.AWSRegion == "" {
		config.AWSRegion = os.Getenv("AWS_DEFAULT_REGION")
	}

	engine := graft.NewDefaultEngineWithConfig(config)

	// Register all default operators
	registerDefaultOperators(engine)

	return engine
}

// registerDefaultOperators registers all built-in operators with the engine
func registerDefaultOperators(engine *graft.DefaultEngine) {
	// Type checking operators
	engine.RegisterOperator("empty", &operators.EmptyOperator{})
	engine.RegisterOperator("null", &operators.NullOperator{})

	// Reference operators
	engine.RegisterOperator("grab", &operators.GrabOperator{})
	engine.RegisterOperator("param", &operators.ParamOperator{})

	// String manipulation
	engine.RegisterOperator("concat", &operators.ConcatOperator{})
	engine.RegisterOperator("join", &operators.JoinOperator{})
	engine.RegisterOperator("stringify", &operators.StringifyOperator{})
	engine.RegisterOperator("base64", &operators.Base64Operator{})
	engine.RegisterOperator("base64-decode", &operators.Base64DecodeOperator{})

	// Data manipulation
	engine.RegisterOperator("keys", &operators.KeysOperator{})
	engine.RegisterOperator("sort", &operators.SortOperator{})
	engine.RegisterOperator("shuffle", &operators.ShuffleOperator{})
	engine.RegisterOperator("prune", &operators.PruneOperator{})

	// Math operators
	engine.RegisterOperator("calc", &operators.CalcOperator{})

	// Control flow
	engine.RegisterOperator("ternary", &operators.TernaryOperator{})
	engine.RegisterOperator("negate", &operators.NegateOperator{})
	engine.RegisterOperator("defer", &operators.DeferOperator{})

	// External data sources
	engine.RegisterOperator("vault", &operators.VaultOperator{})
	engine.RegisterOperator("vault-try", &operators.VaultTryOperator{})
	engine.RegisterOperator("file", &operators.FileOperator{})
	engine.RegisterOperator("awsparam", operators.NewAwsParamOperator())
	engine.RegisterOperator("awssecret", operators.NewAwsSecretOperator())
	engine.RegisterOperator("load", &operators.LoadOperator{})

	// Network operations
	engine.RegisterOperator("static_ips", &operators.StaticIPOperator{})
	engine.RegisterOperator("ips", &operators.IpsOperator{})

	// Advanced operations
	engine.RegisterOperator("inject", &operators.InjectOperator{})
	engine.RegisterOperator("cartesian-product", &operators.CartesianProductOperator{})
}

// NewMinimalEngine creates an engine with only essential operators
func NewMinimalEngine() *graft.DefaultEngine {
	engine := graft.NewDefaultEngine()

	// Register only essential operators
	engine.RegisterOperator("grab", &operators.GrabOperator{})
	engine.RegisterOperator("concat", &operators.ConcatOperator{})
	engine.RegisterOperator("empty", &operators.EmptyOperator{})

	return engine
}

// NewTestEngine creates an engine suitable for testing
func NewTestEngine() *graft.DefaultEngine {
	config := graft.DefaultEngineConfig()
	config.SkipVault = true
	config.SkipAWS = true
	config.EnableCaching = false

	engine := graft.NewDefaultEngineWithConfig(config)
	registerDefaultOperators(engine)

	return engine
}

