package graft

import (
	"fmt"
	"sort"
	"strings"
	"github.com/wayneeseguin/graft/log"
	"github.com/starkandwayne/goutils/ansi"
)

// MultiError ...
type MultiError struct {
	Errors []error
}

// Error ...
func (e MultiError) Error() string {
	s := []string{}
	for _, err := range e.Errors {
		s = append(s, fmt.Sprintf(" - %s\n", err))
	}

	sort.Strings(s)
	return ansi.Sprintf("@r{%d} error(s) detected:\n%s\n", len(e.Errors), strings.Join(s, ""))
}

// Count ...
func (e *MultiError) Count() int {
	return len(e.Errors)
}

// Append ...
func (e *MultiError) Append(err error) {
	if err == nil {
		return
	}

	if mult, ok := err.(MultiError); ok {
		e.Errors = append(e.Errors, mult.Errors...)
	} else {
		e.Errors = append(e.Errors, err)
	}
}

//WarningError should produce a warning message to stderr if the context set for
// the error fits the context the error was caught in.
type WarningError struct {
	warning string
	context ErrorContext
}

//An ErrorContext is a flag or set of flags representing the contexts that
// an error should have a special meaning in.
type ErrorContext uint

//Bitwise-or these together to represent several contexts
const (
	eContextAll          = 0
	eContextDefaultMerge = 1 << iota
)

var dontPrintWarning bool

//NewWarningError returns a new WarningError object that has the given warning
// message and context(s) assigned. Assigning no context should mean that all
// contexts are active. Ansi library enabled.
func NewWarningError(context ErrorContext, warning string, args ...interface{}) (err WarningError) {
	err.warning = ansi.Sprintf(warning, args...)
	err.context = context
	return
}

//SilenceWarnings when called with true will make it so that warnings will not
// print when Warn is called. Calling it with false will make warnings visible
// again. Warnings will print by default.
func SilenceWarnings(should bool) {
	dontPrintWarning = should
}

//Error will return the configured warning message as a string
func (e WarningError) Error() string {
	return e.warning
}

//HasContext returns true if the WarningError was configured with the given context (or all).
// False otherwise.
func (e WarningError) HasContext(context ErrorContext) bool {
	return e.context == 0 || (context&e.context > 0)
}

//Warn prints the configured warning to stderr.
func (e WarningError) Warn() {
	if !dontPrintWarning {
		log.PrintfStdErr(ansi.Sprintf("@Y{warning:} %s\n", e.warning))
	}
}

// Enhanced error types for library use

// GraftError is the base error type for all graft operations
type GraftError struct {
	Type    ErrorType
	Message string
	Path    string
	Cause   error
}

func (e *GraftError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s at %s: %s", e.Type, e.Path, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *GraftError) Unwrap() error {
	return e.Cause
}

// ErrorType represents different categories of errors
type ErrorType string

const (
	// ParseError indicates a YAML/JSON parsing error
	ParseError ErrorType = "parse_error"
	
	// MergeError indicates an error during document merging
	MergeError ErrorType = "merge_error"
	
	// EvaluationError indicates an error during operator evaluation
	EvaluationError ErrorType = "evaluation_error"
	
	// OperatorError indicates an error with a specific operator
	OperatorError ErrorType = "operator_error"
	
	// ConfigurationError indicates an invalid configuration
	ConfigurationError ErrorType = "configuration_error"
	
	// ValidationError indicates invalid input or state
	ValidationError ErrorType = "validation_error"
	
	// ExternalError indicates an error from external services (Vault, AWS)
	ExternalError ErrorType = "external_error"
)

// NewParseError creates a new parse error
func NewParseError(message string, cause error) *GraftError {
	return &GraftError{
		Type:    ParseError,
		Message: message,
		Cause:   cause,
	}
}

// NewMergeError creates a new merge error
func NewMergeError(message string, cause error) *GraftError {
	return &GraftError{
		Type:    MergeError,
		Message: message,
		Cause:   cause,
	}
}

// NewEvaluationError creates a new evaluation error with path context
func NewEvaluationError(path, message string, cause error) *GraftError {
	return &GraftError{
		Type:    EvaluationError,
		Message: message,
		Path:    path,
		Cause:   cause,
	}
}

// NewOperatorError creates a new operator error
func NewOperatorError(operator, message string, cause error) *GraftError {
	return &GraftError{
		Type:    OperatorError,
		Message: fmt.Sprintf("operator '%s': %s", operator, message),
		Cause:   cause,
	}
}

// NewConfigurationError creates a new configuration error
func NewConfigurationError(message string) *GraftError {
	return &GraftError{
		Type:    ConfigurationError,
		Message: message,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(message string) *GraftError {
	return &GraftError{
		Type:    ValidationError,
		Message: message,
	}
}

// NewExternalError creates a new external service error
func NewExternalError(service, message string, cause error) *GraftError {
	return &GraftError{
		Type:    ExternalError,
		Message: fmt.Sprintf("%s: %s", service, message),
		Cause:   cause,
	}
}

// IsGraftError checks if an error is a GraftError
func IsGraftError(err error) bool {
	_, ok := err.(*GraftError)
	return ok
}

// GetErrorType returns the error type if it's a GraftError, empty string otherwise
func GetErrorType(err error) ErrorType {
	if graftErr, ok := err.(*GraftError); ok {
		return graftErr.Type
	}
	return ""
}
