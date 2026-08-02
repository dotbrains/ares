package reports

import "testing"

func TestValidateSchemaRejectsMissingRunEvidence(t *testing.T) {
	err := ValidateSchema(SchemaRun, []byte(`{"schema_version":"ares.run.v1","plan":{},"result":{"transaction":{"files":[],"commands":[],"backups":[],"rollback_steps":[]}}}`))
	if err == nil {
		t.Fatal("expected schema validation error")
	}
}
