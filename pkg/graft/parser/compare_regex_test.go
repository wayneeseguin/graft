package parser

import (
	"testing"
	"regexp"
	"fmt"
)

func TestCompareRegexResults(t *testing.T) {
	// Test both working and failing cases with original patterns
	workingInput := "(( vault \"secret/path:key\" ))"
	failingInput := "(( vault@production \"secret/path:key\" ))"
	
	// Original patterns (without @ support)
	originalPatterns := []string{
		`^\(\(\s*([a-zA-Z][a-zA-Z0-9_-]*)\((.*)\)\s*\)\)$`, // (( op(x,y,z) )) - no space between op and (
		`^\(\(\s*([a-zA-Z][a-zA-Z0-9_-]*)\s+(\(.*\))\s*\)\)$`, // (( op (x,y,z) )) - space between op and (
		`^\(\(\s*([a-zA-Z][a-zA-Z0-9_-]*)(?:\s+(.*))?\s*\)\)$`,     // (( op x y z ))
	}
	
	// New patterns (with @ support)
	newPatterns := []string{
		`^\(\(\s*([a-zA-Z][a-zA-Z0-9_-]*(?:@[a-zA-Z][a-zA-Z0-9_-]*)?)\((.*)\)\s*\)\)$`, // (( op@target(x,y,z) )) - no space between op and (
		`^\(\(\s*([a-zA-Z][a-zA-Z0-9_-]*(?:@[a-zA-Z][a-zA-Z0-9_-]*)?)\s+(\(.*\))\s*\)\)$`, // (( op@target (x,y,z) )) - space between op and (
		`^\(\(\s*([a-zA-Z][a-zA-Z0-9_-]*(?:@[a-zA-Z][a-zA-Z0-9_-]*)?)(?:\s+(.*))?\s*\)\)$`,     // (( op@target x y z ))
	}
	
	fmt.Printf("=== Testing working input with original patterns ===\n")
	testPatterns(workingInput, originalPatterns)
	
	fmt.Printf("\n=== Testing working input with new patterns ===\n")
	testPatterns(workingInput, newPatterns)
	
	fmt.Printf("\n=== Testing failing input with new patterns ===\n")
	testPatterns(failingInput, newPatterns)
}

func testPatterns(input string, patterns []string) {
	for i, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(input) {
			matches := re.FindStringSubmatch(input)
			fmt.Printf("Pattern %d matched: %s\n", i, pattern)
			fmt.Printf("Input: %q\n", input)
			fmt.Printf("Matches:\n")
			for j, match := range matches {
				fmt.Printf("  [%d]: %q (len=%d)\n", j, match, len(match))
			}
			return
		}
	}
	fmt.Printf("No pattern matched for input: %s\n", input)
}