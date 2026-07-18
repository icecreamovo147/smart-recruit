package recruiting_intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type candidateMatcherProvider struct {
	responses []string
	err       error
	systems   []string
	users     []string
}

func (provider *candidateMatcherProvider) CompleteStructured(_ context.Context, systemPrompt, userPrompt string) (StructuredCompletionResult, error) {
	provider.systems = append(provider.systems, systemPrompt)
	provider.users = append(provider.users, userPrompt)
	if provider.err != nil {
		return StructuredCompletionResult{}, provider.err
	}
	index := len(provider.systems) - 1
	if index >= len(provider.responses) {
		return StructuredCompletionResult{}, errors.New("missing test response")
	}
	return StructuredCompletionResult{Content: provider.responses[index], ModelName: "matcher-model"}, nil
}

func TestBuildEvidenceIndexPrivacyBoundsAllowlistAndStableRealIDs(t *testing.T) {
	source := CandidateEvidenceSource{
		Skills: []CandidateEvidenceSkill{
			{ID: 12, Name: "Kubernetes", Evidence: "联系 test@example.com"},
			{ID: 11, Name: "Go", Evidence: strings.Repeat("服务开发 ", 80)},
			{ID: 0, Name: "不得生成 ID"},
			{ID: 11, Name: "重复行应被确定性去重"},
		},
		Experiences: []CandidateEvidenceExperience{{ID: 21, Title: "后端工程师", Description: "电话 13800138000"}},
		Projects:    []CandidateEvidenceProject{{ID: 31, Name: "招聘平台", Technologies: []string{"Go"}}},
		Educations:  []CandidateEvidenceEducation{{ID: 41, School: "示例大学", Degree: "本科"}},
		Resume:      &CandidateEvidenceResume{ID: 51, ParsedText: "邮箱 resume@example.com，具备分布式经验"},
		CandidateProfile: &CandidateEvidenceProfile{
			ID: 61, Education: "本科", School: "示例大学", WorkExperience: "五年后端经验", Skills: "Go, Redis",
		},
	}

	first := BuildEvidenceIndex(source)
	second := BuildEvidenceIndex(source)
	if !reflect.DeepEqual(first.Units(), second.Units()) {
		t.Fatalf("index is not stable:\nfirst=%#v\nsecond=%#v", first.Units(), second.Units())
	}
	if len(first.Units()) != 7 {
		t.Fatalf("units = %d, want seven allowlisted real rows", len(first.Units()))
	}
	seen := make(map[string]struct{})
	for _, unit := range first.Units() {
		if _, allowed := candidateEvidenceSourceRank(unit.SourceTable); !allowed {
			t.Fatalf("non-allowlisted table = %q", unit.SourceTable)
		}
		if unit.SourceID == 0 {
			t.Fatal("index invented or retained zero source ID")
		}
		key := evidenceKey(unit.SourceTable, unit.SourceID)
		if _, duplicate := seen[key]; duplicate {
			t.Fatalf("duplicate source identity = %s", key)
		}
		seen[key] = struct{}{}
		if len([]rune(unit.Snippet)) > MaxCandidateEvidenceRunes {
			t.Fatalf("snippet length = %d", len([]rune(unit.Snippet)))
		}
		if containsCandidateContactData(unit.Snippet) || strings.Contains(unit.Snippet, "test@example.com") || strings.Contains(unit.Snippet, "13800138000") {
			t.Fatalf("snippet leaked contact data: %q", unit.Snippet)
		}
	}
	if first.Units()[0].SourceID != 11 || first.Units()[1].SourceID != 12 {
		t.Fatalf("skill ordering = %d,%d, want stable ID order", first.Units()[0].SourceID, first.Units()[1].SourceID)
	}
}

func TestCandidateContactRedactionCoversPhonesWithoutErasingOrdinaryNumbers(t *testing.T) {
	contacts := []string{
		"大陆手机 13800138000",
		"分隔手机 +86 138-0013-8000",
		"北京座机 010-12345678",
		"上海座机 021 87654321",
		"北美号码 +1 (415) 555-2671",
		"英国号码 +44 20 7946 0958",
	}
	for _, contact := range contacts {
		if !containsCandidateContactData(contact) {
			t.Fatalf("contact not detected: %q", contact)
		}
		redacted := redactCandidateContactData(contact)
		if !strings.Contains(redacted, "[PHONE]") || containsCandidateContactData(redacted) {
			t.Fatalf("contact not safely redacted: input=%q output=%q", contact, redacted)
		}
	}

	ordinary := "版本 2026-07-15，订单 12345678，编号 01012345678"
	if containsCandidateContactData(ordinary) || redactCandidateContactData(ordinary) != ordinary {
		t.Fatalf("ordinary numeric content was treated as contact data: %q", redactCandidateContactData(ordinary))
	}
}

func TestBuildEvidenceIndexGlobalBound(t *testing.T) {
	source := CandidateEvidenceSource{Skills: make([]CandidateEvidenceSkill, 125)}
	for index := range source.Skills {
		source.Skills[index] = CandidateEvidenceSkill{ID: uint64(index + 1), Name: fmt.Sprintf("skill-%03d", index)}
	}
	got := BuildEvidenceIndex(source)
	if len(got.Units()) != MaxCandidateEvidenceUnits {
		t.Fatalf("units = %d, want %d", len(got.Units()), MaxCandidateEvidenceUnits)
	}
	if got.Units()[0].SourceID != 1 || got.Units()[len(got.Units())-1].SourceID != MaxCandidateEvidenceUnits {
		t.Fatalf("bounded IDs = %d..%d", got.Units()[0].SourceID, got.Units()[len(got.Units())-1].SourceID)
	}
}

func TestEvidenceIndexUnitsAreDefensive(t *testing.T) {
	index := BuildEvidenceIndex(CandidateEvidenceSource{Skills: []CandidateEvidenceSkill{{ID: 7, Name: "Go"}}})
	copyOfUnits := index.Units()
	copyOfUnits[0].SourceTable = "untrusted_table"
	copyOfUnits[0].Snippet = "fabricated"
	copyOfUnits[0].NormalizedTerms[0] = "fabricated"
	original := index.Units()[0]
	if original.SourceTable != "resume_skills" || original.Snippet != "Go" || original.NormalizedTerms[0] != "go" {
		t.Fatalf("trusted index was mutated through defensive copy: %#v", original)
	}
}

func TestCandidateRequirementEvaluatorDeterministicHitMakesZeroMatcherCalls(t *testing.T) {
	provider := &candidateMatcherProvider{}
	evaluator, _ := newTestCandidateEvaluator(provider, RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true, Fallbacks: false})
	index := BuildEvidenceIndex(CandidateEvidenceSource{Skills: []CandidateEvidenceSkill{{ID: 701, Name: "Go", Evidence: "微服务开发"}}})
	profile := testRequirementProfile(JobRequirementItem{ID: "go", Category: RequirementCategoryCoreSkill, Label: "Go 开发能力", Description: "要求具备 Go 开发能力。", Priority: RequirementPriorityMustHave, Weight: 1, Aliases: []string{"Go"}})

	results, err := evaluator.Evaluate(context.Background(), profile, index)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(provider.systems) != 0 {
		t.Fatalf("matcher calls = %d, want zero", len(provider.systems))
	}
	if len(results) != 1 || results[0].EvaluatorType != MatchEvaluatorDeterministic || results[0].Status != MatchStatusStrongMatch || results[0].Evidence[0].SourceID != 701 {
		t.Fatalf("results = %#v", results)
	}
}

func TestCandidateRequirementEvaluatorCallsOncePerUnresolvedWithSystemUserRoles(t *testing.T) {
	provider := &candidateMatcherProvider{responses: []string{
		`{"status":"missing","score":0,"confidence":0.8,"risk":"未找到相关证据","evidence":[]}`,
		`{"status":"missing","score":0,"confidence":0.7,"risk":"未找到相关证据","evidence":[]}`,
	}}
	evaluator, store := newTestCandidateEvaluator(provider, RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true, Fallbacks: false})
	profile := JobRequirementProfile{ProfileVersion: JobRequirementProfileVersion, Requirements: []JobRequirementItem{
		{ID: "java", Category: RequirementCategoryCoreSkill, Label: "Java 开发能力", Description: "要求具备 Java 开发能力。", Priority: RequirementPriorityMustHave, Weight: 0.5, Aliases: []string{"Java"}},
		{ID: "python", Category: RequirementCategoryCoreSkill, Label: "Python 开发能力", Description: "要求具备 Python 开发能力。", Priority: RequirementPriorityMustHave, Weight: 0.5, Aliases: []string{"Python"}},
	}}

	results, err := evaluator.Evaluate(context.Background(), profile, BuildEvidenceIndex(CandidateEvidenceSource{}))
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(results) != 2 || len(provider.systems) != 2 || len(store.calls) != 2 {
		t.Fatalf("results/provider/store calls = %d/%d/%d", len(results), len(provider.systems), len(store.calls))
	}
	for index := range provider.systems {
		if provider.systems[index] != "candidate matcher system" {
			t.Fatalf("system[%d] = %q", index, provider.systems[index])
		}
		if !strings.Contains(provider.users[index], `"requirement"`) || !strings.Contains(provider.users[index], `"candidate_evidence"`) {
			t.Fatalf("user[%d] is not scoped JSON: %q", index, provider.users[index])
		}
		if store.calls[index] != [2]string{AgentTypeCandidateMatchEvaluator, PromptRoleSystem} {
			t.Fatalf("prompt call[%d] = %#v", index, store.calls[index])
		}
	}
}

func TestCandidateRequirementEvaluatorDoesNotMatchRedactionMarkers(t *testing.T) {
	provider := &candidateMatcherProvider{responses: []string{
		`{"status":"missing","score":0,"confidence":0.8,"risk":"未找到相关证据","evidence":[]}`,
		`{"status":"missing","score":0,"confidence":0.8,"risk":"未找到相关证据","evidence":[]}`,
	}}
	evaluator, _ := newTestCandidateEvaluator(provider, RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true, Fallbacks: false})
	index := BuildEvidenceIndex(CandidateEvidenceSource{Resume: &CandidateEvidenceResume{
		ID: 99, ParsedText: "联系方式 test@example.com，电话 010-12345678",
	}})
	terms := index.Units()[0].NormalizedTerms
	if slicesContainString(terms, "phone") || slicesContainString(terms, "email") {
		t.Fatalf("redaction markers entered searchable terms: %#v", terms)
	}
	profile := JobRequirementProfile{ProfileVersion: JobRequirementProfileVersion, Requirements: []JobRequirementItem{
		{ID: "phone", Category: RequirementCategoryOther, Label: "电话相关能力", Description: "评估电话相关岗位能力。", Priority: RequirementPriorityMustHave, Weight: 0.5, Aliases: []string{"phone"}},
		{ID: "email", Category: RequirementCategoryOther, Label: "邮件相关能力", Description: "评估邮件相关岗位能力。", Priority: RequirementPriorityMustHave, Weight: 0.5, Aliases: []string{"email"}},
	}}
	results, err := evaluator.Evaluate(context.Background(), profile, index)
	if err != nil || len(results) != 2 || len(provider.systems) != 2 {
		t.Fatalf("results/error/calls = %#v/%v/%d, want two unresolved matcher calls", results, err, len(provider.systems))
	}
	for _, result := range results {
		if result.EvaluatorType != MatchEvaluatorLLM || result.Status != MatchStatusMissing {
			t.Fatalf("marker produced a deterministic match: %#v", result)
		}
	}
}

func slicesContainString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestDecodeMatcherOutputStrictSchemaAndProvenance(t *testing.T) {
	index := BuildEvidenceIndex(CandidateEvidenceSource{Skills: []CandidateEvidenceSkill{{ID: 9, Name: "Go"}}})
	snippet := index.Units()[0].Snippet
	validEvidence := fmt.Sprintf(`[{"source_table":"resume_skills","source_id":9,"snippet":%q,"reason":"该技能直接支持岗位要求"}]`, snippet)
	tests := []struct {
		name    string
		content string
		wantErr error
	}{
		{name: "valid", content: fmt.Sprintf(`{"status":"match","score":88,"confidence":0.9,"risk":"","evidence":%s}`, validEvidence)},
		{name: "unknown top field", content: `{"status":"missing","score":0,"confidence":0.5,"risk":"","evidence":[],"overall_score":99}`, wantErr: ErrCandidateMatchSchema},
		{name: "duplicate top field", content: `{"status":"missing","status":"match","score":0,"confidence":0.5,"risk":"","evidence":[]}`, wantErr: ErrCandidateMatchSchema},
		{name: "missing required risk", content: `{"status":"missing","score":0,"confidence":0.5,"evidence":[]}`, wantErr: ErrCandidateMatchSchema},
		{name: "invalid enum", content: `{"status":"推荐","score":50,"confidence":0.5,"risk":"","evidence":[]}`, wantErr: ErrCandidateMatchSchema},
		{name: "score range", content: `{"status":"missing","score":101,"confidence":0.5,"risk":"","evidence":[]}`, wantErr: ErrCandidateMatchSchema},
		{name: "confidence range", content: `{"status":"missing","score":0,"confidence":1.1,"risk":"","evidence":[]}`, wantErr: ErrCandidateMatchSchema},
		{name: "unknown source table", content: `{"status":"match","score":80,"confidence":0.8,"risk":"","evidence":[{"source_table":"resume_profiles","source_id":9,"snippet":"Go","reason":"证据支持"}]}`, wantErr: ErrCandidateMatchProvenance},
		{name: "unknown source id", content: `{"status":"match","score":80,"confidence":0.8,"risk":"","evidence":[{"source_table":"resume_skills","source_id":99,"snippet":"Go","reason":"证据支持"}]}`, wantErr: ErrCandidateMatchProvenance},
		{name: "fabricated snippet", content: `{"status":"match","score":80,"confidence":0.8,"risk":"","evidence":[{"source_table":"resume_skills","source_id":9,"snippet":"Go and fabricated secret","reason":"证据支持"}]}`, wantErr: ErrCandidateMatchProvenance},
		{name: "duplicate nested evidence field", content: fmt.Sprintf(`{"status":"match","score":80,"confidence":0.8,"risk":"","evidence":[{"source_table":"resume_skills","source_id":9,"source_id":9,"snippet":%q,"reason":"证据支持"}]}`, snippet), wantErr: ErrCandidateMatchSchema},
		{name: "leaked reason", content: fmt.Sprintf(`{"status":"match","score":80,"confidence":0.8,"risk":"","evidence":[{"source_table":"resume_skills","source_id":9,"snippet":%q,"reason":"联系 test@example.com"}]}`, snippet), wantErr: ErrCandidateMatchSchema},
		{name: "english risk clause", content: `{"status":"missing","score":0,"confidence":0.5,"risk":"风险 candidate lacks required experience","evidence":[]}`, wantErr: ErrCandidateMatchSchema},
		{name: "uppercase english risk clause", content: `{"status":"missing","score":0,"confidence":0.5,"risk":"风险 CANDIDATE LACKS REQUIRED EXPERIENCE","evidence":[]}`, wantErr: ErrCandidateMatchSchema},
		{name: "camel case english risk clause", content: `{"status":"missing","score":0,"confidence":0.5,"risk":"风险 candidateLacksRequiredExperience","evidence":[]}`, wantErr: ErrCandidateMatchSchema},
		{name: "english reason clause", content: fmt.Sprintf(`{"status":"match","score":80,"confidence":0.8,"risk":"","evidence":[{"source_table":"resume_skills","source_id":9,"snippet":%q,"reason":"证据 candidate has required experience"}]}`, snippet), wantErr: ErrCandidateMatchSchema},
		{name: "simplified chinese with existing technical names", content: fmt.Sprintf(`{"status":"match","score":80,"confidence":0.8,"risk":"候选人缺少 Kubernetes 生产环境证据","evidence":[{"source_table":"resume_skills","source_id":9,"snippet":%q,"reason":"候选人的 Go 与 gRPC 经验直接支持岗位要求"}]}`, snippet)},
		{name: "simplified chinese with node js", content: fmt.Sprintf(`{"status":"match","score":80,"confidence":0.8,"risk":"候选人缺少 Node.js 生产环境证据","evidence":[{"source_table":"resume_skills","source_id":9,"snippet":%q,"reason":"该证据可用于评估 Node.js 岗位要求"}]}`, snippet)},
		{name: "trailing json", content: `{"status":"missing","score":0,"confidence":0.5,"risk":"","evidence":[]} {}`, wantErr: ErrCandidateMatchSchema},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := decodeMatcherOutput(test.content, "go", index)
			if test.wantErr == nil {
				if err != nil || result.RequirementID != "go" || result.EvaluatorType != MatchEvaluatorLLM {
					t.Fatalf("result/error = %#v/%v", result, err)
				}
				return
			}
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestCandidateRequirementEvaluatorIndividualFailureFallbackPolicy(t *testing.T) {
	profile := JobRequirementProfile{ProfileVersion: JobRequirementProfileVersion, Requirements: []JobRequirementItem{
		{ID: "go", Category: RequirementCategoryCoreSkill, Label: "Go 开发能力", Description: "要求具备 Go 开发能力。", Priority: RequirementPriorityMustHave, Weight: 0.5, Aliases: []string{"Go"}},
		{ID: "java", Category: RequirementCategoryCoreSkill, Label: "Java 开发能力", Description: "要求具备 Java 开发能力。", Priority: RequirementPriorityMustHave, Weight: 0.5, Aliases: []string{"Java"}},
	}}
	index := BuildEvidenceIndex(CandidateEvidenceSource{Skills: []CandidateEvidenceSkill{{ID: 101, Name: "Go"}}})

	for _, fallback := range []bool{true, false} {
		t.Run(fmt.Sprintf("fallback_%t", fallback), func(t *testing.T) {
			provider := &candidateMatcherProvider{err: errors.New("provider unavailable with private payload")}
			evaluator, _ := newTestCandidateEvaluator(provider, RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true, Fallbacks: fallback})
			results, err := evaluator.Evaluate(context.Background(), profile, index)
			if len(results) != 2 || results[0].Status != MatchStatusStrongMatch || results[1].Status != MatchStatusMissing || results[1].FallbackUsed != fallback {
				t.Fatalf("results = %#v", results)
			}
			if fallback && err != nil {
				t.Fatalf("fallback-enabled error = %v", err)
			}
			if !fallback && err == nil {
				t.Fatal("fallback-disabled evaluation should abort")
			}
			if err != nil && strings.Contains(err.Error(), "private payload") {
				t.Fatalf("error leaked provider content: %v", err)
			}
		})
	}
}

func TestCandidateRequirementEvaluatorLanguageFailureFollowsFallbackPolicy(t *testing.T) {
	profile := testRequirementProfile(JobRequirementItem{ID: "java", Category: RequirementCategoryCoreSkill, Label: "Java 开发能力", Description: "要求具备 Java 开发能力。", Priority: RequirementPriorityMustHave, Weight: 1, Aliases: []string{"Java"}})
	for _, fallback := range []bool{true, false} {
		t.Run(fmt.Sprintf("fallback_%t", fallback), func(t *testing.T) {
			provider := &candidateMatcherProvider{responses: []string{`{"status":"missing","score":0,"confidence":0.8,"risk":"风险 candidate lacks required experience","evidence":[]}`}}
			evaluator, _ := newTestCandidateEvaluator(provider, RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true, Fallbacks: fallback})
			results, err := evaluator.Evaluate(context.Background(), profile, BuildEvidenceIndex(CandidateEvidenceSource{}))
			if len(results) != 1 || results[0].Status != MatchStatusMissing || results[0].FallbackUsed != fallback {
				t.Fatalf("results = %#v", results)
			}
			if fallback && err != nil {
				t.Fatalf("fallback-enabled error = %v", err)
			}
			if !fallback && !errors.Is(err, ErrCandidateMatchSchema) {
				t.Fatalf("fallback-disabled error = %v, want schema", err)
			}
		})
	}
}

func TestCandidateRequirementEvaluatorSemanticAndShadowIntent(t *testing.T) {
	profile := testRequirementProfile(JobRequirementItem{ID: "java", Category: RequirementCategoryCoreSkill, Label: "Java 开发能力", Description: "要求具备 Java 开发能力。", Priority: RequirementPriorityMustHave, Weight: 1, Aliases: []string{"Java"}})
	tests := []struct {
		name     string
		semantic bool
		shadow   bool
		calls    int
	}{
		{name: "deterministic only", calls: 0},
		{name: "enhanced primary", semantic: true, calls: 1},
		{name: "enhanced shadow", shadow: true, calls: 1},
		{name: "enhanced primary deterministic shadow", semantic: true, shadow: true, calls: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := &candidateMatcherProvider{responses: []string{`{"status":"missing","score":0,"confidence":0.8,"risk":"缺少证据","evidence":[]}`}}
			evaluator, _ := newTestCandidateEvaluator(provider, RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: test.semantic, CandidateMatchShadow: test.shadow, Fallbacks: false})
			results, err := evaluator.Evaluate(context.Background(), profile, BuildEvidenceIndex(CandidateEvidenceSource{}))
			if err != nil || len(results) != 1 || len(provider.systems) != test.calls {
				t.Fatalf("results/error/calls = %#v/%v/%d, want calls %d", results, err, len(provider.systems), test.calls)
			}
		})
	}
}

func newTestCandidateEvaluator(provider StructuredCompletionProvider, config RuntimePolicyConfig) (*CandidateRequirementEvaluator, *runtimePromptStore) {
	policy := NewRuntimePolicy(config)
	store := &runtimePromptStore{prompt: PromptDescriptor{
		ID: 88, Name: "candidate-matcher-v2", Version: 2,
		AgentType: AgentTypeCandidateMatchEvaluator, Role: PromptRoleSystem, Content: "candidate matcher system",
	}}
	runtime := NewRuntime(NewPromptLoader(store), provider, policy)
	return NewCandidateRequirementEvaluator(runtime, policy), store
}

func testRequirementProfile(requirement JobRequirementItem) JobRequirementProfile {
	return JobRequirementProfile{ProfileVersion: JobRequirementProfileVersion, Requirements: []JobRequirementItem{requirement}}
}

func TestCandidateMatcherUserMessageContainsOnlyIndexedEvidence(t *testing.T) {
	index := BuildEvidenceIndex(CandidateEvidenceSource{Resume: &CandidateEvidenceResume{ID: 55, ParsedText: "Go 开发，联系 test@example.com，座机 010-12345678，国际 +44 20 7946 0958"}})
	requirement := JobRequirementItem{ID: "go", Category: RequirementCategoryCoreSkill, Label: "Go 开发能力", Description: "要求具备 Go 开发能力。", Priority: RequirementPriorityMustHave, Weight: 1, Aliases: []string{"Go"}}
	message, err := candidateMatcherUserMessage(requirement, index)
	if err != nil {
		t.Fatalf("candidateMatcherUserMessage: %v", err)
	}
	if strings.Contains(message, "test@example.com") || strings.Contains(message, "010-12345678") || strings.Contains(message, "+44 20 7946 0958") || !strings.Contains(message, "[EMAIL]") || !strings.Contains(message, "[PHONE]") {
		t.Fatalf("message privacy = %q", message)
	}
	var decoded struct {
		Evidence []EvidenceUnit `json:"candidate_evidence"`
	}
	if err := json.Unmarshal([]byte(message), &decoded); err != nil || !reflect.DeepEqual(decoded.Evidence, index.Units()) {
		t.Fatalf("message/index mismatch: err=%v decoded=%#v index=%#v", err, decoded.Evidence, index.Units())
	}
}
