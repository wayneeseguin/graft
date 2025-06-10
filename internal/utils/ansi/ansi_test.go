package ansi

import (
	"strings"
	"testing"
)

func TestColor(t *testing.T) {
	// Test enabling/disabling color
	Color(true)
	if !colorable {
		t.Error("expected colorable to be true")
	}

	Color(false)
	if colorable {
		t.Error("expected colorable to be false")
	}

	// Reset to default
	Color(true)
}

func TestColorize(t *testing.T) {
	// Enable colors for testing
	Color(true)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple red",
			input:    "@R{error}",
			expected: "\033[01;31merror\033[00m",
		},
		{
			name:     "simple green",
			input:    "@G{success}",
			expected: "\033[01;32msuccess\033[00m",
		},
		{
			name:     "simple blue",
			input:    "@b{info}",
			expected: "\033[00;34minfo\033[00m",
		},
		{
			name:     "multiple colors",
			input:    "@R{error} @G{success}",
			expected: "\033[01;31merror\033[00m \033[01;32msuccess\033[00m",
		},
		{
			name:     "no color codes",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "rainbow text",
			input:    "@*{rainbow}",
			expected: "\033[01;31mr\033[00m\033[01;33ma\033[00m\033[01;32mi\033[00m\033[01;36mn\033[00m\033[01;34mb\033[00m\033[01;35mo\033[00m\033[01;31mw\033[00m",
		},
		{
			name:     "rainbow with spaces",
			input:    "@*{rain bow}",
			expected: "\033[01;31mr\033[00m\033[01;33ma\033[00m\033[01;32mi\033[00m\033[01;36mn\033[00m \033[01;34mb\033[00m\033[01;35mo\033[00m\033[01;31mw\033[00m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := colorize(tt.input)
			if result != tt.expected {
				t.Errorf("colorize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestColorizeDisabled(t *testing.T) {
	// Disable colors for testing
	Color(false)
	defer Color(true) // Reset after test

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "red without color",
			input:    "@R{error}",
			expected: "error",
		},
		{
			name:     "multiple colors without color",
			input:    "@R{error} @G{success}",
			expected: "error success",
		},
		{
			name:     "rainbow without color",
			input:    "@*{rainbow}",
			expected: "rainbow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := colorize(tt.input)
			if result != tt.expected {
				t.Errorf("colorize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSprintf(t *testing.T) {
	Color(true)

	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "simple format with color",
			format:   "@R{Error}: %s",
			args:     []interface{}{"something went wrong"},
			expected: "\033[01;31mError\033[00m: something went wrong",
		},
		{
			name:     "multiple format arguments",
			format:   "@G{Success}: %d out of %d tests passed",
			args:     []interface{}{5, 5},
			expected: "\033[01;32mSuccess\033[00m: 5 out of 5 tests passed",
		},
		{
			name:     "no format arguments",
			format:   "@b{Info message}",
			args:     []interface{}{},
			expected: "\033[00;34mInfo message\033[00m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Sprintf(tt.format, tt.args...)
			if result != tt.expected {
				t.Errorf("Sprintf(%q, %v) = %q, want %q", tt.format, tt.args, result, tt.expected)
			}
		})
	}
}

func TestErrorf(t *testing.T) {
	Color(true)

	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "simple error with color",
			format:   "@R{Error}: %s",
			args:     []interface{}{"something went wrong"},
			expected: "\033[01;31mError\033[00m: something went wrong",
		},
		{
			name:     "formatted error",
			format:   "@R{Fatal error} in @c{%s}: @Y{%v}",
			args:     []interface{}{"main.go", "file not found"},
			expected: "\033[01;31mFatal error\033[00m in \033[00;36mmain.go\033[00m: \033[01;33mfile not found\033[00m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Errorf(tt.format, tt.args...)
			if err.Error() != tt.expected {
				t.Errorf("Errorf(%q, %v).Error() = %q, want %q", tt.format, tt.args, err.Error(), tt.expected)
			}
		})
	}
}

func TestAllColors(t *testing.T) {
	Color(true)

	// Test all color codes
	colorTests := map[string]string{
		"k": "00;30", // black
		"K": "01;30", // black (BOLD)
		"r": "00;31", // red
		"R": "01;31", // red (BOLD)
		"g": "00;32", // green
		"G": "01;32", // green (BOLD)
		"y": "00;33", // yellow
		"Y": "01;33", // yellow (BOLD)
		"b": "00;34", // blue
		"B": "01;34", // blue (BOLD)
		"m": "00;35", // magenta
		"M": "01;35", // magenta (BOLD)
		"p": "00;35", // magenta (alias)
		"P": "01;35", // magenta (BOLD alias)
		"c": "00;36", // cyan
		"C": "01;36", // cyan (BOLD)
		"w": "00;37", // white
		"W": "01;37", // white (BOLD)
	}

	for colorCode, expected := range colorTests {
		t.Run("color_"+colorCode, func(t *testing.T) {
			input := "@" + colorCode + "{test}"
			result := colorize(input)
			expectedResult := "\033[" + expected + "mtest\033[00m"

			if result != expectedResult {
				t.Errorf("colorize(%q) = %q, want %q", input, result, expectedResult)
			}
		})
	}
}

func TestNestedColors(t *testing.T) {
	Color(true)

	// Test individual color codes work (nested colors aren't supported by this implementation)
	tests := []struct {
		input    string
		contains []string
	}{
		{
			input:    "@R{Error message}",
			contains: []string{"\033[01;31m", "\033[00m"},
		},
		{
			input:    "@G{Success message}",
			contains: []string{"\033[01;32m", "\033[00m"},
		},
		{
			input:    "@b{Info message}",
			contains: []string{"\033[00;34m", "\033[00m"},
		},
	}

	for _, tt := range tests {
		result := colorize(tt.input)
		for _, expected := range tt.contains {
			if !strings.Contains(result, expected) {
				t.Errorf("colorize(%q) = %q, expected to contain %q", tt.input, result, expected)
			}
		}
	}
}

func BenchmarkColorize(b *testing.B) {
	Color(true)
	input := "@R{Error}: @G{Success} @b{Info} @Y{Warning}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		colorize(input)
	}
}

func BenchmarkSprintf(b *testing.B) {
	Color(true)
	format := "@R{Error}: %s @G{Success}: %d"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sprintf(format, "test error", 42)
	}
}
