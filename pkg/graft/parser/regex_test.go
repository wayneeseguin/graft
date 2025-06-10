package parser

import (
	"fmt"
	"regexp"
	"testing"
)

func TestRegexPatterns(t *testing.T) {
	input := "(( vault@production \"secret/path:key\" ))"

	patterns := []string{
		`^\(\(\s*([a-zA-Z][a-zA-Z0-9_-]*(?:@[a-zA-Z][a-zA-Z0-9_-]*)?)\((.*)\)\s*\)\)$`,     // (( op@target(x,y,z) )) - no space between op and (
		`^\(\(\s*([a-zA-Z][a-zA-Z0-9_-]*(?:@[a-zA-Z][a-zA-Z0-9_-]*)?)\s+(\(.*\))\s*\)\)$`,  // (( op@target (x,y,z) )) - space between op and (
		`^\(\(\s*([a-zA-Z][a-zA-Z0-9_-]*(?:@[a-zA-Z][a-zA-Z0-9_-]*)?)(?:\s+(.*))?\s*\)\)$`, // (( op@target x y z ))
	}

	for i, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(input) {
			matches := re.FindStringSubmatch(input)
			fmt.Printf("Pattern %d matched: %s\n", i, pattern)
			fmt.Printf("Matches:\n")
			for j, match := range matches {
				fmt.Printf("  [%d]: %q\n", j, match)
			}
			return
		}
	}

	fmt.Printf("No pattern matched for input: %s\n", input)
}
