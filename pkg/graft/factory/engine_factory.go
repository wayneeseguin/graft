package factory

import (
	"fmt"
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
	if err := registerDefaultOperators(engine); err != nil {
		panic(fmt.Sprintf("failed to register default operators: %v", err))
	}

	return engine
}

// registerDefaultOperators registers all built-in operators with the engine
func registerDefaultOperators(engine *graft.DefaultEngine) error {
	// Type checking operators
	if err := engine.RegisterOperator("empty", &operators.EmptyOperator{}); err != nil {
		return err
	}
	if err := engine.RegisterOperator("null", &operators.NullOperator{}); err != nil {
		return err
	}

	// Reference operators
	if err := engine.RegisterOperator("grab", &operators.GrabOperator{}); err != nil {
		return err
	}
	if err := engine.RegisterOperator("param", &operators.ParamOperator{}); err != nil {
		return err
	}

	// String manipulation
	if err := engine.RegisterOperator("concat", &operators.ConcatOperator{}); err != nil {
		return err
	}
	if err := engine.RegisterOperator("join", &operators.JoinOperator{}); err != nil {
		return err
	}
	if err := engine.RegisterOperator("stringify", &operators.StringifyOperator{}); err != nil {
		return err
	}
	if err := engine.RegisterOperator("base64", &operators.Base64Operator{}); err != nil {
		return err
	}
	if err := engine.RegisterOperator("base64-decode", &operators.Base64DecodeOperator{}); err != nil {
		return err
	}

	// Data manipulation
	if err := engine.RegisterOperator("keys", &operators.KeysOperator{}); err != nil {
		return err
	}
	if err := engine.RegisterOperator("sort", &operators.SortOperator{}); err != nil {
		return err
	}
	if err := engine.RegisterOperator("shuffle", &operators.ShuffleOperator{}); err != nil {
		return err
	}
	if err := engine.RegisterOperator("prune", &operators.PruneOperator{}); err != nil {
		return err
	}

	// Math operators
	if err := engine.RegisterOperator("calc", &operators.CalcOperator{}); err != nil {
		return err
	}

	// Control flow
	if err := engine.RegisterOperator("ternary", &operators.TernaryOperator{}); err != nil {
		return err
	}
	if err := engine.RegisterOperator("negate", &operators.NegateOperator{}); err != nil {
		return err
	}
	if err := engine.RegisterOperator("defer", &operators.DeferOperator{}); err != nil {
		return err
	}

	// External data sources
	if err := engine.RegisterOperator("vault", &operators.VaultOperator{}); err != nil {
		return err
	}
	if err := engine.RegisterOperator("vault-try", &operators.VaultTryOperator{}); err != nil {
		return err
	}
	if err := engine.RegisterOperator("file", &operators.FileOperator{}); err != nil {
		return err
	}
	if err := engine.RegisterOperator("awsparam", operators.NewAwsParamOperator()); err != nil {
		return err
	}
	if err := engine.RegisterOperator("awssecret", operators.NewAwsSecretOperator()); err != nil {
		return err
	}
	if err := engine.RegisterOperator("load", &operators.LoadOperator{}); err != nil {
		return err
	}

	// Network operations
	if err := engine.RegisterOperator("static_ips", &operators.StaticIPOperator{}); err != nil {
		return err
	}
	if err := engine.RegisterOperator("ips", &operators.IpsOperator{}); err != nil {
		return err
	}

	// Advanced operations
	if err := engine.RegisterOperator("inject", &operators.InjectOperator{}); err != nil {
		return err
	}
	if err := engine.RegisterOperator("cartesian-product", &operators.CartesianProductOperator{}); err != nil {
		return err
	}

	// Boolean operators
	if err := engine.RegisterOperator("&&", operators.NewTypeAwareAndOperator()); err != nil {
		return err
	}
	if err := engine.RegisterOperator("||", operators.NewTypeAwareOrOperator()); err != nil {
		return err
	}
	if err := engine.RegisterOperator("!", operators.NewTypeAwareNotOperator()); err != nil {
		return err
	}

	// Comparison operators
	if err := engine.RegisterOperator("==", operators.NewTypeAwareEqualOperator()); err != nil {
		return err
	}
	if err := engine.RegisterOperator("!=", operators.NewTypeAwareNotEqualOperator()); err != nil {
		return err
	}
	if err := engine.RegisterOperator("<", operators.NewTypeAwareLessOperator()); err != nil {
		return err
	}
	if err := engine.RegisterOperator(">", operators.NewTypeAwareGreaterOperator()); err != nil {
		return err
	}
	if err := engine.RegisterOperator("<=", operators.NewTypeAwareLessOrEqualOperator()); err != nil {
		return err
	}
	if err := engine.RegisterOperator(">=", operators.NewTypeAwareGreaterOrEqualOperator()); err != nil {
		return err
	}

	return nil
}

// NewMinimalEngine creates an engine with only essential operators
func NewMinimalEngine() *graft.DefaultEngine {
	engine := graft.NewDefaultEngine()

	// Register only essential operators
	if err := engine.RegisterOperator("grab", &operators.GrabOperator{}); err != nil {
		panic(fmt.Sprintf("failed to register grab operator: %v", err))
	}
	if err := engine.RegisterOperator("concat", &operators.ConcatOperator{}); err != nil {
		panic(fmt.Sprintf("failed to register concat operator: %v", err))
	}
	if err := engine.RegisterOperator("empty", &operators.EmptyOperator{}); err != nil {
		panic(fmt.Sprintf("failed to register empty operator: %v", err))
	}

	return engine
}

// NewTestEngine creates an engine suitable for testing
func NewTestEngine() *graft.DefaultEngine {
	config := graft.DefaultEngineConfig()
	config.SkipVault = true
	config.SkipAWS = true
	config.EnableCaching = false

	engine := graft.NewDefaultEngineWithConfig(config)
	if err := registerDefaultOperators(engine); err != nil {
		panic(fmt.Sprintf("failed to register default operators for test engine: %v", err))
	}

	return engine
}
