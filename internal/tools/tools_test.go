package tools

import "testing"

func TestObjectSchemaIncludesDraftAndProperties(t *testing.T) {
	schema := objectSchema()

	if got := schema["$schema"]; got != jsonSchemaDraft07 {
		t.Fatalf("expected draft-07 schema, got %v", got)
	}
	if got := schema["type"]; got != "object" {
		t.Fatalf("expected object type, got %v", got)
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties object, got %T", schema["properties"])
	}
	if len(props) != 0 {
		t.Fatalf("expected empty properties object, got %v", props)
	}
}

func TestIntegerIDSchemaMatchesCompatibilityShape(t *testing.T) {
	schema := integerIDSchema()

	if got := schema["type"]; got != "integer" {
		t.Fatalf("expected integer type, got %v", got)
	}
	if got := schema["exclusiveMinimum"]; got != 0 {
		t.Fatalf("expected exclusiveMinimum 0, got %v", got)
	}
}

func TestEnumArrayUsesExplicitEnums(t *testing.T) {
	schema := objectSchema(optionalEnumArray("status", "subscribed", "pending", "unsubscribed"))
	props := schema["properties"].(map[string]any)
	status := props["status"].(map[string]any)
	items := status["items"].(map[string]any)
	enum := items["enum"].([]string)

	if len(enum) != 3 {
		t.Fatalf("expected 3 enum values, got %d", len(enum))
	}
	if enum[0] != "subscribed" || enum[1] != "pending" || enum[2] != "unsubscribed" {
		t.Fatalf("unexpected enum values: %v", enum)
	}
}
