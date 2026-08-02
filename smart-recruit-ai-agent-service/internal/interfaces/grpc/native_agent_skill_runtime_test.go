package grpc

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"

	"smart-recruit-ai-agent-service/internal/application/contextbudget"
	domainagentskill "smart-recruit-ai-agent-service/internal/domain/agentskill"
	embeddinginfra "smart-recruit-ai-agent-service/internal/infrastructure/provider"
	"smart-recruit-proto/recruitment/pb"
)

func TestSelectHRRuntimeAgentSkillPackagesRequiresExactReleaseVersion(t *testing.T) {
	store := newFakeAIStore()
	store.agentSkillVersionDocs = []embeddinginfra.AgentSkillVersionEmbeddingDocument{
		runtimeSkillVersionDocument(101, 1, domainagentskill.CompositionRolePrimary, domainagentskill.RiskLevelLow, "resume screening core"),
		runtimeSkillVersionDocument(102, 1, domainagentskill.CompositionRolePrimary, domainagentskill.RiskLevelLow, "unreleased draft core"),
	}
	service := newNativeAIService(store, nil, nil, nil, nil)
	model := runtimeSkillModel([]int64{101}, CapabilitySkillRuntimePolicy{})

	selected, evidence, confirmationRequired, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
		context.Background(),
		&pb.ChatRequest{Message: "resume screening", AgentSkillVersionIds: []int64{102}},
		nil,
		model,
		true,
	)

	if len(selected) != 0 || len(evidence) != 0 || confirmationRequired {
		t.Fatalf("selection = %#v evidence=%#v confirmation=%v, want fail closed", selected, evidence, confirmationRequired)
	}
	if len(governanceErrors) != 1 || governanceErrors[0].Code != "outside_capability_release" || governanceErrors[0].ResourceID != 102 {
		t.Fatalf("governance errors = %#v", governanceErrors)
	}
}

func TestSelectHRRuntimeAgentSkillPackagesRejectsInvalidOrDisabledManualSelection(t *testing.T) {
	t.Run("duplicate exact version", func(t *testing.T) {
		store := newFakeAIStore()
		store.agentSkillVersionDocs = []embeddinginfra.AgentSkillVersionEmbeddingDocument{
			runtimeSkillVersionDocument(101, 1, domainagentskill.CompositionRolePrimary, domainagentskill.RiskLevelLow, "resume"),
		}
		service := newNativeAIService(store, nil, nil, nil, nil)
		_, _, _, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
			context.Background(),
			&pb.ChatRequest{Message: "resume", AgentSkillVersionIds: []int64{101, 101}},
			nil,
			runtimeSkillModel([]int64{101}, CapabilitySkillRuntimePolicy{}),
			true,
		)
		if len(governanceErrors) != 1 || governanceErrors[0].Code != "manual_selection_invalid" {
			t.Fatalf("governance errors = %#v", governanceErrors)
		}
	})

	t.Run("registry disables manual invocation", func(t *testing.T) {
		store := newFakeAIStore()
		document := runtimeSkillVersionDocument(101, 1, domainagentskill.CompositionRolePrimary, domainagentskill.RiskLevelLow, "resume")
		document.ManualInvocable = false
		store.agentSkillVersionDocs = []embeddinginfra.AgentSkillVersionEmbeddingDocument{document}
		service := newNativeAIService(store, nil, nil, nil, nil)
		_, evidence, _, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
			context.Background(),
			&pb.ChatRequest{Message: "resume", AgentSkillVersionIds: []int64{101}},
			nil,
			runtimeSkillModel([]int64{101}, CapabilitySkillRuntimePolicy{}),
			true,
		)
		if len(governanceErrors) != 1 || governanceErrors[0].Code != "manual_invocation_disabled" ||
			len(evidence) != 1 || evidence[0].GetDecisionReason() != "manual_invocation_disabled" {
			t.Fatalf("evidence=%#v governance errors=%#v", evidence, governanceErrors)
		}
	})
}

func TestSelectHRRuntimeAgentSkillPackagesComposesPrimaryThenSupporting(t *testing.T) {
	store := newFakeAIStore()
	store.agentSkillVersionDocs = []embeddinginfra.AgentSkillVersionEmbeddingDocument{
		runtimeSkillVersionDocument(101, 1, domainagentskill.CompositionRolePrimary, domainagentskill.RiskLevelLow, "resume screening primary core"),
		runtimeSkillVersionDocument(102, 2, domainagentskill.CompositionRoleSupporting, domainagentskill.RiskLevelMedium, "resume screening supporting core"),
	}
	service := newNativeAIService(store, nil, nil, nil, nil)
	model := runtimeSkillModel([]int64{101, 102}, CapabilitySkillRuntimePolicy{})

	selected, evidence, confirmationRequired, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
		context.Background(),
		&pb.ChatRequest{Message: "resume screening"},
		nil,
		model,
		true,
	)

	if len(governanceErrors) != 0 || confirmationRequired {
		t.Fatalf("errors=%#v confirmation=%v", governanceErrors, confirmationRequired)
	}
	if len(selected) != 2 || selected[0].VersionID != 101 || selected[1].VersionID != 102 ||
		selected[0].CompositionRole != domainagentskill.CompositionRolePrimary ||
		selected[1].CompositionRole != domainagentskill.CompositionRoleSupporting {
		t.Fatalf("selected = %#v", selected)
	}
	if len(evidence) != 2 || !evidence[0].GetIncluded() || !evidence[1].GetIncluded() {
		t.Fatalf("evidence = %#v, want both included", evidence)
	}
	prompt := renderHRProviderPrompt(&pb.ChatRequest{Message: "resume screening"}, nil, ChatMessageRow{}, nil, hrRuntimeGovernanceContext{SelectedAgentSkills: selected})
	primaryIndex := strings.Index(prompt, "primary core")
	supportingIndex := strings.Index(prompt, "supporting core")
	if primaryIndex < 0 || supportingIndex <= primaryIndex {
		t.Fatalf("prompt did not inject Primary before Supporting: %s", prompt)
	}
}

func TestSelectHRRuntimeAgentSkillPackagesComposesOnlyCompatibleScenario(t *testing.T) {
	primary := runtimeSkillVersionDocument(
		101,
		1,
		domainagentskill.CompositionRolePrimary,
		domainagentskill.RiskLevelLow,
		"resume screening primary",
	)
	primary.Manifest.Scenario = " Resume Screening "
	incompatible := runtimeSkillVersionDocument(
		102,
		2,
		domainagentskill.CompositionRoleSupporting,
		domainagentskill.RiskLevelLow,
		"resume screening first supporting",
	)
	incompatible.Manifest.Scenario = "interview"
	compatible := runtimeSkillVersionDocument(
		103,
		3,
		domainagentskill.CompositionRoleSupporting,
		domainagentskill.RiskLevelLow,
		"resume screening compatible supporting",
	)
	compatible.Manifest.Scenario = "resume screening"
	store := newFakeAIStore()
	store.agentSkillVersionDocs = []embeddinginfra.AgentSkillVersionEmbeddingDocument{
		primary,
		incompatible,
		compatible,
	}
	service := newNativeAIService(store, nil, nil, nil, nil)

	selected, evidence, confirmationRequired, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
		context.Background(),
		&pb.ChatRequest{Message: "resume screening"},
		nil,
		runtimeSkillModel([]int64{101, 102, 103}, CapabilitySkillRuntimePolicy{}),
		true,
	)

	if len(governanceErrors) != 0 || confirmationRequired {
		t.Fatalf("errors=%#v confirmation=%v", governanceErrors, confirmationRequired)
	}
	if len(selected) != 2 || selected[0].VersionID != 101 || selected[1].VersionID != 103 {
		t.Fatalf("selected=%#v, want compatible Primary and later Supporting", selected)
	}
	if len(evidence) != 3 {
		t.Fatalf("evidence=%#v, want all composition decisions", evidence)
	}
	if evidence[0].GetVersionId() != 101 || !evidence[0].GetIncluded() ||
		evidence[1].GetVersionId() != 102 || evidence[1].GetIncluded() ||
		evidence[1].GetDecisionReason() != "composition_scenario_mismatch" ||
		evidence[2].GetVersionId() != 103 || !evidence[2].GetIncluded() {
		t.Fatalf("evidence=%#v", evidence)
	}
}

func TestSelectHRRuntimeAgentSkillPackagesRejectsManualIncompatibleCompositionDeterministically(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*embeddinginfra.AgentSkillVersionEmbeddingDocument)
		wantReason string
	}{
		{
			name: "scenario mismatch",
			mutate: func(document *embeddinginfra.AgentSkillVersionEmbeddingDocument) {
				document.Manifest.Scenario = "interview"
			},
			wantReason: "composition_scenario_mismatch",
		},
		{
			name: "agent type mismatch",
			mutate: func(document *embeddinginfra.AgentSkillVersionEmbeddingDocument) {
				document.Manifest.AgentType = "candidate_assistant"
			},
			wantReason: "agent_type_mismatch",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primary := runtimeSkillVersionDocument(
				101,
				1,
				domainagentskill.CompositionRolePrimary,
				domainagentskill.RiskLevelLow,
				"resume primary",
			)
			supporting := runtimeSkillVersionDocument(
				102,
				2,
				domainagentskill.CompositionRoleSupporting,
				domainagentskill.RiskLevelLow,
				"resume supporting",
			)
			tt.mutate(&supporting)
			store := newFakeAIStore()
			store.agentSkillVersionDocs = []embeddinginfra.AgentSkillVersionEmbeddingDocument{supporting, primary}
			service := newNativeAIService(store, nil, nil, nil, nil)

			selected, evidence, confirmationRequired, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
				context.Background(),
				&pb.ChatRequest{
					Message:              "resume",
					AgentSkillVersionIds: []int64{102, 101},
				},
				nil,
				runtimeSkillModel([]int64{101, 102}, CapabilitySkillRuntimePolicy{}),
				true,
			)

			if len(selected) != 0 || confirmationRequired ||
				len(governanceErrors) != 1 || governanceErrors[0].Code != tt.wantReason ||
				governanceErrors[0].ResourceID != 102 {
				t.Fatalf(
					"selected=%#v evidence=%#v errors=%#v confirmation=%v",
					selected,
					evidence,
					governanceErrors,
					confirmationRequired,
				)
			}
			if len(evidence) != 1 || evidence[0].GetVersionId() != 102 ||
				evidence[0].GetIncluded() || evidence[0].GetDecisionReason() != tt.wantReason {
				t.Fatalf("evidence=%#v", evidence)
			}
		})
	}
}

func TestSelectHRRuntimeAgentSkillPackagesNeverLoadsSupportingAlone(t *testing.T) {
	store := newFakeAIStore()
	store.agentSkillVersionDocs = []embeddinginfra.AgentSkillVersionEmbeddingDocument{
		runtimeSkillVersionDocument(102, 2, domainagentskill.CompositionRoleSupporting, domainagentskill.RiskLevelLow, "resume supporting core"),
	}
	service := newNativeAIService(store, nil, nil, nil, nil)
	model := runtimeSkillModel([]int64{102}, CapabilitySkillRuntimePolicy{})

	selected, evidence, confirmationRequired, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
		context.Background(),
		&pb.ChatRequest{Message: "resume", AgentSkillVersionIds: []int64{102}},
		nil,
		model,
		true,
	)

	if len(selected) != 0 || confirmationRequired {
		t.Fatalf("selected=%#v confirmation=%v", selected, confirmationRequired)
	}
	if len(governanceErrors) != 1 || governanceErrors[0].Code != "supporting_requires_primary" {
		t.Fatalf("errors = %#v", governanceErrors)
	}
	if len(evidence) != 1 || evidence[0].GetDecisionReason() != "supporting_requires_primary" {
		t.Fatalf("evidence = %#v", evidence)
	}
}

func TestSelectHRRuntimeAgentSkillPackagesScopesSectionsAndDropsWholeSectionAtBudget(t *testing.T) {
	store := newFakeAIStore()
	store.agentSkillVersionDocs = []embeddinginfra.AgentSkillVersionEmbeddingDocument{
		runtimeSkillVersionDocument(101, 1, domainagentskill.CompositionRolePrimary, domainagentskill.RiskLevelLow, "resume core"),
		runtimeSkillVersionDocument(999, 9, domainagentskill.CompositionRolePrimary, domainagentskill.RiskLevelLow, "foreign core"),
	}
	store.agentSkillSectionDocs = []embeddinginfra.AgentSkillSectionEmbeddingDocument{
		{ID: 1001, SkillID: 1, VersionID: 101, SectionKey: "small", Title: "Resume rubric", ContentMarkdown: "resume rubric short", TriggerTerms: []string{"resume"}, Priority: 10},
		{ID: 1002, SkillID: 1, VersionID: 101, SectionKey: "large", Title: "Resume details", ContentMarkdown: strings.Repeat("resume ", 80), TriggerTerms: []string{"resume"}, Priority: 5},
		{ID: 9001, SkillID: 9, VersionID: 999, SectionKey: "foreign", Title: "Resume foreign", ContentMarkdown: "PRIVATE_FOREIGN_SECTION", TriggerTerms: []string{"resume"}, Priority: 100},
	}
	for i := range store.agentSkillSectionDocs {
		document := &store.agentSkillSectionDocs[i]
		document.ContentHash = sha256Hex(document.ContentMarkdown)
		document.EstimatedTokens = contextbudget.EstimateTokensConservative(document.ContentMarkdown)
	}
	service := newNativeAIService(store, nil, nil, nil, nil)
	model := runtimeSkillModel([]int64{101}, CapabilitySkillRuntimePolicy{MaxSkillTokens: 12, MaxInputRatio: 0.01, MaxSkills: 1})

	selected, evidence, confirmationRequired, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
		context.Background(),
		&pb.ChatRequest{Message: "resume"},
		nil,
		model,
		true,
	)

	if len(governanceErrors) != 0 || confirmationRequired || len(selected) != 1 {
		t.Fatalf("selected=%#v errors=%#v confirmation=%v", selected, governanceErrors, confirmationRequired)
	}
	if len(selected[0].Sections) != 2 {
		t.Fatalf("sections = %#v, want only selected version sections", selected[0].Sections)
	}
	var included, dropped bool
	for _, section := range selected[0].Sections {
		switch section.DecisionReason {
		case "section_included":
			included = section.Included && section.ContentMarkdown != ""
		case "section_budget_exceeded":
			dropped = !section.Included && section.ContentMarkdown == ""
		}
		if section.ID == 9001 {
			t.Fatal("section outside selected version leaked into runtime")
		}
	}
	if !included || !dropped {
		t.Fatalf("sections = %#v, want complete inclusion and budget drop", selected[0].Sections)
	}
	rawEvidence, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rawEvidence), "resume rubric short") || strings.Contains(string(rawEvidence), "PRIVATE_FOREIGN_SECTION") {
		t.Fatalf("runtime evidence leaked Skill content: %s", rawEvidence)
	}
	prompt := renderHRProviderPrompt(&pb.ChatRequest{Message: "resume"}, nil, ChatMessageRow{}, nil, hrRuntimeGovernanceContext{SelectedAgentSkills: selected})
	if !strings.Contains(prompt, "resume rubric short") || strings.Contains(prompt, strings.Repeat("resume ", 20)) ||
		strings.Contains(prompt, "PRIVATE_FOREIGN_SECTION") {
		t.Fatalf("prompt violated complete-section budget/scope: %s", prompt)
	}
}

func TestSelectHRRuntimeAgentSkillPackagesFailsClosedOnImmutablePackageTampering(t *testing.T) {
	document := runtimeSkillVersionDocument(
		101,
		1,
		domainagentskill.CompositionRolePrimary,
		domainagentskill.RiskLevelLow,
		"resume screening core",
	)
	section := embeddinginfra.AgentSkillSectionEmbeddingDocument{
		ID:              1001,
		SkillID:         1,
		VersionID:       101,
		Version:         "2.0.0",
		SectionKey:      "resume-rubric",
		Title:           "Resume rubric",
		ContentMarkdown: "resume scoring detail",
		TriggerTerms:    []string{"resume"},
		Priority:        10,
	}
	validPackage := runtimeSkillPackage(t, document, []embeddinginfra.AgentSkillSectionEmbeddingDocument{section})

	t.Run("equivalent reordered and whitespace manifest JSON is accepted", func(t *testing.T) {
		var manifestObject map[string]any
		if err := json.Unmarshal([]byte(validPackage.ManifestJSON), &manifestObject); err != nil {
			t.Fatalf("decode valid manifest JSON: %v", err)
		}
		reordered, err := json.MarshalIndent(manifestObject, "", "  ")
		if err != nil {
			t.Fatalf("encode reordered manifest JSON: %v", err)
		}
		if string(reordered) == validPackage.ManifestJSON {
			t.Fatal("test setup did not change the manifest JSON representation")
		}
		represented := validPackage
		represented.ManifestJSON = string(reordered)
		store := newFakeAIStore()
		store.agentSkillPackages = []embeddinginfra.AgentSkillRuntimePackage{represented}
		service := newNativeAIService(store, nil, nil, nil, nil)

		selected, evidence, confirmationRequired, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
			context.Background(),
			&pb.ChatRequest{Message: "resume screening"},
			nil,
			runtimeSkillModel([]int64{101}, CapabilitySkillRuntimePolicy{}),
			true,
		)
		if len(selected) != 1 || selected[0].VersionID != 101 || confirmationRequired || len(governanceErrors) != 0 ||
			len(evidence) != 1 || !evidence[0].GetIncluded() {
			t.Fatalf("selected=%#v evidence=%#v errors=%#v confirmation=%v", selected, evidence, governanceErrors, confirmationRequired)
		}
	})

	t.Run("nil and empty section sets are semantically equivalent", func(t *testing.T) {
		represented := validPackage
		represented.Sections = append([]embeddinginfra.AgentSkillRuntimeSection(nil), validPackage.Sections...)
		represented.Sections[0].SemanticTags = nil
		represented.Sections[0].PlannerIntents = nil

		store := newFakeAIStore()
		store.agentSkillPackages = []embeddinginfra.AgentSkillRuntimePackage{represented}
		service := newNativeAIService(store, nil, nil, nil, nil)

		selected, evidence, confirmationRequired, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
			context.Background(),
			&pb.ChatRequest{Message: "resume screening"},
			nil,
			runtimeSkillModel([]int64{101}, CapabilitySkillRuntimePolicy{}),
			true,
		)
		if len(selected) != 1 || selected[0].VersionID != 101 || confirmationRequired || len(governanceErrors) != 0 ||
			len(evidence) != 1 || !evidence[0].GetIncluded() {
			t.Fatalf("selected=%#v evidence=%#v errors=%#v confirmation=%v", selected, evidence, governanceErrors, confirmationRequired)
		}
	})

	tests := []struct {
		name   string
		mutate func(*embeddinginfra.AgentSkillRuntimePackage)
	}{
		{
			name: "manifest duplicate known field",
			mutate: func(runtimePackage *embeddinginfra.AgentSkillRuntimePackage) {
				runtimePackage.ManifestJSON = strings.Replace(
					runtimePackage.ManifestJSON,
					`"display_name":`,
					`"display_name":"shadow","display_name":`,
					1,
				)
			},
		},
		{
			name: "manifest duplicate nested known field",
			mutate: func(runtimePackage *embeddinginfra.AgentSkillRuntimePackage) {
				runtimePackage.ManifestJSON = strings.Replace(
					runtimePackage.ManifestJSON,
					`"role":"primary"`,
					`"role":"supporting","role":"primary"`,
					1,
				)
			},
		},
		{
			name: "manifest duplicate unknown field",
			mutate: func(runtimePackage *embeddinginfra.AgentSkillRuntimePackage) {
				runtimePackage.ManifestJSON = strings.TrimSuffix(runtimePackage.ManifestJSON, "}") +
					`,"unknown_runtime_field":true,"unknown_runtime_field":false}`
			},
		},
		{
			name: "manifest unknown field",
			mutate: func(runtimePackage *embeddinginfra.AgentSkillRuntimePackage) {
				runtimePackage.ManifestJSON = strings.TrimSuffix(runtimePackage.ManifestJSON, "}") + `,"unknown_runtime_field":true}`
			},
		},
		{
			name: "manifest semantic content",
			mutate: func(runtimePackage *embeddinginfra.AgentSkillRuntimePackage) {
				runtimePackage.ManifestJSON = strings.Replace(
					runtimePackage.ManifestJSON,
					`"display_name":"Skill"`,
					`"display_name":"Tampered Skill"`,
					1,
				)
			},
		},
		{
			name: "core",
			mutate: func(runtimePackage *embeddinginfra.AgentSkillRuntimePackage) {
				runtimePackage.CoreMarkdown += " tampered"
			},
		},
		{
			name: "section",
			mutate: func(runtimePackage *embeddinginfra.AgentSkillRuntimePackage) {
				runtimePackage.Sections[0].ContentMarkdown += " tampered"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tampered := validPackage
			tampered.Sections = append([]embeddinginfra.AgentSkillRuntimeSection(nil), validPackage.Sections...)
			tt.mutate(&tampered)

			base := newFakeAIStore()
			base.agentConfigs = []*pb.AgentConfigInfo{{
				Id: 10, Name: "hr", AgentType: hrRecruitingAgentType, PromptTemplateId: 20, IsDefault: true, IsEnabled: true,
			}}
			base.promptByID[20] = &pb.PromptTemplateInfo{
				Id: 20, AgentType: hrRecruitingAgentType, PromptRole: hrRuntimePromptRoleSystem, IsActive: true, Content: "system",
			}
			base.agentSkillPackages = []embeddinginfra.AgentSkillRuntimePackage{tampered}
			store := &releasedSkillFakeStore{
				fakeAIStore: base,
				resolution:  runtimeSkillModel([]int64{101}, CapabilitySkillRuntimePolicy{}),
			}
			service := newNativeAIService(store, &fakeChatProvider{reply: "must not run"}, nil, nil, nil)

			selected, evidence, confirmationRequired, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
				context.Background(),
				&pb.ChatRequest{Message: "resume screening"},
				nil,
				store.resolution,
				true,
			)
			if len(selected) != 0 || confirmationRequired ||
				len(evidence) != 1 || evidence[0].GetDecisionReason() != "package_integrity_failed" ||
				len(governanceErrors) != 1 || governanceErrors[0].Code != "package_integrity_failed" {
				t.Fatalf("selected=%#v evidence=%#v errors=%#v confirmation=%v", selected, evidence, governanceErrors, confirmationRequired)
			}

			provider := &fakeChatProvider{reply: "must not run"}
			service = newNativeAIService(store, provider, nil, nil, nil)
			service.skillPackageV2Enabled = true
			_, _ = service.Chat(context.Background(), &pb.ChatRequest{HrId: 77, Message: "resume screening"})
			if provider.calls != 0 {
				t.Fatalf("provider calls = %d, want zero after package integrity failure", provider.calls)
			}
		})
	}
}

func TestSelectHRRuntimeAgentSkillPackagesCanonicalizesNestedOutputSchemaJSON(t *testing.T) {
	for _, tt := range []struct {
		name     string
		mode     domainagentskill.OutputMode
		schemaID string
	}{
		{name: "advisory", mode: domainagentskill.OutputModeAdvisory},
		{name: "strict", mode: domainagentskill.OutputModeStrict, schemaID: "candidate-score-v1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			document := runtimeSkillVersionDocument(
				101,
				1,
				domainagentskill.CompositionRolePrimary,
				domainagentskill.RiskLevelLow,
				"resume screening core",
			)
			document.Manifest.OutputContract = domainagentskill.OutputContract{
				Mode:     tt.mode,
				SchemaID: tt.schemaID,
				Schema: json.RawMessage(`{
					"required": ["score"],
					"properties": {
						"score": {
							"description": "Candidate score",
							"type": "number"
						}
					},
					"type": "object"
				}`),
			}
			runtimePackage := runtimeSkillPackage(t, document, nil)
			runtimePackage.ManifestJSON = reformatRuntimeManifestJSON(t, runtimePackage.ManifestJSON, nil)
			store := newFakeAIStore()
			store.agentSkillPackages = []embeddinginfra.AgentSkillRuntimePackage{runtimePackage}
			service := newNativeAIService(store, nil, nil, nil, nil)

			selected, evidence, confirmationRequired, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
				context.Background(),
				&pb.ChatRequest{Message: "resume screening"},
				nil,
				runtimeSkillModel([]int64{101}, CapabilitySkillRuntimePolicy{}),
				true,
			)
			if tt.mode == domainagentskill.OutputModeStrict {
				if len(selected) != 0 || confirmationRequired || len(governanceErrors) != 1 ||
					governanceErrors[0].Code != "strict_output_contract_unsupported" ||
					len(evidence) != 1 || evidence[0].GetDecisionReason() != "strict_output_contract_unsupported" {
					t.Fatalf("strict selected=%#v evidence=%#v errors=%#v confirmation=%v", selected, evidence, governanceErrors, confirmationRequired)
				}
				return
			}
			if len(selected) != 1 || selected[0].VersionID != 101 || confirmationRequired || len(governanceErrors) != 0 ||
				len(evidence) != 1 || !evidence[0].GetIncluded() {
				t.Fatalf("selected=%#v evidence=%#v errors=%#v confirmation=%v", selected, evidence, governanceErrors, confirmationRequired)
			}
		})
	}
}

func TestSelectHRRuntimeAgentSkillPackagesRejectsNestedOutputSchemaSemanticTampering(t *testing.T) {
	document := runtimeSkillVersionDocument(
		101,
		1,
		domainagentskill.CompositionRolePrimary,
		domainagentskill.RiskLevelLow,
		"resume screening core",
	)
	document.Manifest.OutputContract = domainagentskill.OutputContract{
		Mode: domainagentskill.OutputModeAdvisory,
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"score": {"type": "number"}
			}
		}`),
	}
	tampered := runtimeSkillPackage(t, document, nil)
	tampered.ManifestJSON = reformatRuntimeManifestJSON(
		t,
		tampered.ManifestJSON,
		func(manifest map[string]any) {
			outputContract, ok := manifest["output_contract"].(map[string]any)
			if !ok {
				t.Fatal("output_contract is not an object")
			}
			schemaObject, ok := outputContract["schema"].(map[string]any)
			if !ok {
				t.Fatal("output_contract.schema is not an object")
			}
			properties, ok := schemaObject["properties"].(map[string]any)
			if !ok {
				t.Fatal("schema.properties is not an object")
			}
			score, ok := properties["score"].(map[string]any)
			if !ok {
				t.Fatal("schema.properties.score is not an object")
			}
			score["type"] = "string"
		},
	)

	base := newFakeAIStore()
	base.agentConfigs = []*pb.AgentConfigInfo{{
		Id: 10, Name: "hr", AgentType: hrRecruitingAgentType, PromptTemplateId: 20, IsDefault: true, IsEnabled: true,
	}}
	base.promptByID[20] = &pb.PromptTemplateInfo{
		Id: 20, AgentType: hrRecruitingAgentType, PromptRole: hrRuntimePromptRoleSystem, IsActive: true, Content: "system",
	}
	base.agentSkillPackages = []embeddinginfra.AgentSkillRuntimePackage{tampered}
	store := &releasedSkillFakeStore{
		fakeAIStore: base,
		resolution:  runtimeSkillModel([]int64{101}, CapabilitySkillRuntimePolicy{}),
	}
	provider := &fakeChatProvider{reply: "must not run"}
	service := newNativeAIService(store, provider, nil, nil, nil)
	service.skillPackageV2Enabled = true

	selected, evidence, confirmationRequired, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
		context.Background(),
		&pb.ChatRequest{Message: "resume screening"},
		nil,
		store.resolution,
		true,
	)
	if len(selected) != 0 || confirmationRequired ||
		len(evidence) != 1 || evidence[0].GetDecisionReason() != "package_integrity_failed" ||
		len(governanceErrors) != 1 || governanceErrors[0].Code != "package_integrity_failed" {
		t.Fatalf("selected=%#v evidence=%#v errors=%#v confirmation=%v", selected, evidence, governanceErrors, confirmationRequired)
	}
	_, _ = service.Chat(context.Background(), &pb.ChatRequest{HrId: 77, Message: "resume screening"})
	if provider.calls != 0 {
		t.Fatalf("provider calls = %d, want zero after nested schema semantic tampering", provider.calls)
	}
}

func TestSelectHRRuntimeAgentSkillPackagesRecordsSupportingBlockedByPrimaryBudget(t *testing.T) {
	store := newFakeAIStore()
	store.agentSkillVersionDocs = []embeddinginfra.AgentSkillVersionEmbeddingDocument{
		runtimeSkillVersionDocument(101, 1, domainagentskill.CompositionRolePrimary, domainagentskill.RiskLevelLow, "resume primary core exceeds tiny runtime budget"),
		runtimeSkillVersionDocument(102, 2, domainagentskill.CompositionRoleSupporting, domainagentskill.RiskLevelLow, "resume supporting core"),
	}
	service := newNativeAIService(store, nil, nil, nil, nil)

	selected, evidence, confirmationRequired, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
		context.Background(),
		&pb.ChatRequest{Message: "resume"},
		nil,
		runtimeSkillModel([]int64{101, 102}, CapabilitySkillRuntimePolicy{MaxSkillTokens: 1, MaxInputRatio: 0.01}),
		true,
	)
	if len(selected) != 0 || confirmationRequired || len(governanceErrors) != 0 {
		t.Fatalf("selected=%#v errors=%#v confirmation=%v", selected, governanceErrors, confirmationRequired)
	}
	if len(evidence) != 2 {
		t.Fatalf("evidence = %#v, want primary and supporting drop evidence", evidence)
	}
	if evidence[0].GetVersionId() != 101 || evidence[0].GetIncluded() ||
		evidence[0].GetDecisionReason() != "core_budget_exceeded" {
		t.Fatalf("primary evidence = %#v, want core_budget_exceeded", evidence[0])
	}
	if evidence[1].GetVersionId() != 102 || evidence[1].GetIncluded() ||
		evidence[1].GetDecisionReason() != "blocked_by_primary_budget" {
		t.Fatalf("supporting evidence = %#v, want blocked_by_primary_budget", evidence[1])
	}
}

func TestSelectHRRuntimeAgentSkillPackagesUsesLexicalMetadataFallback(t *testing.T) {
	store := newFakeAIStore()
	relevant := runtimeSkillVersionDocument(101, 1, domainagentskill.CompositionRolePrimary, domainagentskill.RiskLevelLow, "candidate resume screening rubric")
	relevant.Manifest.SkillName = "resume-screening"
	relevant.Manifest.DisplayName = "Resume Screening"
	relevant.Manifest.Category = "screening"
	relevant.Manifest.Scenario = "resume-screening"
	relevant.Manifest.TriggerKeywords = []string{"resume"}
	unrelated := runtimeSkillVersionDocument(102, 2, domainagentskill.CompositionRolePrimary, domainagentskill.RiskLevelLow, "offer negotiation")
	unrelated.Manifest.SkillName = "offer-negotiation"
	unrelated.Manifest.DisplayName = "Offer Negotiation"
	unrelated.Manifest.Category = "offer"
	unrelated.Manifest.Scenario = "offer-negotiation"
	unrelated.Manifest.TriggerKeywords = nil
	unrelated.Manifest.SemanticTags = nil
	unrelated.Manifest.Priority = 1000
	store.agentSkillVersionDocs = []embeddinginfra.AgentSkillVersionEmbeddingDocument{unrelated, relevant}
	service := newNativeAIService(store, nil, nil, nil, nil)
	model := runtimeSkillModel([]int64{101, 102}, CapabilitySkillRuntimePolicy{})

	selected, evidence, _, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
		context.Background(),
		&pb.ChatRequest{Message: "please screen this resume"},
		nil,
		model,
		true,
	)

	if len(governanceErrors) != 0 || len(selected) != 1 || selected[0].VersionID != 101 || selected[0].RelevanceMode != string(domainagentskill.RelevanceModeLexicalMetadata) {
		t.Fatalf("selected=%#v evidence=%#v errors=%#v", selected, evidence, governanceErrors)
	}
	var rejectedEvidence *pb.AgentSkillRuntimeEvidence
	for _, item := range evidence {
		if item.GetVersionId() == 102 {
			rejectedEvidence = item
			break
		}
	}
	if rejectedEvidence == nil || rejectedEvidence.GetIncluded() ||
		rejectedEvidence.GetDecisionReason() != "below_relevance_gate" ||
		rejectedEvidence.GetRelevanceScore() >= domainagentskill.RelevanceGate {
		t.Fatalf("unrelated Skill evidence = %#v, want below_relevance_gate with score", rejectedEvidence)
	}
}

func TestSelectHRRuntimeAgentSkillPackagesRecallsChineseCandidateScreeningRequest(t *testing.T) {
	store := newFakeAIStore()
	screening := runtimeSkillVersionDocument(
		101,
		1,
		domainagentskill.CompositionRolePrimary,
		domainagentskill.RiskLevelLow,
		"评估候选人与岗位要求的匹配情况，并区分满足项、缺口和缺失证据。",
	)
	screening.Manifest.SkillName = "candidate-screening-method"
	screening.Manifest.DisplayName = "候选人筛选方法"
	screening.Manifest.Description = "根据岗位要求与候选人信息评估匹配情况"
	screening.Manifest.Category = "screening"
	screening.Manifest.Scenario = "candidate-screening"
	screening.Manifest.TriggerKeywords = []string{"候选人筛选", "岗位匹配", "Java", "Spring Boot", "MySQL", "微服务"}
	screening.Manifest.SemanticTags = []string{"招聘", "候选人评估"}
	store.agentSkillVersionDocs = []embeddinginfra.AgentSkillVersionEmbeddingDocument{screening}
	service := newNativeAIService(store, nil, nil, nil, nil)

	query := `我要筛选一名 Java 后端候选人。

岗位要求：
- 5 年以上 Java 经验
- 熟悉 Spring Boot 和 MySQL
- 有微服务架构经验

候选人信息：
- 4 年 Java 经验
- 熟悉 Spring Boot 和 PostgreSQL
- 参与过两个微服务项目
- 没有提供 MySQL 项目证据

请评估该候选人与岗位的匹配情况。`
	selected, evidence, confirmationRequired, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
		context.Background(),
		&pb.ChatRequest{Message: query},
		nil,
		runtimeSkillModel([]int64{101}, CapabilitySkillRuntimePolicy{}),
		true,
	)
	if len(governanceErrors) != 0 || confirmationRequired || len(selected) != 1 ||
		selected[0].VersionID != 101 || selected[0].RelevanceMode != string(domainagentskill.RelevanceModeLexicalMetadata) {
		t.Fatalf("selected=%#v evidence=%#v errors=%#v confirmation=%v", selected, evidence, governanceErrors, confirmationRequired)
	}
	if len(evidence) != 1 || !evidence[0].GetIncluded() ||
		evidence[0].GetRelevanceScore() < domainagentskill.RelevanceGate {
		t.Fatalf("runtime evidence=%#v, want included Chinese screening Skill above gate", evidence)
	}
}

func TestSelectHRRuntimeAgentSkillPackagesFailsClosedForHighAndCriticalRisk(t *testing.T) {
	for _, tt := range []struct {
		name   string
		risk   domainagentskill.RiskLevel
		manual bool
	}{
		{name: "high auto", risk: domainagentskill.RiskLevelHigh},
		{name: "high manual", risk: domainagentskill.RiskLevelHigh, manual: true},
		{name: "critical manual", risk: domainagentskill.RiskLevelCritical, manual: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeAIStore()
			store.agentSkillVersionDocs = []embeddinginfra.AgentSkillVersionEmbeddingDocument{
				runtimeSkillVersionDocument(101, 1, domainagentskill.CompositionRolePrimary, tt.risk, "resume screening core"),
			}
			service := newNativeAIService(store, nil, nil, nil, nil)
			request := &pb.ChatRequest{Message: "resume screening"}
			if tt.manual {
				request.AgentSkillVersionIds = []int64{101}
			}
			selected, evidence, confirmationRequired, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
				context.Background(), request, nil, runtimeSkillModel([]int64{101}, CapabilitySkillRuntimePolicy{}), true,
			)
			if len(selected) != 0 || len(governanceErrors) != 0 || !confirmationRequired {
				t.Fatalf("selected=%#v evidence=%#v errors=%#v confirmation=%v", selected, evidence, governanceErrors, confirmationRequired)
			}
			if len(evidence) != 1 || evidence[0].GetIncluded() || evidence[0].GetDecisionReason() != "confirmation_required" {
				t.Fatalf("evidence = %#v", evidence)
			}
		})
	}

	t.Run("critical auto is excluded", func(t *testing.T) {
		store := newFakeAIStore()
		store.agentSkillVersionDocs = []embeddinginfra.AgentSkillVersionEmbeddingDocument{
			runtimeSkillVersionDocument(101, 1, domainagentskill.CompositionRolePrimary, domainagentskill.RiskLevelCritical, "resume screening core"),
		}
		service := newNativeAIService(store, nil, nil, nil, nil)
		selected, evidence, confirmationRequired, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
			context.Background(), &pb.ChatRequest{Message: "resume screening"}, nil,
			runtimeSkillModel([]int64{101}, CapabilitySkillRuntimePolicy{}), true,
		)
		if len(selected) != 0 || confirmationRequired || len(governanceErrors) != 0 ||
			len(evidence) != 1 || evidence[0].GetDecisionReason() != "critical_auto_excluded" {
			t.Fatalf("selected=%#v evidence=%#v errors=%#v confirmation=%v", selected, evidence, governanceErrors, confirmationRequired)
		}
	})
}

func TestHRRuntimeSkillConfirmationStopsProviderBeforeInvocation(t *testing.T) {
	base := newFakeAIStore()
	base.agentConfigs = []*pb.AgentConfigInfo{{
		Id: 10, Name: "hr", AgentType: hrRecruitingAgentType, PromptTemplateId: 20, IsDefault: true, IsEnabled: true,
	}}
	base.promptByID[20] = &pb.PromptTemplateInfo{Id: 20, AgentType: hrRecruitingAgentType, PromptRole: hrRuntimePromptRoleSystem, IsActive: true, Content: "system"}
	base.agentSkillVersionDocs = []embeddinginfra.AgentSkillVersionEmbeddingDocument{
		runtimeSkillVersionDocument(101, 1, domainagentskill.CompositionRolePrimary, domainagentskill.RiskLevelHigh, "resume screening core"),
	}
	store := &releasedSkillFakeStore{
		fakeAIStore: base,
		resolution:  runtimeSkillModel([]int64{101}, CapabilitySkillRuntimePolicy{}),
	}
	provider := &fakeChatProvider{reply: "must not run"}
	service := newNativeAIService(store, provider, nil, nil, nil)
	service.skillPackageV2Enabled = true

	_, err := service.Chat(context.Background(), &pb.ChatRequest{HrId: 77, Message: "resume screening"})
	if status.Code(err) != codes.FailedPrecondition || !strings.Contains(status.Convert(err).Message(), "AGENT_SKILL_CONFIRMATION_REQUIRES_DURABLE_RUN") {
		t.Fatalf("Chat error = %v", err)
	}
	if provider.calls != 0 {
		t.Fatalf("provider calls = %d, want zero before Skill confirmation", provider.calls)
	}
}

func TestSelectHRRuntimeAgentSkillPackagesBlocksEntireCompositionPendingConfirmation(t *testing.T) {
	store := newFakeAIStore()
	store.agentSkillVersionDocs = []embeddinginfra.AgentSkillVersionEmbeddingDocument{
		runtimeSkillVersionDocument(101, 1, domainagentskill.CompositionRolePrimary, domainagentskill.RiskLevelHigh, "resume primary"),
		runtimeSkillVersionDocument(102, 2, domainagentskill.CompositionRoleSupporting, domainagentskill.RiskLevelLow, "resume supporting"),
	}
	service := newNativeAIService(store, nil, nil, nil, nil)

	selected, evidence, confirmationRequired, governanceErrors := service.selectHRRuntimeAgentSkillPackages(
		context.Background(),
		&pb.ChatRequest{Message: "resume"},
		nil,
		runtimeSkillModel([]int64{101, 102}, CapabilitySkillRuntimePolicy{}),
		true,
	)

	if len(selected) != 0 || len(governanceErrors) != 0 || !confirmationRequired {
		t.Fatalf("selected=%#v evidence=%#v errors=%#v confirmation=%v", selected, evidence, governanceErrors, confirmationRequired)
	}
	if len(evidence) != 2 ||
		evidence[0].GetDecisionReason() != "confirmation_required" ||
		evidence[1].GetDecisionReason() != "blocked_by_confirmation" ||
		evidence[0].GetIncluded() || evidence[1].GetIncluded() {
		t.Fatalf("evidence = %#v, want whole composition blocked", evidence)
	}
}

func TestEffectiveAgentSkillBudgetHonorsStricterSnapshotPolicy(t *testing.T) {
	model := RuntimeModelInfo{
		ContextWindowTokens: 10000,
		MaxOutputTokens:     1000,
		SkillRuntimePolicy: CapabilitySkillRuntimePolicy{
			MaxSkillTokens: 700,
			MaxInputRatio:  0.05,
			MaxSkills:      1,
		},
	}
	if got := effectiveAgentSkillBudget(model); got != 425 {
		t.Fatalf("budget = %d, want floor((10000-1000-500)*0.05)=425", got)
	}
	if got := effectiveAgentSkillMaxCount(model.SkillRuntimePolicy); got != 1 {
		t.Fatalf("max skills = %d, want 1", got)
	}
}

func TestRuntimeRequestDescriptorsExposeOnlyV2SkillAndCapabilityFields(t *testing.T) {
	messages := []struct {
		name   string
		fields protoreflect.FieldDescriptors
	}{
		{name: "ChatRequest", fields: (&pb.ChatRequest{}).ProtoReflect().Descriptor().Fields()},
		{name: "CreateAgentRunRequest", fields: (&pb.CreateAgentRunRequest{}).ProtoReflect().Descriptor().Fields()},
		{name: "PreviewChatContextRequest", fields: (&pb.PreviewChatContextRequest{}).ProtoReflect().Descriptor().Fields()},
	}
	for _, message := range messages {
		t.Run(message.name, func(t *testing.T) {
			for _, name := range []string{
				"agent_skill_ids",
				"agent_skill_selection_confirmed",
				"agent_skill_selection_message_id",
				"skill_capability_keys",
			} {
				if message.fields.ByName(protoreflectName(name)) != nil {
					t.Fatalf("legacy field %q still exists", name)
				}
			}
			for _, name := range []string{"agent_skill_version_ids", "capability_keys"} {
				if message.fields.ByName(protoreflectName(name)) == nil {
					t.Fatalf("v2 field %q is missing", name)
				}
			}
		})
	}
}

type releasedSkillFakeStore struct {
	*fakeAIStore
	resolution RuntimeModelInfo
}

func (s *releasedSkillFakeStore) ResolveCapabilityRuntimeModel(_ context.Context, _, _ string, _, requestedModelID int64) (CapabilityRuntimeModelResolution, error) {
	model := s.resolution
	if requestedModelID > 0 {
		model.ID = requestedModelID
		model.RequestedModelID = requestedModelID
	}
	return CapabilityRuntimeModelResolution{
		EffectiveModelID:       model.ID,
		ModelName:              model.Name,
		ProviderName:           model.ProviderName,
		RequestedModelID:       model.RequestedModelID,
		CapabilityVersionID:    model.CapabilityVersionID,
		CapabilitySnapshotHash: model.CapabilitySnapshotHash,
		ContextWindowTokens:    model.ContextWindowTokens,
		MaxOutputTokens:        model.MaxOutputTokens,
		ConfigurationRefs:      model.ConfigurationRefs,
		SkillRuntimePolicy:     model.SkillRuntimePolicy,
	}, nil
}

func runtimeSkillModel(versionIDs []int64, policy CapabilitySkillRuntimePolicy) RuntimeModelInfo {
	return RuntimeModelInfo{
		ID:                     1,
		Name:                   "test-model",
		ContextWindowTokens:    8192,
		MaxOutputTokens:        1024,
		CapabilityVersionID:    77,
		CapabilitySnapshotHash: strings.Repeat("a", 64),
		ConfigurationRefs: CapabilityConfigurationRefs{
			AgentIDs:             []int64{10},
			PromptTemplateIDs:    []int64{20},
			AgentSkillVersionIDs: append([]int64(nil), versionIDs...),
		},
		SkillRuntimePolicy: policy,
	}
}

func runtimeSkillVersionDocument(versionID, skillID int64, role domainagentskill.CompositionRole, risk domainagentskill.RiskLevel, core string) embeddinginfra.AgentSkillVersionEmbeddingDocument {
	activation := domainagentskill.ActivationPolicyAuto
	switch risk {
	case domainagentskill.RiskLevelHigh:
		activation = domainagentskill.ActivationPolicyConfirm
	case domainagentskill.RiskLevelCritical:
		activation = domainagentskill.ActivationPolicyManualOnly
	}
	return embeddinginfra.AgentSkillVersionEmbeddingDocument{
		ID:              versionID,
		SkillID:         skillID,
		Version:         "2.0.0",
		CompiledHash:    strings.Repeat("b", 64),
		CoreMarkdown:    core,
		RegistryName:    "registry",
		RegistryLabel:   "Registry",
		Enabled:         true,
		ManualInvocable: true,
		Manifest: domainagentskill.Manifest{
			SchemaVersion:    domainagentskill.SchemaVersion,
			SkillName:        "runtime-skill",
			DisplayName:      "Skill",
			AgentType:        hrRecruitingAgentType,
			Category:         "screening",
			Scenario:         "resume",
			RiskLevel:        risk,
			ActivationPolicy: activation,
			Composition:      domainagentskill.Composition{Role: role},
			TriggerKeywords:  []string{"resume", "screening"},
		},
	}
}

func runtimeSkillPackage(
	t *testing.T,
	document embeddinginfra.AgentSkillVersionEmbeddingDocument,
	sections []embeddinginfra.AgentSkillSectionEmbeddingDocument,
) embeddinginfra.AgentSkillRuntimePackage {
	t.Helper()
	draft := domainagentskill.PackageDraft{
		Manifest: document.Manifest,
		Core:     domainagentskill.Core{ContentMarkdown: document.CoreMarkdown},
	}
	for i, section := range sections {
		draft.Sections = append(draft.Sections, domainagentskill.ReferenceSection{
			SectionKey:      section.SectionKey,
			Title:           section.Title,
			Description:     section.Description,
			ContentMarkdown: section.ContentMarkdown,
			TriggerTerms:    append([]string(nil), section.TriggerTerms...),
			SemanticTags:    append([]string(nil), section.SemanticTags...),
			PlannerIntents:  append([]string(nil), section.PlannerIntents...),
			Priority:        section.Priority,
			Ordinal:         i,
		})
	}
	compiled, err := domainagentskill.Compile(draft)
	if err != nil {
		t.Fatalf("compile runtime Skill package: %v", err)
	}
	runtimePackage := embeddinginfra.AgentSkillRuntimePackage{
		ID:                  document.ID,
		SkillID:             document.SkillID,
		Version:             document.Version,
		ManifestJSON:        compiled.ManifestJSON,
		CoreMarkdown:        compiled.Core.ContentMarkdown,
		CompiledMarkdown:    compiled.CompiledMarkdown,
		CompiledHash:        compiled.CompiledHash,
		CoreEstimatedTokens: compiled.Core.EstimatedTokens,
		Enabled:             document.Enabled,
		ManualInvocable:     document.ManualInvocable,
	}
	for i, compiledSection := range compiled.Sections {
		runtimePackage.Sections = append(runtimePackage.Sections, embeddinginfra.AgentSkillRuntimeSection{
			ID:              sections[i].ID,
			SectionKey:      compiledSection.SectionKey,
			Title:           compiledSection.Title,
			Description:     compiledSection.Description,
			ContentMarkdown: compiledSection.ContentMarkdown,
			TriggerTerms:    append([]string{}, compiledSection.TriggerTerms...),
			SemanticTags:    append([]string{}, compiledSection.SemanticTags...),
			PlannerIntents:  append([]string{}, compiledSection.PlannerIntents...),
			Priority:        compiledSection.Priority,
			Ordinal:         compiledSection.Ordinal,
			EstimatedTokens: compiledSection.EstimatedTokens,
			ContentHash:     compiledSection.ContentHash,
		})
	}
	return runtimePackage
}

func reformatRuntimeManifestJSON(
	t *testing.T,
	raw string,
	mutate func(map[string]any),
) string {
	t.Helper()
	var manifest map[string]any
	if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
		t.Fatalf("decode runtime manifest: %v", err)
	}
	if mutate != nil {
		mutate(manifest)
	}
	reformatted, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("reformat runtime manifest: %v", err)
	}
	return string(reformatted)
}

func protoreflectName(value string) protoreflect.Name {
	return protoreflect.Name(value)
}
