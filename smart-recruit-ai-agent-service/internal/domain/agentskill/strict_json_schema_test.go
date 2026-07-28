package agentskill

import (
	"strings"
	"testing"
)

func compileStrictJSONSchema(raw []byte) (*StrictOutputSchema, error) {
	return CompileStrictOutputSchema(raw)
}

func TestStrictJSONSchemaValidatesSupportedObjectArrayAndScalarKeywords(t *testing.T) {
	schema, err := compileStrictJSONSchema([]byte(`{
		"$schema":"https://json-schema.org/draft/2020-12/schema",
		"title":"result",
		"description":"structured result",
		"default":{},
		"examples":[{"status":"ready"}],
		"type":"object",
		"minProperties":3,
		"maxProperties":3,
		"properties":{
			"status":{"type":"string","enum":["ready","done"],"minLength":4,"maxLength":5,"pattern":"^[a-z]+$"},
			"score":{"type":["number","null"],"minimum":0,"maximum":10,"exclusiveMinimum":-1,"exclusiveMaximum":11,"multipleOf":0.5},
			"items":{"type":"array","minItems":2,"maxItems":2,"uniqueItems":true,"items":{"type":"integer","minimum":1}}
		},
		"required":["status","score","items"],
		"additionalProperties":false
	}`))
	if err != nil {
		t.Fatalf("compileStrictJSONSchema: %v", err)
	}
	canonical, summary, err := schema.ValidateJSON(` { "items": [1, 2], "score": 2.5, "status": "ready" } `)
	if err != nil {
		t.Fatalf("ValidateJSON: %v (%s)", err, summary)
	}
	if canonical != `{"items":[1,2],"score":2.5,"status":"ready"}` {
		t.Fatalf("canonical = %s", canonical)
	}
	if summary != "stage=validation category=valid count=0 path_depth=2" {
		t.Fatalf("summary = %q", summary)
	}

	for _, content := range []string{
		`{"status":"READY","score":2.3,"items":[1,1]}`,
		`{"status":"ready","score":2.5,"items":[1,2],"extra":true}`,
		`{"status":"ready","score":2.5,"items":[1.5,2]}`,
		`{"status":"ready","score":null,"items":[1]}`,
	} {
		if _, _, err := schema.ValidateJSON(content); err == nil {
			t.Fatalf("ValidateJSON(%s) succeeded, want mismatch", content)
		}
	}
}

func TestStrictJSONSchemaValidatesCompositionConstAndSchemaAdditionalProperties(t *testing.T) {
	schema, err := compileStrictJSONSchema([]byte(`{
		"type":"object",
		"properties":{
			"kind":{"const":"result"},
			"value":{
				"allOf":[{"type":"integer"},{"minimum":1}],
				"anyOf":[{"maximum":2},{"minimum":8}],
				"oneOf":[{"multipleOf":2},{"multipleOf":3}],
				"not":{"const":10}
			}
		},
		"required":["kind","value"],
		"additionalProperties":{"type":"boolean"}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := schema.ValidateJSON(`{"kind":"result","value":8,"flag":true}`); err != nil {
		t.Fatalf("valid composition: %v", err)
	}
	for _, content := range []string{
		`{"kind":"other","value":8}`,
		`{"kind":"result","value":6}`,
		`{"kind":"result","value":10}`,
		`{"kind":"result","value":8,"flag":"yes"}`,
	} {
		if _, _, err := schema.ValidateJSON(content); err == nil {
			t.Fatalf("ValidateJSON(%s) succeeded, want mismatch", content)
		}
	}
}

func TestStrictJSONSchemaRejectsUnknownKeywordsReferencesAndDialects(t *testing.T) {
	cases := []struct {
		name     string
		schema   string
		category string
	}{
		{name: "unknown", schema: `{"type":"object","format":"custom"}`, category: "unknown_keyword"},
		{name: "external ref", schema: `{"$ref":"https://example.invalid/schema"}`, category: "unsupported_reference"},
		{name: "local ref", schema: `{"$ref":"#/$defs/value"}`, category: "unsupported_reference"},
		{name: "defs", schema: `{"$defs":{"value":{"type":"string"}}}`, category: "unknown_keyword"},
		{name: "wrong dialect", schema: `{"$schema":"http://json-schema.org/draft-07/schema#","type":"object"}`, category: "unsupported_dialect"},
		{name: "nested dialect", schema: `{"properties":{"value":{"$schema":"https://json-schema.org/draft/2020-12/schema"}}}`, category: "unsupported_dialect"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := compileStrictJSONSchema([]byte(testCase.schema))
			if err == nil || !strings.Contains(err.Error(), "category="+testCase.category) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestStrictJSONSchemaRejectsInvalidKeywordShapes(t *testing.T) {
	schemas := []string{
		`true`,
		`{"type":[]}`,
		`{"type":["string","string"]}`,
		`{"type":"date"}`,
		`{"properties":[]}`,
		`{"properties":{"value":true}}`,
		`{"required":["value","value"]}`,
		`{"additionalProperties":"false"}`,
		`{"items":[]}`,
		`{"enum":[]}`,
		`{"enum":[1,1.0]}`,
		`{"allOf":[]}`,
		`{"anyOf":{}}`,
		`{"oneOf":[true]}`,
		`{"not":false}`,
		`{"minItems":-1}`,
		`{"maxLength":1.5}`,
		`{"uniqueItems":"true"}`,
		`{"pattern":"["}`,
		`{"minimum":"1"}`,
		`{"exclusiveMinimum":true}`,
		`{"multipleOf":0}`,
		`{"multipleOf":-1}`,
		`{"title":1}`,
		`{"description":[]}`,
		`{"examples":{}}`,
	}
	for _, schema := range schemas {
		if _, err := compileStrictJSONSchema([]byte(schema)); err == nil {
			t.Fatalf("compileStrictJSONSchema(%s) succeeded", schema)
		}
	}
}

func TestStrictJSONSchemaAcceptsEmptyRequiredAndMathematicalNonNegativeIntegers(t *testing.T) {
	schema, err := compileStrictJSONSchema([]byte(`{
		"type":"array",
		"minItems":-0,
		"maxItems":1e1,
		"items":{"type":"object","required":[]}
	}`))
	if err != nil {
		t.Fatalf("compileStrictJSONSchema: %v", err)
	}
	if _, _, err := schema.ValidateJSON(`[{}]`); err != nil {
		t.Fatalf("ValidateJSON: %v", err)
	}
}

func TestStrictJSONSchemaRejectsNonJSONDuplicateKeysAndTrailingValues(t *testing.T) {
	schema, err := compileStrictJSONSchema([]byte(`{"type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		content  string
		category string
	}{
		{content: "```json\n{}\n```", category: "invalid_json"},
		{content: `before {"ok":true}`, category: "invalid_json"},
		{content: `{} {}`, category: "trailing_value"},
		{content: `{"secret":1,"secret":2}`, category: "duplicate_key"},
	}
	for _, testCase := range cases {
		_, summary, err := schema.ValidateJSON(testCase.content)
		if err == nil || !strings.Contains(summary, "category="+testCase.category) {
			t.Fatalf("content=%q summary=%q err=%v", testCase.content, summary, err)
		}
	}
	if _, err := compileStrictJSONSchema([]byte(`{"type":"object","type":"array"}`)); err == nil ||
		!strings.Contains(err.Error(), "category=duplicate_key") {
		t.Fatalf("duplicate schema key error = %v", err)
	}
}

func TestStrictJSONSchemaUsesJSONNumberSemantics(t *testing.T) {
	schema, err := compileStrictJSONSchema([]byte(`{
		"type":"object",
		"properties":{
			"integer":{"type":"integer"},
			"decimal":{"type":"number","const":1,"multipleOf":0.1}
		},
		"required":["integer","decimal"],
		"additionalProperties":false
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := schema.ValidateJSON(`{"integer":1.0e2,"decimal":1.00}`); err != nil {
		t.Fatalf("numeric equivalent values rejected: %v", err)
	}
	if _, _, err := schema.ValidateJSON(`{"integer":1.5,"decimal":1}`); err == nil {
		t.Fatal("non-integer accepted")
	}
	if _, err := compileStrictJSONSchema([]byte(`{"minimum":1e999999999}`)); err == nil ||
		!strings.Contains(err.Error(), "category=invalid_number") {
		t.Fatalf("unbounded numeric semantics error = %v", err)
	}
}

func TestStrictJSONSchemaValidatesArbitraryJSONRootValues(t *testing.T) {
	cases := []struct {
		schema    string
		content   string
		canonical string
	}{
		{schema: `{"type":"null"}`, content: `null`, canonical: `null`},
		{schema: `{"type":"boolean"}`, content: `true`, canonical: `true`},
		{schema: `{"type":"string"}`, content: `"value"`, canonical: `"value"`},
		{schema: `{"type":"number"}`, content: `1.25`, canonical: `1.25`},
		{schema: `{"type":"array","items":{"type":"boolean"}}`, content: `[true,false]`, canonical: `[true,false]`},
		{schema: `{}`, content: `{"free":null}`, canonical: `{"free":null}`},
	}
	for _, testCase := range cases {
		schema, err := compileStrictJSONSchema([]byte(testCase.schema))
		if err != nil {
			t.Fatalf("compileStrictJSONSchema(%s): %v", testCase.schema, err)
		}
		canonical, _, err := schema.ValidateJSON(testCase.content)
		if err != nil {
			t.Fatalf("ValidateJSON(%s): %v", testCase.content, err)
		}
		if canonical != testCase.canonical {
			t.Fatalf("canonical = %s, want %s", canonical, testCase.canonical)
		}
	}
}

func TestStrictJSONSchemaErrorsAndSummariesDoNotLeakOutputOrSchemaIdentity(t *testing.T) {
	const secretProperty = "candidate_private_salary"
	const secretValue = "private-value-74291"
	const schemaIdentity = "schema-version-secret"
	schema, err := compileStrictJSONSchema([]byte(`{
		"$schema":"https://json-schema.org/draft/2020-12/schema",
		"type":"object",
		"properties":{"candidate_private_salary":{"type":"integer"}},
		"required":["candidate_private_salary"],
		"additionalProperties":false
	}`))
	if err != nil {
		t.Fatal(err)
	}
	_, summary, err := schema.ValidateJSON(`{"candidate_private_salary":"private-value-74291"}`)
	if err == nil {
		t.Fatal("validation succeeded")
	}
	combined := summary + " " + err.Error()
	for _, forbidden := range []string{secretProperty, secretValue, schemaIdentity} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("privacy output leaked %q: %s", forbidden, combined)
		}
	}
	if !strings.Contains(combined, "stage=validation") ||
		!strings.Contains(combined, "category=schema_mismatch") ||
		!strings.Contains(combined, "count=") ||
		!strings.Contains(combined, "path_depth=") {
		t.Fatalf("privacy-safe diagnostic incomplete: %s", combined)
	}

	_, compileErr := compileStrictJSONSchema([]byte(`{
		"properties":{"candidate_private_salary":{"format":"schema-version-secret"}}
	}`))
	if compileErr == nil || strings.Contains(compileErr.Error(), secretProperty) || strings.Contains(compileErr.Error(), schemaIdentity) {
		t.Fatalf("compile error leaked schema identity/property: %v", compileErr)
	}
}
