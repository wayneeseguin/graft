package operators

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/geofffranks/simpleyaml"
	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"
)

// LoadOperator is invoked with (( load <location> ))
type LoadOperator struct{}

// Setup ...
func (LoadOperator) Setup() error {
	return nil
}

// Phase ...
func (LoadOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (LoadOperator) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (LoadOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( load ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( load ... )) operation at $%s\n", ev.Here)

	if len(args) != 1 {
		return nil, fmt.Errorf("load operator requires exactly one literal string or reference argument")
	}

	// Use ResolveOperatorArgument to support nested expressions
	val, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		DEBUG("  arg[0]: failed to resolve expression to a concrete value")
		DEBUG("     [0]: error was: %s", err)
		return nil, err
	}

	if val == nil {
		DEBUG("  arg[0]: resolved to nil")
		return nil, fmt.Errorf("load operator argument cannot be nil")
	}

	var location string
	switch v := val.(type) {
	case string:
		DEBUG("  arg[0]: using string value '%v'", v)
		location = v

	case int, int64, float64, bool:
		DEBUG("  arg[0]: converting %T to string", v)
		location = fmt.Sprintf("%v", v)

	case map[interface{}]interface{}, map[string]interface{}:
		DEBUG("  arg[0]: %v is not a string scalar", v)
		return nil, ansi.Errorf("@R{load operator argument is a map; only string scalars are supported}")

	case []interface{}:
		DEBUG("  arg[0]: %v is not a string scalar", v)
		return nil, ansi.Errorf("@R{load operator argument is a list; only string scalars are supported}")

	default:
		DEBUG("  arg[0]: using value of type %T as string", val)
		location = fmt.Sprintf("%v", val)
	}

	bytes, err := getBytesFromLocation(location)
	if err != nil {
		return nil, err
	}

	data, err := simpleyaml.NewYaml(bytes)
	if err != nil {
		return nil, err
	}

	if listroot, err := data.Array(); err == nil {
		return &Response{
			Type:  Replace,
			Value: listroot,
		}, nil
	}

	if maproot, err := data.Map(); err == nil {
		return &Response{
			Type:  Replace,
			Value: maproot,
		}, nil
	}

	return nil, fmt.Errorf("unsupported root type in loaded content, only map or list roots are supported")
}

func getBytesFromLocation(location string) ([]byte, error) {
	// Handle location as a URI if it looks like one and has a scheme
	if locURL, err := url.ParseRequestURI(location); err == nil && locURL.Scheme != "" {
		response, err := http.Get(location) // #nosec G107 - load operator needs to fetch from user-specified URLs
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()

		data, err := io.ReadAll(response.Body)
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to retrieve data from location %s: %s", location, string(data))
		}

		return data, err
	}

	// Preprend the optional Graft base path override
	if !filepath.IsAbs(location) {
		location = filepath.Join(os.Getenv("GRAFT_FILE_BASE_PATH"), location)
	}

	// Handle location as local file if there is a file at that location
	if _, err := os.Stat(location); err == nil {
		// #nosec G304 - File path is from user-provided configuration/input which is expected behavior for the load operator
		return os.ReadFile(location)
	}

	// In any other case, bail out ...
	return nil, fmt.Errorf("unable to get any content using location %s: it is not a file or usable URI", location)
}

func init() {
	RegisterOp("load", LoadOperator{})
}
