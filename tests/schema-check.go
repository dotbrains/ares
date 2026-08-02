//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/dotbrains/ares/internal/reports"
)

func main() {
	if len(os.Args) != 3 {
		fatalf("usage: go run tests/schema-check.go <preflight|run|report|rollback> <path>")
	}
	mode := os.Args[1]
	path := os.Args[2]
	data, err := os.ReadFile(path)
	if err != nil {
		fatalf("read %s: %v", path, err)
	}
	if err := reports.ValidateSchema(reports.SchemaMode(mode), data); err != nil {
		fatalf("%v", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "schema-check: "+format+"\n", args...)
	os.Exit(1)
}
