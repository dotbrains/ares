//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
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
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		fatalf("decode %s: %v", path, err)
	}
	switch mode {
	case "preflight":
		requireString(doc, "schema_version", "ares.preflight.v1")
		requireString(doc, "profile", "basic")
		requireObject(doc, "host")
		requireArray(doc, "plugins")
		requireArray(doc, "checks")
		requireTransaction(doc)
	case "run":
		requireString(doc, "schema_version", "ares.run.v1")
		requireObject(doc, "plan")
		requireObject(doc, "result")
		result := requireObject(doc, "result")
		requireArray(result, "safety_evidence")
		requireTransaction(result)
	case "report":
		requireString(doc, "schema_version", "ares.report.v1")
		requireString(doc, "profile", "basic")
		requireObject(doc, "host")
		requireArray(doc, "plugins")
		requireArray(doc, "safety_evidence")
		requireTransaction(doc)
		requireArray(doc, "applied")
	case "rollback":
		requireString(doc, "schema_version", "ares.rollback.v1")
		requireArray(doc, "applied")
		requireArray(doc, "skipped")
		requireArray(doc, "failed")
	default:
		fatalf("unknown mode %q", mode)
	}
}

func requireTransaction(doc map[string]any) {
	transaction := requireObject(doc, "transaction")
	requireArray(transaction, "files")
	requireArray(transaction, "commands")
	requireArray(transaction, "backups")
	requireArray(transaction, "rollback_steps")
}

func requireString(doc map[string]any, name string, want string) string {
	value, ok := doc[name].(string)
	if !ok {
		fatalf("%s must be a string", name)
	}
	if want != "" && value != want {
		fatalf("%s = %q, want %q", name, value, want)
	}
	return value
}

func requireObject(doc map[string]any, name string) map[string]any {
	value, ok := doc[name].(map[string]any)
	if !ok {
		fatalf("%s must be an object", name)
	}
	return value
}

func requireArray(doc map[string]any, name string) []any {
	value, ok := doc[name].([]any)
	if !ok {
		fatalf("%s must be an array", name)
	}
	return value
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "schema-check: "+format+"\n", args...)
	os.Exit(1)
}
