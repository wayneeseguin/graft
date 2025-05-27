package graft

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/starkandwayne/goutils/tree"
	
	"github.com/wayneeseguin/graft/log"
)

// ParseOpcallCompat provides backward compatibility while allowing enhanced parser usage
func ParseOpcallCompat(phase OperatorPhase, src string) (*Opcall, error) {
	// Only parse strings that look like operator expressions
	if !strings.HasPrefix(strings.TrimSpace(src), "((") || !strings.HasSuffix(strings.TrimSpace(src), "))") {
		log.DEBUG("ParseOpcallCompat: '%s' does not look like an operator expression", src)
		return nil, nil
	}
	
	log.DEBUG("ParseOpcallCompat: checking '%s' in phase %v", src, phase)
	
	// For now, use the basic parser
	// TODO: In phase 2, integrate enhanced parser
	return ParseOpcall(phase, src)
}

// ParseOpcall parses an operator call expression  
func ParseOpcall(phase OperatorPhase, src string) (*Opcall, error) {
	// Basic implementation - this will be enhanced later
	log.DEBUG("ParseOpcall: parsing '%s'", src)
	
	// Check if it's an operator expression
	for _, pattern := range []string{
		`^\Q((\E\s*([a-zA-Z][a-zA-Z0-9_-]*)(?:\s*\((.*)\))?\s*\Q))\E$`, // (( op(x,y,z) ))
		`^\Q((\E\s*([a-zA-Z][a-zA-Z0-9_-]*)(?:\s+(.*))?\s*\Q))\E$`,     // (( op x y z ))
	} {
		re := regexp.MustCompile(pattern)
		if !re.MatchString(src) {
			continue
		}

		m := re.FindStringSubmatch(src)
		log.DEBUG("parsing `%s': looks like a (( %s ... )) operator", src, m[1])

		op := OpRegistry[m[1]]
		if op == nil {
			log.DEBUG("  - skipping: %s is not a known operator", m[1])
			continue
		}
		if op.Phase() != phase {
			log.DEBUG("  - skipping (( %s ... )) operation; it belongs to a different phase", m[1])
			return nil, nil
		}

		// Parse arguments
		args, err := parseArgs(m[2])
		if err != nil {
			return nil, err
		}

		return &Opcall{
			src:  src,
			op:   op,
			args: args,
		}, nil
	}

	return nil, nil
}

// parseArgs parses operator arguments
func parseArgs(src string) ([]*Expr, error) {
	if src == "" {
		return []*Expr{}, nil
	}

	// Simple argument parsing - just split by spaces for now
	// TODO: Implement proper argument parsing with quotes, commas, etc.
	parts := strings.Fields(src)
	args := make([]*Expr, 0, len(parts))
	
	for _, part := range parts {
		// Try to parse as different types
		if part == "nil" || part == "null" {
			args = append(args, &Expr{Type: Literal, Literal: nil})
		} else if part == "true" {
			args = append(args, &Expr{Type: Literal, Literal: true})
		} else if part == "false" {
			args = append(args, &Expr{Type: Literal, Literal: false})
		} else if strings.HasPrefix(part, "$") {
			// Environment variable
			args = append(args, &Expr{Type: EnvVar, Name: part[1:]})
		} else if i, err := strconv.ParseInt(part, 10, 64); err == nil {
			// Integer
			args = append(args, &Expr{Type: Literal, Literal: i})
		} else if f, err := strconv.ParseFloat(part, 64); err == nil {
			// Float
			args = append(args, &Expr{Type: Literal, Literal: f})
		} else if strings.HasPrefix(part, "\"") && strings.HasSuffix(part, "\"") {
			// Quoted string
			args = append(args, &Expr{Type: Literal, Literal: part[1:len(part)-1]})
		} else {
			// Reference
			cursor, err := tree.ParseCursor(part)
			if err != nil {
				return nil, fmt.Errorf("invalid reference: %s", part)
			}
			args = append(args, &Expr{Type: Reference, Reference: cursor})
		}
	}
	
	return args, nil
}