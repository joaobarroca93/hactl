package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// PrintJSON writes v as indented JSON to stdout.
func PrintJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// PrintPlain writes a plain-text line to stdout.
func PrintPlain(s string) {
	fmt.Println(s)
}

// Err writes a formatted error message to stderr and returns an error
// suitable for use as a cobra RunE return value (nil so cobra doesn't
// double-print it).
func Err(format string, args ...any) error {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
	return nil // unreachable, keeps compiler happy
}

// Fatal prints msg to stderr and exits 1.
func Fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
