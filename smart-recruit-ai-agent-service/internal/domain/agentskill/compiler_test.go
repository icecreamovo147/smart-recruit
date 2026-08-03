package agentskill

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	domaintokenbudget "smart-recruit-ai-agent-service/internal/domain/tokenbudget"
)

func TestCompileNormalizesDeterministically(t *testing.T) {
	first := validDraft()
	first.Manifest.RequiredCapabilities = []string{" search ", "calendar", "search", ""}
	first.Manifest.TriggerKeywords = []string{"招聘", " candidate ", "招聘"}
	first.Manifest.SemanticTags = []string{"z", "a", "z"}
	first.Manifest.EvaluationCriteria = []string{" complete ", "accurate", "complete"}
	first.Manifest.OutputContract = OutputContract{
		Mode:   OutputModeAdvisory,
		Schema: json.RawMessage(`{ "type": "object", "properties": { "z": {"type":"string"}, "a":{"type":"number"} } }`),
	}
	first.Core.ContentMarkdown = "\r\n  Follow the policy.\r\n\r\n"
	first.Sections = []ReferenceSection{
		{
			SectionKey:      " details ",
			Title:           " Details ",
			ContentMarkdown: "\r\nExplain details.\r\n",
			TriggerTerms:    []string{" detail ", "explain", "detail"},
			SemanticTags:    []string{"z", "a"},
			PlannerIntents:  []string{"answer", "analyze", "answer"},
			Ordinal:         20,
		},
		{
			SectionKey:      "examples",
			Title:           "Examples",
			ContentMarkdown: "Provide examples.",
			Ordinal:         10,
		},
	}

	second := validDraft()
	second.Manifest.RequiredCapabilities = []string{"calendar", "search"}
	second.Manifest.TriggerKeywords = []string{"candidate", "招聘"}
	second.Manifest.SemanticTags = []string{"a", "z"}
	second.Manifest.EvaluationCriteria = []string{"accurate", "complete"}
	second.Manifest.OutputContract = OutputContract{
		Mode:   OutputModeAdvisory,
		Schema: json.RawMessage(`{"properties":{"a":{"type":"number"},"z":{"type":"string"}},"type":"object"}`),
	}
	second.Core.ContentMarkdown = "Follow the policy.\n"
	second.Sections = []ReferenceSection{
		{
			SectionKey:      "examples",
			Title:           "Examples",
			ContentMarkdown: "Provide examples.\n",
			Ordinal:         10,
		},
		{
			SectionKey:      "details",
			Title:           "Details",
			ContentMarkdown: "Explain details.\n",
			TriggerTerms:    []string{"explain", "detail"},
			SemanticTags:    []string{"a", "z"},
			PlannerIntents:  []string{"analyze", "answer"},
			Ordinal:         20,
		},
	}

	compiledFirst, err := Compile(first)
	if err != nil {
		t.Fatalf("Compile(first) error = %v", err)
	}
	compiledSecond, err := Compile(second)
	if err != nil {
		t.Fatalf("Compile(second) error = %v", err)
	}

	if compiledFirst.CompiledHash != compiledSecond.CompiledHash {
		t.Fatalf("hashes differ:\n%s\n%s", compiledFirst.CompiledHash, compiledSecond.CompiledHash)
	}
	if compiledFirst.CanonicalJSON != compiledSecond.CanonicalJSON {
		t.Fatalf("canonical JSON differs:\n%s\n%s", compiledFirst.CanonicalJSON, compiledSecond.CanonicalJSON)
	}
	if strings.Contains(compiledFirst.CanonicalJSON, "estimated_tokens") ||
		strings.Contains(compiledFirst.CanonicalJSON, "content_hash") {
		t.Fatalf("canonical source package contains derived fields: %s", compiledFirst.CanonicalJSON)
	}
	if compiledFirst.CompiledMarkdown != compiledSecond.CompiledMarkdown {
		t.Fatalf("compiled Markdown differs:\n%s\n%s", compiledFirst.CompiledMarkdown, compiledSecond.CompiledMarkdown)
	}
	if got := compiledFirst.Core.ContentMarkdown; got != "Follow the policy.\n" {
		t.Fatalf("Core.ContentMarkdown = %q", got)
	}
	if got := sectionKeys(compiledFirst.Sections); strings.Join(got, ",") != "examples,details" {
		t.Fatalf("section order = %v", got)
	}
	if got := compiledFirst.Manifest.RequiredCapabilities; strings.Join(got, ",") != "calendar,search" {
		t.Fatalf("RequiredCapabilities = %v", got)
	}
	if got := string(compiledFirst.Manifest.OutputContract.Schema); got != `{"properties":{"a":{"type":"number"},"z":{"type":"string"}},"type":"object"}` {
		t.Fatalf("canonical schema = %s", got)
	}
	if !strings.HasSuffix(compiledFirst.CompiledMarkdown, "\n") ||
		strings.HasSuffix(compiledFirst.CompiledMarkdown, "\n\n") {
		t.Fatalf("compiled Markdown must have exactly one trailing newline: %q", compiledFirst.CompiledMarkdown)
	}
}

func TestCompileRejectsUnsupportedStrictSchemaBeforeReleaseEvaluation(t *testing.T) {
	draft := validDraft()
	draft.Manifest.OutputContract = OutputContract{
		Mode:     OutputModeStrict,
		SchemaID: "strict-test-v1",
		Schema:   json.RawMessage(`{"type":"object","properties":{"created_at":{"type":"string","format":"date-time"}}}`),
	}
	_, err := Compile(draft)
	if err == nil {
		t.Fatal("Compile() accepted unsupported strict-schema keyword")
	}
	var compileErr *CompileError
	if !errors.As(err, &compileErr) || compileErr.Code != CodePackageInvalid {
		t.Fatalf("Compile() error = %v", err)
	}
}

func TestCompileRejectsDuplicateStrictSchemaKeysBeforeNormalization(t *testing.T) {
	tests := []struct {
		name   string
		schema json.RawMessage
	}{
		{
			name:   "root keyword",
			schema: json.RawMessage(`{"type":"object","type":"string"}`),
		},
		{
			name:   "nested property keyword",
			schema: json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean","type":"string"}}}`),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			draft := validDraft()
			draft.Manifest.OutputContract = OutputContract{
				Mode:     OutputModeStrict,
				SchemaID: "strict-test-v1",
				Schema:   test.schema,
			}
			if _, err := Compile(draft); err == nil {
				t.Fatal("Compile() accepted duplicate strict-schema key")
			}
		})
	}
}

func TestCompileDerivesActivationPolicy(t *testing.T) {
	tests := []struct {
		name string
		risk RiskLevel
		want ActivationPolicy
	}{
		{name: "low", risk: RiskLevelLow, want: ActivationPolicyAuto},
		{name: "medium", risk: RiskLevelMedium, want: ActivationPolicyAuto},
		{name: "high", risk: RiskLevelHigh, want: ActivationPolicyConfirm},
		{name: "critical", risk: RiskLevelCritical, want: ActivationPolicyManualOnly},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			draft := validDraft()
			draft.Manifest.RiskLevel = test.risk
			draft.Manifest.ActivationPolicy = ""
			compiled, err := Compile(draft)
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}
			if compiled.Manifest.ActivationPolicy != test.want {
				t.Fatalf("ActivationPolicy = %q, want %q", compiled.Manifest.ActivationPolicy, test.want)
			}
		})
	}
}

func TestCompileRejectsInvalidManifest(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*PackageDraft)
		wantCode  string
		wantField string
	}{
		{
			name:      "schema version",
			mutate:    func(draft *PackageDraft) { draft.Manifest.SchemaVersion = 1 },
			wantCode:  CodePackageInvalid,
			wantField: "manifest.schema_version",
		},
		{
			name:      "risk",
			mutate:    func(draft *PackageDraft) { draft.Manifest.RiskLevel = RiskLevel("severe") },
			wantCode:  CodePackageInvalid,
			wantField: "manifest.risk_level",
		},
		{
			name: "contradictory activation policy",
			mutate: func(draft *PackageDraft) {
				draft.Manifest.RiskLevel = RiskLevelHigh
				draft.Manifest.ActivationPolicy = ActivationPolicyAuto
			},
			wantCode:  CodePackageInvalid,
			wantField: "manifest.activation_policy",
		},
		{
			name:      "composition role",
			mutate:    func(draft *PackageDraft) { draft.Manifest.Composition.Role = CompositionRole("peer") },
			wantCode:  CodePackageInvalid,
			wantField: "manifest.composition.role",
		},
		{
			name:      "output mode",
			mutate:    func(draft *PackageDraft) { draft.Manifest.OutputContract.Mode = OutputMode("xml") },
			wantCode:  CodePackageInvalid,
			wantField: "manifest.output_contract.mode",
		},
		{
			name: "invalid output schema",
			mutate: func(draft *PackageDraft) {
				draft.Manifest.OutputContract = OutputContract{
					Mode:   OutputModeAdvisory,
					Schema: json.RawMessage(`[]`),
				}
			},
			wantCode:  CodePackageInvalid,
			wantField: "manifest.output_contract.schema",
		},
		{
			name: "strict requires schema id",
			mutate: func(draft *PackageDraft) {
				draft.Manifest.OutputContract = OutputContract{
					Mode:   OutputModeStrict,
					Schema: json.RawMessage(`{}`),
				}
			},
			wantCode:  CodePackageInvalid,
			wantField: "manifest.output_contract.schema_id",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			draft := validDraft()
			test.mutate(&draft)
			_, err := Compile(draft)
			assertCompileError(t, err, test.wantCode, test.wantField)
		})
	}
}

func TestCompileRejectsSupportingOutputContract(t *testing.T) {
	draft := validDraft()
	draft.Manifest.Composition.Role = CompositionRoleSupporting
	draft.Manifest.OutputContract = OutputContract{
		Mode:   OutputModeAdvisory,
		Schema: json.RawMessage(`{}`),
	}

	_, err := Compile(draft)
	assertCompileError(t, err, CodeCompositionConflict, "manifest.output_contract")
}

func TestCompileEnforcesOutputSchemaIDByteBoundary(t *testing.T) {
	exactASCII := strings.Repeat("a", MaxOutputSchemaIDBytes)
	exactMultibyte := strings.Repeat("界", MaxOutputSchemaIDBytes/len("界")) +
		strings.Repeat("a", MaxOutputSchemaIDBytes%len("界"))
	if len(exactMultibyte) != MaxOutputSchemaIDBytes {
		t.Fatalf("multibyte fixture bytes=%d, want %d", len(exactMultibyte), MaxOutputSchemaIDBytes)
	}

	for _, test := range []struct {
		name     string
		mode     OutputMode
		schemaID string
		wantErr  bool
	}{
		{name: "advisory exact ASCII boundary", mode: OutputModeAdvisory, schemaID: exactASCII},
		{name: "strict exact multibyte boundary", mode: OutputModeStrict, schemaID: exactMultibyte},
		{name: "advisory over byte boundary", mode: OutputModeAdvisory, schemaID: exactASCII + "a", wantErr: true},
		{name: "strict multibyte over byte boundary", mode: OutputModeStrict, schemaID: exactMultibyte + "界", wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			draft := validDraft()
			draft.Manifest.OutputContract = OutputContract{
				Mode:     test.mode,
				SchemaID: test.schemaID,
				Schema:   json.RawMessage(`{"type":"object"}`),
			}
			compiled, err := Compile(draft)
			if test.wantErr {
				assertCompileError(t, err, CodePackageInvalid, "manifest.output_contract.schema_id")
				return
			}
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}
			if got := len(compiled.Manifest.OutputContract.SchemaID); got != MaxOutputSchemaIDBytes {
				t.Fatalf("compiled schema_id bytes=%d, want %d", got, MaxOutputSchemaIDBytes)
			}
		})
	}
}

func TestCompileEnforcesCanonicalOutputSchemaByteBoundary(t *testing.T) {
	schemaWithSize := func(size int) json.RawMessage {
		return json.RawMessage(`{"x":"` +
			strings.Repeat("a", size-len(`{"x":""}`)) +
			`"}`)
	}
	for _, test := range []struct {
		name    string
		schema  json.RawMessage
		wantErr bool
	}{
		{name: "exact boundary", schema: schemaWithSize(MaxOutputSchemaBytes)},
		{
			name:    "canonical escaping over boundary",
			schema:  json.RawMessage(`{"x":"` + strings.Repeat("<", MaxOutputSchemaBytes/4) + `"}`),
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			draft := validDraft()
			draft.Manifest.OutputContract = OutputContract{
				Mode:   OutputModeAdvisory,
				Schema: test.schema,
			}
			compiled, err := Compile(draft)
			if test.wantErr {
				assertCompileError(t, err, CodePackageInvalid, "manifest.output_contract.schema")
				return
			}
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}
			if got := len(compiled.Manifest.OutputContract.Schema); got != MaxOutputSchemaBytes {
				t.Fatalf("compiled schema bytes=%d, want %d", got, MaxOutputSchemaBytes)
			}
		})
	}
}

func TestCompileEnforcesCoreBudget(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantCode string
	}{
		{name: "at limit", content: strings.Repeat("a", MaxCoreTokens*4)},
		{name: "over limit", content: strings.Repeat("a", MaxCoreTokens*4+1), wantCode: CodeCoreBudgetExceeded},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			draft := validDraft()
			draft.Core.ContentMarkdown = test.content
			_, err := Compile(draft)
			if test.wantCode == "" && err != nil {
				t.Fatalf("Compile() error = %v", err)
			}
			if test.wantCode != "" {
				assertCompileError(t, err, test.wantCode, "core.content_markdown")
			}
		})
	}
}

func TestCompileEnforcesSectionRules(t *testing.T) {
	tests := []struct {
		name      string
		sections  []ReferenceSection
		wantField string
	}{
		{
			name: "invalid key",
			sections: []ReferenceSection{
				validSection("Upper", 1, "content"),
			},
			wantField: "sections[0].section_key",
		},
		{
			name: "one character key",
			sections: []ReferenceSection{
				validSection("a", 1, "content"),
			},
			wantField: "sections[0].section_key",
		},
		{
			name: "duplicate normalized key",
			sections: []ReferenceSection{
				validSection("same", 1, "one"),
				validSection(" same ", 2, "two"),
			},
			wantField: "sections[1].section_key",
		},
		{
			name: "negative ordinal",
			sections: []ReferenceSection{
				validSection("negative", -1, "content"),
			},
			wantField: "sections[0].ordinal",
		},
		{
			name: "section over budget",
			sections: []ReferenceSection{
				validSection("oversized", 1, strings.Repeat("a", MaxSectionTokens*4+1)),
			},
			wantField: "sections[0].content_markdown",
		},
		{
			name:      "too many sections",
			sections:  makeSections(MaxSections+1, "content"),
			wantField: "sections",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			draft := validDraft()
			draft.Sections = test.sections
			_, err := Compile(draft)
			assertCompileError(t, err, CodeSectionInvalid, test.wantField)
		})
	}
}

func TestCompileAllowsSectionAtBudgetLimit(t *testing.T) {
	draft := validDraft()
	draft.Sections = []ReferenceSection{
		validSection("at-limit", 1, strings.Repeat("a", MaxSectionTokens*4)),
	}

	compiled, err := Compile(draft)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if got := compiled.Sections[0].EstimatedTokens; got != MaxSectionTokens {
		t.Fatalf("EstimatedTokens = %d, want %d", got, MaxSectionTokens)
	}
}

func TestCompileUsesSharedDomainTokenEstimatorForEveryArtifactBoundary(t *testing.T) {
	draft := validDraft()
	draft.Core.ContentMarkdown = "abcd招聘🙂"
	draft.Sections = []ReferenceSection{
		validSection("mixed-content", 1, "abcde面试🙂"),
	}

	compiled, err := Compile(draft)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if got, want := compiled.Core.EstimatedTokens, domaintokenbudget.EstimateConservative(compiled.Core.ContentMarkdown); got != want {
		t.Fatalf("Core.EstimatedTokens = %d, want shared estimate %d", got, want)
	}
	if got, want := compiled.Sections[0].EstimatedTokens, domaintokenbudget.EstimateConservative(compiled.Sections[0].ContentMarkdown); got != want {
		t.Fatalf("Section.EstimatedTokens = %d, want shared estimate %d", got, want)
	}
	if got, want := compiled.EstimatedTokens, domaintokenbudget.EstimateConservative(compiled.CompiledMarkdown); got != want {
		t.Fatalf("Package.EstimatedTokens = %d, want shared estimate %d", got, want)
	}
}

func TestCompileAllowsPackageAtBudgetLimit(t *testing.T) {
	draft := validDraft()
	draft.Core.ContentMarkdown = strings.Repeat("a", MaxCoreTokens*4)
	draft.Sections = makeSections(9, strings.Repeat("b", MaxSectionTokens*4))
	draft.Sections = append(draft.Sections, validSection("section-final", 10, "c"))

	for contentLength := 1; contentLength <= MaxSectionTokens*4; contentLength++ {
		draft.Sections[len(draft.Sections)-1].ContentMarkdown = strings.Repeat("c", contentLength)
		if compiledArtifactTokensForDraft(t, draft) == MaxPackageTokens {
			break
		}
	}
	if got := compiledArtifactTokensForDraft(t, draft); got != MaxPackageTokens {
		t.Fatalf("could not construct exact-limit package: EstimatedTokens = %d, want %d", got, MaxPackageTokens)
	}

	compiled, err := Compile(draft)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if compiled.EstimatedTokens != MaxPackageTokens {
		t.Fatalf("EstimatedTokens = %d, want %d", compiled.EstimatedTokens, MaxPackageTokens)
	}
}

func TestCompileEnforcesPackageBudget(t *testing.T) {
	draft := validDraft()
	draft.Core.ContentMarkdown = strings.Repeat("a", MaxCoreTokens*4)
	draft.Sections = makeSections(10, strings.Repeat("b", MaxSectionTokens*4))

	_, err := Compile(draft)
	assertCompileError(t, err, CodePackageInvalid, "package")
}

func TestCompilePackageBudgetIncludesRenderedMetadata(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*PackageDraft)
		wantField string
	}{
		{
			name:      "manifest metadata",
			wantField: "package",
			mutate: func(draft *PackageDraft) {
				draft.Manifest.Description = strings.Repeat("m", MaxPackageTokens*4)
			},
		},
		{
			name:      "output schema",
			wantField: "manifest.output_contract.schema",
			mutate: func(draft *PackageDraft) {
				schema, err := json.Marshal(map[string]any{
					"type":        "object",
					"description": strings.Repeat("s", MaxPackageTokens*4),
				})
				if err != nil {
					t.Fatalf("json.Marshal() error = %v", err)
				}
				draft.Manifest.OutputContract = OutputContract{
					Mode:   OutputModeAdvisory,
					Schema: schema,
				}
			},
		},
		{
			name:      "section title and description",
			wantField: "package",
			mutate: func(draft *PackageDraft) {
				draft.Sections = []ReferenceSection{
					{
						SectionKey:      "metadata",
						Title:           strings.Repeat("t", MaxPackageTokens*2),
						Description:     strings.Repeat("d", MaxPackageTokens*2),
						ContentMarkdown: "short content",
					},
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			draft := validDraft()
			test.mutate(&draft)
			_, err := Compile(draft)
			assertCompileError(t, err, CodePackageInvalid, test.wantField)
		})
	}
}

func TestCompileHashChangesWithNormalizedPackage(t *testing.T) {
	base, err := Compile(validDraft())
	if err != nil {
		t.Fatalf("Compile(base) error = %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*PackageDraft)
	}{
		{name: "core", mutate: func(draft *PackageDraft) { draft.Core.ContentMarkdown = "Different core" }},
		{name: "manifest", mutate: func(draft *PackageDraft) { draft.Manifest.Priority++ }},
		{
			name: "section",
			mutate: func(draft *PackageDraft) {
				draft.Sections = []ReferenceSection{validSection("details", 1, "Details")}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			draft := validDraft()
			test.mutate(&draft)
			compiled, compileErr := Compile(draft)
			if compileErr != nil {
				t.Fatalf("Compile() error = %v", compileErr)
			}
			if compiled.CompiledHash == base.CompiledHash {
				t.Fatal("CompiledHash did not change")
			}
		})
	}
}

func TestErrorCodeReturnsEmptyForUnrelatedError(t *testing.T) {
	if got := ErrorCode(errors.New("other")); got != "" {
		t.Fatalf("ErrorCode() = %q", got)
	}
}

func validDraft() PackageDraft {
	return PackageDraft{
		Manifest: Manifest{
			SchemaVersion:      SchemaVersion,
			SkillName:          "candidate-screening",
			DisplayName:        "Candidate Screening",
			Description:        "Screen candidates consistently.",
			AgentType:          "hr_recruiting_agent",
			Category:           "screening",
			Scenario:           "candidate_screening",
			Priority:           100,
			RiskLevel:          RiskLevelMedium,
			Composition:        Composition{Role: CompositionRolePrimary},
			OutputContract:     OutputContract{Mode: OutputModeNone},
			TriggerKeywords:    []string{},
			SemanticTags:       []string{},
			EvaluationCriteria: []string{},
		},
		Core:     Core{ContentMarkdown: "Follow the screening policy."},
		Sections: []ReferenceSection{},
	}
}

func validSection(key string, ordinal int, content string) ReferenceSection {
	return ReferenceSection{
		SectionKey:      key,
		Title:           strings.ToUpper(key),
		ContentMarkdown: content,
		Ordinal:         ordinal,
	}
}

func makeSections(count int, content string) []ReferenceSection {
	sections := make([]ReferenceSection, 0, count)
	for i := 0; i < count; i++ {
		sections = append(sections, validSection(fmt.Sprintf("section-%02d", i), i, content))
	}
	return sections
}

func sectionKeys(sections []CompiledSection) []string {
	keys := make([]string, 0, len(sections))
	for _, section := range sections {
		keys = append(keys, section.SectionKey)
	}
	return keys
}

func compiledArtifactTokensForDraft(t *testing.T, draft PackageDraft) int {
	t.Helper()
	manifest, err := normalizeManifest(draft.Manifest)
	if err != nil {
		t.Fatalf("normalizeManifest() error = %v", err)
	}
	sections, _, err := normalizeSections(draft.Sections)
	if err != nil {
		t.Fatalf("normalizeSections() error = %v", err)
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	core := normalizeMarkdown(draft.Core.ContentMarkdown)
	return estimateCompiledArtifactTokens(
		renderCompiledMarkdown(manifest, string(manifestJSON), core, sections),
	)
}

func assertCompileError(t *testing.T, err error, wantCode, wantField string) {
	t.Helper()
	if err == nil {
		t.Fatal("Compile() error = nil")
	}
	if got := ErrorCode(err); got != wantCode {
		t.Fatalf("ErrorCode() = %q, want %q; err = %v", got, wantCode, err)
	}
	var target *CompileError
	if !errors.As(err, &target) {
		t.Fatalf("error type = %T, want *CompileError", err)
	}
	if target.Field != wantField {
		t.Fatalf("CompileError.Field = %q, want %q", target.Field, wantField)
	}
}
