package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"

	"github.com/cppforlife/go-patch/patch"
	"github.com/gonvenience/ytbx"
	"github.com/homeport/dyff/pkg/dyff"
	"github.com/mattn/go-isatty"
	"github.com/starkandwayne/goutils/ansi"

	"github.com/wayneeseguin/graft/log"
	"github.com/wayneeseguin/graft/pkg/graft"
	_ "github.com/wayneeseguin/graft/pkg/graft/operators" // Register operators
	"github.com/starkandwayne/goutils/tree"

	"strings"

	// Use geofffranks forks to persist the fix in https://github.com/go-yaml/yaml/pull/133/commits
	// Also https://github.com/go-yaml/yaml/pull/195
	"github.com/geofffranks/simpleyaml"
	"github.com/geofffranks/yaml"
	"github.com/voxelbrain/goptions"
)

// Version holds the Current version of graft
var Version = "(development)"

var printfStdOut = func(format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, format, args...)
}

var getopts = func(o interface{}) {
	err := goptions.Parse(o)
	if err != nil {
		usage()
	}
}

var exit = func(code int) {
	os.Exit(code)
}

var usage = func() {
	goptions.PrintHelp()
	exit(1)
}

func envFlag(varname string) bool {
	val := os.Getenv(varname)
	return val != "" && strings.ToLower(val) != "false" && val != "0"
}

type YamlFile struct {
	Path   string
	Reader io.ReadCloser
}

type jsonOpts struct {
	Strict bool               `goptions:"--strict, description='Refuse to convert non-string keys to strings'"`
	Help   bool               `goptions:"--help, -h"`
	Files  goptions.Remainder `goptions:"description='Files to convert to JSON'"`
}

type mergeOpts struct {
	SkipEval       bool               `goptions:"--skip-eval, description='Do not evaluate graft logic after merging docs'"`
	Prune          []string           `goptions:"--prune, description='Specify keys to prune from final output (may be specified more than once)'"`
	CherryPick     []string           `goptions:"--cherry-pick, description='The opposite of prune, specify keys to cherry-pick from final output (may be specified more than once)'"`
	FallbackAppend bool               `goptions:"--fallback-append, description='Default merge normally tries to key merge, then inline. This flag says do an append instead of an inline.'"`
	EnableGoPatch  bool               `goptions:"--go-patch, description='Enable the use of go-patch when parsing files to be merged'"`
	MultiDoc       bool               `goptions:"--multi-doc, -m, description='Treat multi-doc yaml as multiple files.'"`
	DataflowOrder  string             `goptions:"--dataflow-order, description='Order of operations in dataflow output: alphabetical (default) or insertion'"`
	Help           bool               `goptions:"--help, -h"`
	Files          goptions.Remainder `goptions:"description='List of files to merge. To read STDIN, specify a filename of \\'-\\'.'"`
}

// checkForCycles detects circular references in the data structure
func checkForCycles(root interface{}, maxDepth int) error {
	visited := make(map[uintptr]bool)
	
	var check func(o interface{}, depth int) error
	check = func(o interface{}, depth int) error {
		if depth == 0 {
			return ansi.Errorf("@*{Hit max recursion depth. You seem to have a self-referencing dataset}")
		}

		switch v := o.(type) {
		case map[interface{}]interface{}:
			// Check if we've seen this map before (circular reference)
			ptr := reflect.ValueOf(v).Pointer()
			if visited[ptr] {
				return ansi.Errorf("@*{Hit max recursion depth. You seem to have a self-referencing dataset}")
			}
			visited[ptr] = true
			
			for _, val := range v {
				if err := check(val, depth-1); err != nil {
					return err
				}
			}
			
			delete(visited, ptr) // Remove after visiting children
			
		case []interface{}:
			// Check if we've seen this slice before (circular reference)
			ptr := reflect.ValueOf(v).Pointer()
			if visited[ptr] {
				return ansi.Errorf("@*{Hit max recursion depth. You seem to have a self-referencing dataset}")
			}
			visited[ptr] = true
			
			for _, val := range v {
				if err := check(val, depth-1); err != nil {
					return err
				}
			}
			
			delete(visited, ptr) // Remove after visiting children
		}

		return nil
	}

	return check(root, maxDepth)
}

func main() {
	var options struct {
		Debug   bool   `goptions:"-D, --debug, description='Enable debugging'"`
		Trace   bool   `goptions:"-T, --trace, description='Enable trace mode debugging (very verbose)'"`
		Version bool   `goptions:"-v, --version, description='Display version information'"`
		Color   string `goptions:"--color, description='Control color output (on/off/auto, default: auto)'"`
		Action  goptions.Verbs
		Merge   mergeOpts `goptions:"merge"`
		Fan     mergeOpts `goptions:"fan"`
		JSON    jsonOpts  `goptions:"json"`
		Diff    struct {
			Files goptions.Remainder `goptions:"description='Show the semantic differences between two YAML files'"`
		} `goptions:"diff"`
		VaultInfo struct {
			EnableGoPatch bool               `goptions:"--go-patch, description='Enable the use of go-patch when parsing files to be merged'"`
			Files         goptions.Remainder `goptions:"description='List vault references in the given files'"`
		} `goptions:"vaultinfo"`
	}
	getopts(&options)

	if envFlag("DEBUG") || options.Debug {
		log.DebugOn = true
	}

	if envFlag("TRACE") || options.Trace {
		log.TraceOn = true
		log.DebugOn = true
	}

	if options.JSON.Help || options.Merge.Help || options.Fan.Help {
		usage()
		return
	}

	if options.Version {
		printfStdOut("%s - Version %s\n", os.Args[0], Version)
		exit(0)
		return
	}

	// Handle color flag
	shouldEnableColor := false
	switch options.Color {
	case "on":
		shouldEnableColor = true
	case "off":
		shouldEnableColor = false
	case "auto", "":
		// Auto-detect based on whether stderr is a terminal
		shouldEnableColor = isatty.IsTerminal(os.Stderr.Fd())
	default:
		log.PrintfStdErr("Invalid --color option: %s. Must be 'on', 'off', or 'auto'.\n", options.Color)
		exit(1)
		return
	}
	ansi.Color(shouldEnableColor)

	switch options.Action {
	case "merge":
		tree, err := cmdMergeEval(options.Merge)
		if err != nil {
			log.PrintfStdErr("%s\n", err.Error())
			exit(2)
			return
		}

		log.TRACE("Converting the following data back to YML:")
		log.TRACE("%#v", tree)
		
		// Check for cycles before attempting to marshal
		if err := checkForCycles(tree, 4096); err != nil {
			log.PrintfStdErr("%s\n", err.Error())
			exit(2)
			return
		}
		
		merged, err := yaml.Marshal(tree)
		if err != nil {
			log.PrintfStdErr("Unable to convert merged result back to YAML: %s\nData:\n%#v", err.Error(), tree)
			exit(2)
			return
		}

		printfStdOut("%s\n", string(merged))

	case "fan":
		trees, err := cmdFanEval(options.Fan)
		if err != nil {
			log.PrintfStdErr("%s\n", err.Error())
			exit(2)
			return
		}

		for _, tree := range trees {
			log.TRACE("Converting the following data back to YML:")
			log.TRACE("%#v", tree)
			
			// Check for cycles before attempting to marshal
			if err := checkForCycles(tree, 4096); err != nil {
				log.PrintfStdErr("%s\n", err.Error())
				exit(2)
				return
			}
			
			merged, err := yaml.Marshal(tree)
			if err != nil {
				log.PrintfStdErr("Unable to convert merged result back to YAML: %s\nData:\n%#v", err.Error(), tree)
				exit(2)
				return
			}

			printfStdOut("---\n%s\n", string(merged))
		}

	case "vaultinfo":
		graft.VaultRefs = map[string][]string{}
		graft.SkipVault = true
		options.Merge.Files = options.VaultInfo.Files
		options.Merge.EnableGoPatch = options.VaultInfo.EnableGoPatch
		_, err := cmdMergeEval(options.Merge)
		if err != nil {
			log.PrintfStdErr("%s\n", err.Error())
			exit(2)
			return
		}

		printfStdOut("%s\n", formatVaultRefs())
	case "json":
		jsons, err := cmdJSONEval(options.JSON)
		if err != nil {
			log.PrintfStdErr("%s\n", err)
			exit(2)
			return
		}
		for _, output := range jsons {
			printfStdOut("%s\n", output)
		}

	case "diff":
		// For diff, check stdout instead of stderr when auto-detecting
		if options.Color == "auto" || options.Color == "" {
			ansi.Color(isatty.IsTerminal(os.Stdout.Fd()))
		}
		// Otherwise use the already set color preference from above
		if len(options.Diff.Files) != 2 {
			usage()
			return
		}
		output, differences, err := diffFiles(options.Diff.Files)
		if err != nil {
			log.PrintfStdErr("%s\n", err)
			exit(2)
			return
		}
		printfStdOut("%s\n", output)
		if differences {
			exit(1)
		}

	default:
		usage()
		return
	}
	exit(0)
}

func isArrayError(err error) bool {
	_, ok := err.(RootIsArrayError)
	return ok
}

func parseGoPatch(data []byte) (patch.Ops, error) {
	opdefs := []patch.OpDefinition{}
	err := yaml.Unmarshal(data, &opdefs)
	if err != nil {
		return nil, ansi.Errorf("@R{Root of YAML document is not a hash/map. Tried parsing it as go-patch, but got}: %s\n", err)
	}
	ops, err := patch.NewOpsFromDefinitions(opdefs)
	if err != nil {
		return nil, ansi.Errorf("@R{Unable to parse go-patch definitions: %s\n", err)
	}
	return ops, nil
}

func parseYAML(data []byte) (map[interface{}]interface{}, error) {
	y, err := simpleyaml.NewYaml(data)
	if err != nil {
		return nil, err
	}

	if empty_y, _ := simpleyaml.NewYaml([]byte{}); *y == *empty_y {
		log.DEBUG("YAML doc is empty, creating empty hash/map")
		return make(map[interface{}]interface{}), nil
	}

	doc, err := y.Map()

	if err != nil {
		if _, arrayErr := y.Array(); arrayErr == nil {
			return nil, RootIsArrayError{msg: ansi.Sprintf("@R{Root of YAML document is not a hash/map}: %s\n", err)}
		}
		return nil, ansi.Errorf("@R{Root of YAML document is not a hash/map}: %s\n", err.Error())
	}

	return doc, nil
}

func loadYamlFile(file string) (YamlFile, error) {
	var target YamlFile
	if file == "-" {
		target = YamlFile{Reader: os.Stdin, Path: "-"}
	} else {
		f, err := os.Open(file)
		if err != nil {
			return YamlFile{}, ansi.Errorf("@R{Error reading file} @m{%s}: %s", file, err.Error())
		}
		target = YamlFile{Path: file, Reader: f}
	}
	return target, nil
}

func splitLoadYamlFile(file string) ([]YamlFile, error) {
	docs := []YamlFile{}

	yamlFile, err := loadYamlFile(file)
	if err != nil {
		return nil, err
	}

	fileData, err := readFile(&yamlFile)
	if err != nil {
		return nil, err
	}

	rawDocs := bytes.Split(fileData, []byte("\n---\n"))
	// strip off empty document created if the first three bytes of the file are the doc separator
	// keeps the indexing correct for when used with error messages
	if len(rawDocs[0]) == 0 {
		rawDocs = rawDocs[1:]
	}

	for i, docBytes := range rawDocs {
		buf := bytes.NewBuffer(docBytes)
		doc := YamlFile{Path: fmt.Sprintf("%s[%d]", yamlFile.Path, i), Reader: io.NopCloser(buf)}
		docs = append(docs, doc)
	}
	return docs, nil
}

func cmdMergeEval(options mergeOpts) (map[interface{}]interface{}, error) {
	files := []YamlFile{}

	if len(options.Files) < 1 {
		stdinInfo, err := os.Stdin.Stat()
		if err != nil {
			return nil, ansi.Errorf("@R{Error statting STDIN} - Bailing out: %s\n", err.Error())
		}

		if stdinInfo.Mode()&os.ModeCharDevice != 0 {
			return nil, ansi.Errorf("@R{Error reading STDIN}: no data found. Did you forget to pipe data to STDIN, or specify yaml files to merge?")
		}

		options.Files = append(options.Files, "-")
	}

	for _, file := range options.Files {
		if options.MultiDoc {
			docs, err := splitLoadYamlFile(file)
			if err != nil {
				return nil, err
			}
			files = append(files, docs...)
		} else {
			yamlFile, err := loadYamlFile(file)
			if err != nil {
				return nil, err
			}
			files = append(files, yamlFile)
		}
	}

	result, err := mergeAllDocs(files, options)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func cmdFanEval(options mergeOpts) ([]map[interface{}]interface{}, error) {
	stdinInfo, err := os.Stdin.Stat()
	if err != nil {
		return nil, ansi.Errorf("@R{Error statting STDIN} - Bailing out: %s\n", err.Error())
	}
	if stdinInfo.Mode()&os.ModeCharDevice == 0 {
		options.Files = append(options.Files, "-")
	}

	if len(options.Files) == 0 {
		return nil, ansi.Errorf("@R{Missing Input:} You must specify at least a source document to graft fan. If no files are specified, STDIN is used. Using STDIN for source and target docs only works with -m.")
	}

	roots := []map[interface{}]interface{}{}
	sourcePath := options.Files[0]
	options.Files = options.Files[1:]

	docs := []YamlFile{}
	source := YamlFile{}
	if options.MultiDoc {
		sourceDocs, err := splitLoadYamlFile(sourcePath)
		if err != nil {
			return nil, err
		}
		// only the first yaml document of the source will be treated as actual source, all others
		// will be treated as target documents
		source = sourceDocs[0]
		docs = append(sourceDocs[1:], docs...)
	} else {
		source, err = loadYamlFile(sourcePath)
		if err != nil {
			return nil, err
		}
	}

	for _, file := range options.Files {
		yamlDocs, err := splitLoadYamlFile(file)
		if err != nil {
			return nil, err
		}
		docs = append(docs, yamlDocs...)
	}

	sourceBytes, err := readFile(&source)
	if err != nil {
		return nil, err
	}

	if len(docs) < 1 {
		return nil, ansi.Errorf("@R{Missing Input:} You must specify at least one target document to graft fan. If no files are specified, STDIN is used. Using STDIN for source and target docs only works with -m.")
	}

	for _, doc := range docs {
		sourceBuffer := bytes.NewBuffer(sourceBytes)
		source = YamlFile{Path: source.Path, Reader: io.NopCloser(sourceBuffer)}
		result, err := mergeAllDocs([]YamlFile{source, doc}, options)
		if err != nil {
			return nil, err
		}
		roots = append(roots, result)
	}

	return roots, nil
}

func cmdJSONEval(options jsonOpts) ([]string, error) {
	stdinInfo, err := os.Stdin.Stat()
	if err != nil {
		return nil, ansi.Errorf("@R{Error statting STDIN} - Bailing out: %s\n", err.Error())
	}
	if stdinInfo.Mode()&os.ModeCharDevice == 0 {
		options.Files = append(options.Files, "-")
	}

	output, err := graft.JSONifyFiles(options.Files, options.Strict)
	if err != nil {
		return nil, err
	}

	return output, nil
}

type yamlVaultSecret struct {
	Key        string
	References []string
}

type byKey []yamlVaultSecret

type yamlVaultRefs struct {
	Secrets []yamlVaultSecret
}

func (refs byKey) Len() int           { return len(refs) }
func (refs byKey) Swap(i, j int)      { refs[i], refs[j] = refs[j], refs[i] }
func (refs byKey) Less(i, j int) bool { return refs[i].Key < refs[j].Key }

func formatVaultRefs() string {
	refs := yamlVaultRefs{}
	for secret, srcs := range graft.VaultRefs {
		refs.Secrets = append(refs.Secrets, yamlVaultSecret{secret, srcs})
	}

	sort.Sort(byKey(refs.Secrets))
	for _, secret := range refs.Secrets {
		sort.Strings(secret.References)
	}

	output, err := yaml.Marshal(refs)
	if err != nil {
		panic(fmt.Sprintf("Could not marshal YAML for vault references: %+v", graft.VaultRefs))
	}

	return string(output)
}

func readFile(file *YamlFile) ([]byte, error) {
	var data []byte
	var err error

	if file.Path == "-" {
		file.Path = "STDIN"
		stat, err := os.Stdin.Stat()
		if err != nil {
			return nil, ansi.Errorf("@R{Error statting STDIN} - Bailing out: %s\n", err.Error())
		}
		if stat.Mode()&os.ModeCharDevice == 0 {
			data, err = io.ReadAll(os.Stdin)
			if err != nil {
				return nil, ansi.Errorf("@R{Error reading file} @m{%s}: %s\n", file.Path, err.Error())
			}
		}
	} else {
		data, err = io.ReadAll(file.Reader)
		if err != nil {
			return nil, ansi.Errorf("@R{Error reading file} @m{%s}: %s\n", file.Path, err.Error())
		}
	}
	if len(data) == 0 && file.Path == "STDIN" {
		return nil, ansi.Errorf("@R{Error reading STDIN}: no data found. Did you forget to pipe data to STDIN, or specify yaml files to merge?")
	}

	return data, nil
}

// validatePath checks if a path exists in the data structure
func validatePath(data map[interface{}]interface{}, path string) error {
	// Use graft's tree package to parse and resolve the path
	cursor, err := tree.ParseCursor(path)
	if err != nil {
		return err
	}
	
	_, err = cursor.Resolve(data)
	if err != nil {
		// Check if the error is due to a missing parent path
		// Walk through the path segments to find where it fails
		parts := strings.Split(path, ".")
		failedPath := ""
		
		for i, part := range parts {
			if i == 0 {
				failedPath = part
			} else {
				failedPath = strings.Join(parts[:i+1], ".")
			}
			
			// Try to resolve up to this point
			partialCursor, _ := tree.ParseCursor(failedPath)
			if _, resolveErr := partialCursor.Resolve(data); resolveErr != nil {
				// This is where it failed, report this path
				return ansi.Errorf("1 error(s) detected:\n - `$.%s` could not be found in the datastructure\n\n", failedPath)
			}
		}
		
		// If we couldn't find where it failed, report the full path
		return ansi.Errorf("1 error(s) detected:\n - `$.%s` could not be found in the datastructure\n\n", path)
	}
	
	return nil
}

func mergeAllDocs(files []YamlFile, options mergeOpts) (map[interface{}]interface{}, error) {
	// Create engine with settings from options
	engineOpts := []graft.EngineOption{
		graft.WithCache(true, 1000),
		graft.WithConcurrency(10),
		graft.WithEnhancedParser(true),
	}
	
	// Set dataflow order if specified (default to alphabetical if not set)
	dataflowOrder := options.DataflowOrder
	if dataflowOrder == "" {
		dataflowOrder = "alphabetical"
	}
	engineOpts = append(engineOpts, graft.WithDataflowOrder(dataflowOrder))
	
	engine, err := graft.NewEngine(engineOpts...)
	if err != nil {
		return nil, ansi.Errorf("@R{Failed to create graft engine}: %s", err.Error())
	}

	// Parse all documents
	docs := []graft.Document{}
	for _, file := range files {
		log.DEBUG("Processing file '%s'", file.Path)

		data, err := readFile(&file)
		if err != nil {
			return nil, err
		}

		// Check if it's a go-patch document
		if options.EnableGoPatch {
			_, parseErr := parseYAML(data)
			if isArrayError(parseErr) {
				log.DEBUG("Detected root of document as an array. Attempting go-patch parsing")
				_, err := parseGoPatch(data)
				if err != nil {
					return nil, ansi.Errorf("@m{%s}: @R{%s}\n", file.Path, err.Error())
				}
				// For go-patch, we need to apply it after merging other docs
				// Store it for later application
				// TODO: Properly integrate go-patch with new API
				return nil, ansi.Errorf("@R{go-patch support needs to be reimplemented with new API}")
			}
		}

		// Parse as YAML
		doc, err := engine.ParseYAML(data)
		if err != nil {
			return nil, ansi.Errorf("@m{%s}: @R{%s}\n", file.Path, err.Error())
		}
		docs = append(docs, doc)
	}

	// Merge all documents
	mergeBuilder := engine.Merge(nil, docs...)
	
	// Apply merge options
	if options.FallbackAppend {
		mergeBuilder = mergeBuilder.WithArrayMergeStrategy(graft.AppendArrays)
	}
	
	if options.SkipEval {
		mergeBuilder = mergeBuilder.SkipEvaluation()
	}
	
	// Execute merge
	merged, err := mergeBuilder.Execute()
	if err != nil {
		// Check if this is a MultiError from the merger (Issue #172)
		if strings.Contains(err.Error(), "error(s) detected:") {
			return nil, err
		}
		return nil, ansi.Errorf("@R{Merge failed}: %s", err.Error())
	}

	// Evaluate operators unless skipped
	var result graft.Document
	if !options.SkipEval {
		result, err = engine.Evaluate(nil, merged)
		if err != nil {
			return nil, ansi.Errorf("@R{Evaluation failed}: %s", err.Error())
		}
	} else {
		result = merged
	}

	// Apply pruning and cherry-picking
	if len(options.Prune) > 0 {
		for _, key := range options.Prune {
			result = result.Prune(key)
		}
	}
	
	if len(options.CherryPick) > 0 {
		// Validate cherry-pick paths exist before extraction
		data := result.GetData().(map[interface{}]interface{})
		for _, path := range options.CherryPick {
			if err := validatePath(data, path); err != nil {
				return nil, err
			}
		}
		result = result.CherryPick(options.CherryPick...)
	}

	// Get the raw data for backward compatibility
	// The CLI expects a map[interface{}]interface{}
	return result.GetData().(map[interface{}]interface{}), nil
}

func diffFiles(paths []string) (string, bool, error) {
	if len(paths) != 2 {
		return "", false, ansi.Errorf("incorrect number of files given to diffFiles(); please file a bug report")
	}

	from, to, err := ytbx.LoadFiles(paths[0], paths[1])
	if err != nil {
		return "", false, err
	}

	report, err := dyff.CompareInputFiles(from, to)
	if err != nil {
		return "", false, err
	}

	reportWriter := &dyff.HumanReport{
		Report:            report,
		DoNotInspectCerts: false,
		NoTableStyle:      false,
		OmitHeader:        true,
	}

	var buf bytes.Buffer
	out := bufio.NewWriter(&buf)
	reportWriter.WriteReport(out)
	out.Flush()

	return buf.String(), len(report.Diffs) > 0, nil
}

type RootIsArrayError struct {
	msg string
}

func (r RootIsArrayError) Error() string {
	return r.msg
}
