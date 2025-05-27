package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TuningAPI provides HTTP endpoints for runtime configuration tuning
type TuningAPI struct {
	mu           sync.RWMutex
	config       *PerformanceConfig
	configLoader *ConfigLoader
	validator    *ConfigValidator
	history      *TuningHistory
	server       *http.Server
}

// TuningChange represents a configuration change
type TuningChange struct {
	Field        string      `json:"field"`
	OldValue     interface{} `json:"old_value"`
	NewValue     interface{} `json:"new_value"`
	Timestamp    time.Time   `json:"timestamp"`
	Applied      bool        `json:"applied"`
	ErrorMessage string      `json:"error_message,omitempty"`
}

// TuningHistory tracks configuration changes
type TuningHistory struct {
	mu      sync.RWMutex
	changes []TuningChange
	maxSize int
}

// NewTuningHistory creates a new tuning history tracker
func NewTuningHistory(maxSize int) *TuningHistory {
	return &TuningHistory{
		changes: make([]TuningChange, 0),
		maxSize: maxSize,
	}
}

// AddChange adds a change to the history
func (h *TuningHistory) AddChange(change TuningChange) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.changes = append(h.changes, change)
	if len(h.changes) > h.maxSize {
		h.changes = h.changes[1:]
	}
}

// GetChanges returns recent changes
func (h *TuningHistory) GetChanges(limit int) []TuningChange {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if limit <= 0 || limit > len(h.changes) {
		limit = len(h.changes)
	}

	start := len(h.changes) - limit
	if start < 0 {
		start = 0
	}

	result := make([]TuningChange, limit)
	copy(result, h.changes[start:])
	return result
}

// NewTuningAPI creates a new tuning API server
func NewTuningAPI(config *PerformanceConfig, loader *ConfigLoader) *TuningAPI {
	return &TuningAPI{
		config:       config,
		configLoader: loader,
		validator:    NewConfigValidator(),
		history:      NewTuningHistory(1000),
	}
}

// Start starts the tuning API server
func (api *TuningAPI) Start(addr string) error {
	mux := http.NewServeMux()

	// Register endpoints
	mux.HandleFunc("/api/config", api.handleConfig)
	mux.HandleFunc("/api/config/get", api.handleGetConfig)
	mux.HandleFunc("/api/config/set", api.handleSetConfig)
	mux.HandleFunc("/api/config/validate", api.handleValidateConfig)
	mux.HandleFunc("/api/config/reload", api.handleReloadConfig)
	mux.HandleFunc("/api/config/export", api.handleExportConfig)
	mux.HandleFunc("/api/config/import", api.handleImportConfig)
	mux.HandleFunc("/api/profiles", api.handleProfiles)
	mux.HandleFunc("/api/profiles/apply", api.handleApplyProfile)
	mux.HandleFunc("/api/history", api.handleHistory)
	mux.HandleFunc("/api/metrics", api.handleMetrics)
	mux.HandleFunc("/api/health", api.handleHealth)

	api.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return api.server.ListenAndServe()
}

// Stop stops the tuning API server
func (api *TuningAPI) Stop() error {
	if api.server != nil {
		return api.server.Close()
	}
	return nil
}

// handleConfig returns the current configuration
func (api *TuningAPI) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	api.mu.RLock()
	config := api.config
	api.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// handleGetConfig gets a specific configuration value
func (api *TuningAPI) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Missing 'path' parameter", http.StatusBadRequest)
		return
	}

	api.mu.RLock()
	value, err := GetFieldValue(api.config, path)
	api.mu.RUnlock()

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"path":  path,
		"value": value,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSetConfig sets a configuration value
func (api *TuningAPI) handleSetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Path  string `json:"path"`
		Value string `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.Path == "" {
		http.Error(w, "Missing 'path' field", http.StatusBadRequest)
		return
	}

	// Get current value for history
	api.mu.RLock()
	oldValue, _ := GetFieldValue(api.config, request.Path)
	api.mu.RUnlock()

	// Parse the new value
	var newValue interface{}
	if strings.Contains(request.Path, "_seconds") || strings.Contains(request.Path, "_size") ||
		strings.Contains(request.Path, "_mb") || strings.Contains(request.Path, "_ms") {
		// Try to parse as integer
		if val, err := strconv.Atoi(request.Value); err == nil {
			newValue = val
		} else {
			http.Error(w, "Invalid integer value", http.StatusBadRequest)
			return
		}
	} else if request.Value == "true" || request.Value == "false" {
		// Parse as boolean
		newValue = request.Value == "true"
	} else if strings.Contains(request.Path, "threshold") {
		// Try to parse as float
		if val, err := strconv.ParseFloat(request.Value, 64); err == nil {
			newValue = val
		} else {
			http.Error(w, "Invalid float value", http.StatusBadRequest)
			return
		}
	} else {
		// Keep as string
		newValue = request.Value
	}

	// Create a copy of the config for validation
	api.mu.Lock()
	configCopy := *api.config
	err := SetFieldValue(&configCopy, request.Path, newValue)
	if err != nil {
		api.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate the new configuration
	if err := api.validator.Validate(&configCopy); err != nil {
		api.mu.Unlock()
		http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// Apply the change
	*api.config = configCopy
	api.mu.Unlock()

	// Record the change
	change := TuningChange{
		Field:     request.Path,
		OldValue:  oldValue,
		NewValue:  newValue,
		Timestamp: time.Now(),
		Applied:   true,
	}
	api.history.AddChange(change)

	// Notify change handlers
	if api.configLoader != nil {
		for _, handler := range api.configLoader.changeHandlers {
			handler(api.config)
		}
	}

	response := map[string]interface{}{
		"success": true,
		"path":    request.Path,
		"value":   newValue,
		"message": "Configuration updated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleValidateConfig validates a configuration change
func (api *TuningAPI) handleValidateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config PerformanceConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := api.validator.Validate(&config)
	response := map[string]interface{}{
		"valid": err == nil,
	}

	if err != nil {
		if validationErrors, ok := err.(ValidationErrors); ok {
			var errors []map[string]string
			for _, e := range validationErrors {
				errors = append(errors, map[string]string{
					"field":   e.Field,
					"message": e.Message,
				})
			}
			response["errors"] = errors
		} else {
			response["error"] = err.Error()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleReloadConfig reloads configuration from file
func (api *TuningAPI) handleReloadConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if api.configLoader == nil {
		http.Error(w, "Configuration loader not available", http.StatusInternalServerError)
		return
	}

	if err := api.configLoader.Reload(); err != nil {
		http.Error(w, fmt.Sprintf("Reload failed: %v", err), http.StatusInternalServerError)
		return
	}

	api.mu.Lock()
	api.config = api.configLoader.GetConfig()
	api.mu.Unlock()

	response := map[string]interface{}{
		"success": true,
		"message": "Configuration reloaded successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleExportConfig exports the current configuration
func (api *TuningAPI) handleExportConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	api.mu.RLock()
	yamlStr, err := ConfigToYAML(api.config)
	api.mu.RUnlock()

	if err != nil {
		http.Error(w, fmt.Sprintf("Export failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=graft_performance.yaml")
	w.Write([]byte(yamlStr))
}

// handleImportConfig imports a configuration
func (api *TuningAPI) handleImportConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	config, err := ConfigFromYAML(string(body))
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid YAML: %v", err), http.StatusBadRequest)
		return
	}

	// Validate the configuration
	if err := api.validator.Validate(config); err != nil {
		http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusBadRequest)
		return
	}

	api.mu.Lock()
	*api.config = *config
	api.mu.Unlock()

	// Notify change handlers
	if api.configLoader != nil {
		for _, handler := range api.configLoader.changeHandlers {
			handler(api.config)
		}
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Configuration imported successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleProfiles lists available profiles
func (api *TuningAPI) handleProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pm := GetProfileManager()
	profiles := pm.ListProfiles()

	var profileList []map[string]string
	for _, name := range profiles {
		profileList = append(profileList, map[string]string{
			"name":        name,
			"description": pm.GetProfileDescription(name),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profileList)
}

// handleApplyProfile applies a performance profile
func (api *TuningAPI) handleApplyProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Profile string `json:"profile"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.Profile == "" {
		http.Error(w, "Missing 'profile' field", http.StatusBadRequest)
		return
	}

	pm := GetProfileManager()
	
	api.mu.Lock()
	err := pm.ApplyProfile(request.Profile, api.config)
	api.mu.Unlock()

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Record the change
	change := TuningChange{
		Field:     "profile",
		OldValue:  "custom",
		NewValue:  request.Profile,
		Timestamp: time.Now(),
		Applied:   true,
	}
	api.history.AddChange(change)

	// Notify change handlers
	if api.configLoader != nil {
		for _, handler := range api.configLoader.changeHandlers {
			handler(api.config)
		}
	}

	response := map[string]interface{}{
		"success": true,
		"profile": request.Profile,
		"message": fmt.Sprintf("Profile '%s' applied successfully", request.Profile),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHistory returns tuning history
func (api *TuningAPI) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
		}
	}

	changes := api.history.GetChanges(limit)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(changes)
}

// handleMetrics returns performance metrics
func (api *TuningAPI) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// This would integrate with the actual metrics system
	// For now, return placeholder metrics
	metrics := map[string]interface{}{
		"cache": map[string]interface{}{
			"expression_hit_rate": 0.85,
			"operator_hit_rate":   0.92,
			"token_hit_rate":      0.78,
		},
		"concurrency": map[string]interface{}{
			"active_workers": 45,
			"queue_depth":    123,
			"throughput_ops": 823,
		},
		"memory": map[string]interface{}{
			"heap_used_mb":  2341,
			"heap_total_mb": 4096,
			"gc_frequency":  1.2,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// handleHealth returns API health status
func (api *TuningAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}