package reports

import (
	"encoding/json"
	"fmt"
)

type SchemaMode string

const (
	SchemaPreflight SchemaMode = "preflight"
	SchemaRun       SchemaMode = "run"
	SchemaReport    SchemaMode = "report"
	SchemaRollback  SchemaMode = "rollback"
)

func ValidateSchema(mode SchemaMode, data []byte) error {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	switch mode {
	case SchemaPreflight:
		if err := requireString(doc, "schema_version", PreflightSchemaVersion); err != nil {
			return err
		}
		if err := requireString(doc, "profile", "basic"); err != nil {
			return err
		}
		for _, name := range []string{"host", "transaction"} {
			if _, err := requireObject(doc, name); err != nil {
				return err
			}
		}
		for _, name := range []string{"plugins", "checks"} {
			if err := requireArray(doc, name); err != nil {
				return err
			}
		}
		return requireTransaction(doc)
	case SchemaRun:
		if err := requireString(doc, "schema_version", RunSchemaVersion); err != nil {
			return err
		}
		if _, err := requireObject(doc, "plan"); err != nil {
			return err
		}
		result, err := requireObject(doc, "result")
		if err != nil {
			return err
		}
		if err := requireArray(result, "safety_evidence"); err != nil {
			return err
		}
		return requireTransaction(result)
	case SchemaReport:
		if err := requireString(doc, "schema_version", ReportSchemaVersion); err != nil {
			return err
		}
		if err := requireString(doc, "profile", "basic"); err != nil {
			return err
		}
		for _, name := range []string{"host", "transaction"} {
			if _, err := requireObject(doc, name); err != nil {
				return err
			}
		}
		for _, name := range []string{"plugins", "safety_evidence", "applied"} {
			if err := requireArray(doc, name); err != nil {
				return err
			}
		}
		return requireTransaction(doc)
	case SchemaRollback:
		if err := requireString(doc, "schema_version", RollbackSchemaVersion); err != nil {
			return err
		}
		for _, name := range []string{"applied", "skipped", "failed"} {
			if err := requireArray(doc, name); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown mode %q", mode)
	}
}

func requireTransaction(doc map[string]any) error {
	transaction, err := requireObject(doc, "transaction")
	if err != nil {
		return err
	}
	for _, name := range []string{"files", "commands", "backups", "rollback_steps"} {
		if err := requireArray(transaction, name); err != nil {
			return err
		}
	}
	return nil
}

func requireString(doc map[string]any, name string, want string) error {
	value, ok := doc[name].(string)
	if !ok {
		return fmt.Errorf("%s must be a string", name)
	}
	if want != "" && value != want {
		return fmt.Errorf("%s = %q, want %q", name, value, want)
	}
	return nil
}

func requireObject(doc map[string]any, name string) (map[string]any, error) {
	value, ok := doc[name].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an object", name)
	}
	return value, nil
}

func requireArray(doc map[string]any, name string) error {
	if _, ok := doc[name].([]any); !ok {
		return fmt.Errorf("%s must be an array", name)
	}
	return nil
}
