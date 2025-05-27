package graft

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/tabwriter"
)

// TuningCLI provides command-line interface for performance tuning
type TuningCLI struct {
	config       *PerformanceConfig
	configLoader *ConfigLoader
	profileMgr   *ProfileManager
}

// NewTuningCLI creates a new tuning CLI handler
func NewTuningCLI(configPath string) (*TuningCLI, error) {
	loader := NewConfigLoader(configPath)
	config, err := loader.Load()
	if err != nil {
		return nil, err
	}

	profileMgr, err := NewProfileManager()
	if err != nil {
		return nil, err
	}

	return &TuningCLI{
		config:       config,
		configLoader: loader,
		profileMgr:   profileMgr,
	}, nil
}

// ExecuteCommand executes a tuning command
func (cli *TuningCLI) ExecuteCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	command := args[0]
	cmdArgs := args[1:]

	switch command {
	case "show":
		return cli.showConfig(cmdArgs)
	case "get":
		return cli.getConfig(cmdArgs)
	case "set":
		return cli.setConfig(cmdArgs)
	case "profile":
		return cli.applyProfile(cmdArgs)
	case "profiles":
		return cli.listProfiles()
	case "export":
		return cli.exportConfig(cmdArgs)
	case "import":
		return cli.importConfig(cmdArgs)
	case "validate":
		return cli.validateConfig(cmdArgs)
	case "reset":
		return cli.resetConfig()
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// showConfig displays the current configuration
func (cli *TuningCLI) showConfig(args []string) error {
	format := "yaml"
	if len(args) > 0 && args[0] == "--json" {
		format = "json"
	}

	if format == "json" {
		data, err := json.MarshalIndent(cli.config, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	} else {
		yamlStr, err := ConfigToYAML(cli.config)
		if err != nil {
			return err
		}
		fmt.Print(yamlStr)
	}

	return nil
}

// getConfig gets a specific configuration value
func (cli *TuningCLI) getConfig(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: config get <path>")
	}

	path := args[0]
	value, err := GetFieldValue(cli.config, path)
	if err != nil {
		return err
	}

	fmt.Printf("%s = %v\n", path, value)
	return nil
}

// setConfig sets a configuration value
func (cli *TuningCLI) setConfig(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: config set <path>=<value>")
	}

	// Parse path=value format
	setting := strings.Join(args, " ")
	parts := strings.SplitN(setting, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format, use: path=value")
	}

	path := strings.TrimSpace(parts[0])
	valueStr := strings.TrimSpace(parts[1])

	// Get current value for display
	oldValue, _ := GetFieldValue(cli.config, path)

	// Parse value based on path
	var newValue interface{}
	if strings.Contains(path, "_seconds") || strings.Contains(path, "_size") ||
		strings.Contains(path, "_mb") || strings.Contains(path, "_ms") ||
		strings.Contains(path, "max_") || strings.Contains(path, "pool") {
		// Integer value
		var intVal int
		if _, err := fmt.Sscanf(valueStr, "%d", &intVal); err != nil {
			return fmt.Errorf("invalid integer value: %s", valueStr)
		}
		newValue = intVal
	} else if valueStr == "true" || valueStr == "false" {
		// Boolean value
		newValue = valueStr == "true"
	} else if strings.Contains(path, "threshold") && strings.Contains(valueStr, ".") {
		// Float value
		var floatVal float64
		if _, err := fmt.Sscanf(valueStr, "%f", &floatVal); err != nil {
			return fmt.Errorf("invalid float value: %s", valueStr)
		}
		newValue = floatVal
	} else {
		// String value
		newValue = valueStr
	}

	// Set the value
	if err := SetFieldValue(cli.config, path, newValue); err != nil {
		return err
	}

	// Validate the configuration
	validator := NewConfigValidator()
	if err := validator.Validate(cli.config); err != nil {
		// Revert the change
		SetFieldValue(cli.config, path, oldValue)
		return fmt.Errorf("validation failed: %v", err)
	}

	fmt.Printf("Updated %s: %v -> %v\n", path, oldValue, newValue)

	// Save if config path is set
	if cli.configLoader.configPath != "" {
		yamlStr, err := ConfigToYAML(cli.config)
		if err != nil {
			return err
		}
		if err := ioutil.WriteFile(cli.configLoader.configPath, []byte(yamlStr), 0644); err != nil {
			return fmt.Errorf("failed to save configuration: %v", err)
		}
		fmt.Println("Configuration saved.")
	}

	return nil
}

// applyProfile applies a performance profile
func (cli *TuningCLI) applyProfile(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: config profile <profile-name>")
	}

	profileName := args[0]
	if err := cli.profileMgr.ApplyProfile(profileName, cli.config); err != nil {
		return err
	}

	fmt.Printf("Applied profile: %s\n", profileName)
	fmt.Printf("Description: %s\n", cli.profileMgr.GetProfileDescription(profileName))

	// Save if config path is set
	if cli.configLoader.configPath != "" {
		yamlStr, err := ConfigToYAML(cli.config)
		if err != nil {
			return err
		}
		if err := ioutil.WriteFile(cli.configLoader.configPath, []byte(yamlStr), 0644); err != nil {
			return fmt.Errorf("failed to save configuration: %v", err)
		}
		fmt.Println("Configuration saved.")
	}

	return nil
}

// listProfiles lists available performance profiles
func (cli *TuningCLI) listProfiles() error {
	profiles := cli.profileMgr.ListProfiles()
	
	fmt.Println("Available performance profiles:")
	fmt.Println()
	
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROFILE\tDESCRIPTION")
	fmt.Fprintln(w, "-------\t-----------")
	
	for _, name := range profiles {
		desc := cli.profileMgr.GetProfileDescription(name)
		fmt.Fprintf(w, "%s\t%s\n", name, desc)
	}
	
	w.Flush()
	return nil
}

// exportConfig exports the configuration to a file
func (cli *TuningCLI) exportConfig(args []string) error {
	yamlStr, err := ConfigToYAML(cli.config)
	if err != nil {
		return err
	}

	if len(args) > 0 && args[0] != "-" {
		// Write to file
		if err := ioutil.WriteFile(args[0], []byte(yamlStr), 0644); err != nil {
			return fmt.Errorf("failed to write configuration: %v", err)
		}
		fmt.Printf("Configuration exported to: %s\n", args[0])
	} else {
		// Write to stdout
		fmt.Print(yamlStr)
	}

	return nil
}

// importConfig imports configuration from a file
func (cli *TuningCLI) importConfig(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: config import <file>")
	}

	data, err := ioutil.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	config, err := ConfigFromYAML(string(data))
	if err != nil {
		return fmt.Errorf("failed to parse configuration: %v", err)
	}

	// Validate before applying
	validator := NewConfigValidator()
	if err := validator.Validate(config); err != nil {
		return fmt.Errorf("validation failed: %v", err)
	}

	*cli.config = *config
	fmt.Printf("Configuration imported from: %s\n", args[0])

	// Save if config path is set
	if cli.configLoader.configPath != "" {
		yamlStr, err := ConfigToYAML(cli.config)
		if err != nil {
			return err
		}
		if err := ioutil.WriteFile(cli.configLoader.configPath, []byte(yamlStr), 0644); err != nil {
			return fmt.Errorf("failed to save configuration: %v", err)
		}
		fmt.Println("Configuration saved.")
	}

	return nil
}

// validateConfig validates a configuration file
func (cli *TuningCLI) validateConfig(args []string) error {
	var config *PerformanceConfig

	if len(args) > 0 {
		// Validate external file
		data, err := ioutil.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read file: %v", err)
		}
		config, err = ConfigFromYAML(string(data))
		if err != nil {
			return fmt.Errorf("failed to parse configuration: %v", err)
		}
	} else {
		// Validate current config
		config = cli.config
	}

	validator := NewConfigValidator()
	if err := validator.Validate(config); err != nil {
		if validationErrors, ok := err.(ValidationErrors); ok {
			fmt.Println("Validation errors:")
			for _, e := range validationErrors {
				fmt.Printf("  - %s: %s\n", e.Field, e.Message)
			}
		} else {
			fmt.Printf("Validation error: %v\n", err)
		}
		return fmt.Errorf("validation failed")
	}

	fmt.Println("Configuration is valid.")
	return nil
}

// resetConfig resets configuration to defaults
func (cli *TuningCLI) resetConfig() error {
	// Apply default profile
	if err := cli.profileMgr.ApplyProfile("default", cli.config); err != nil {
		// If default profile doesn't exist, use built-in defaults
		config := &PerformanceConfig{}
		loader := NewConfigLoader("")
		loader.applyDefaults(config)
		*cli.config = *config
	}

	fmt.Println("Configuration reset to defaults.")

	// Save if config path is set
	if cli.configLoader.configPath != "" {
		yamlStr, err := ConfigToYAML(cli.config)
		if err != nil {
			return err
		}
		if err := ioutil.WriteFile(cli.configLoader.configPath, []byte(yamlStr), 0644); err != nil {
			return fmt.Errorf("failed to save configuration: %v", err)
		}
		fmt.Println("Configuration saved.")
	}

	return nil
}

// PrintHelp prints help information for config commands
func PrintConfigHelp() {
	fmt.Println(`Graft Performance Configuration Commands:

  graft config show [--json]           Show current configuration
  graft config get <path>              Get a specific configuration value
  graft config set <path>=<value>      Set a configuration value
  graft config profile <name>          Apply a performance profile
  graft config profiles                List available profiles
  graft config export [file]           Export configuration to file or stdout
  graft config import <file>           Import configuration from file
  graft config validate [file]         Validate configuration
  graft config reset                   Reset to default configuration

Examples:
  graft config get performance.cache.expression_cache_size
  graft config set performance.cache.expression_cache_size=20000
  graft config profile high-concurrency
  graft config export my_config.yaml

Configuration Paths:
  performance.cache.expression_cache_size
  performance.cache.operator_cache_size
  performance.cache.ttl_seconds
  performance.concurrency.max_workers
  performance.concurrency.queue_size
  performance.memory.max_heap_mb
  ... and many more

Available Profiles:
  default           - Balanced configuration for general use
  small-docs        - Optimized for small YAML documents
  large-docs        - Optimized for large YAML documents
  high-concurrency  - Optimized for high concurrent requests
  low-memory        - Optimized for memory-constrained environments`)
}