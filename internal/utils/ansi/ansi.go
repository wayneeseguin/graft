package ansi

import (
	"fmt"
	"os"
	"regexp"
	"unicode"
)

var (
	colors = map[string]string{
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
		"p": "00;35", // magenta
		"P": "01;35", // magenta (BOLD)

		"c": "00;36", // cyan
		"C": "01;36", // cyan (BOLD)

		"w": "00;37", // white
		"W": "01;37", // white (BOLD)
	}

	re = regexp.MustCompile(`(?s)@[kKrRgGyYbBmMpPcCwW*]{.*?}`)
)

// isTerminal checks if the given file descriptor is a terminal
func isTerminal(fd uintptr) bool {
	// Simple terminal detection without external dependency
	// This replaces github.com/mattn/go-isatty
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

var colorable = isTerminal(os.Stdout.Fd())

// Color enables or disables color output
func Color(c bool) {
	colorable = c
}

func colorize(s string) string {
	return re.ReplaceAllStringFunc(s, func(m string) string {
		if !colorable {
			return m[3 : len(m)-1]
		}
		if m[1:2] == "*" {
			rainbow := "RYGCBM"
			skipCount := 0
			s := ""
			for i, c := range m[3 : len(m)-1] {
				if unicode.IsSpace(c) { //No color wasted on whitespace
					skipCount++
					s += string(c)
					continue
				}
				j := (i - skipCount) % len(rainbow)
				s += "\033[" + colors[rainbow[j:j+1]] + "m" + string(c) + "\033[00m"
			}
			return s
		}
		return "\033[" + colors[m[1:2]] + "m" + m[3:len(m)-1] + "\033[00m"
	})
}

// Sprintf formats according to a format specifier and returns the resulting string with ANSI color codes
func Sprintf(format string, a ...interface{}) string {
	return fmt.Sprintf(colorize(format), a...)
}

// Errorf formats according to a format specifier and returns the string as a value that satisfies error with ANSI color codes
func Errorf(format string, a ...interface{}) error {
	return fmt.Errorf(colorize(format), a...)
}