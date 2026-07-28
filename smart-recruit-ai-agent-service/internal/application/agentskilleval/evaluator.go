package agentskilleval

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	"smart-recruit-ai-agent-service/internal/domain/agentskill"
)

const (
	SuiteSchemaVersion = 1
	PolicyVersion      = "skill-package-v2.1"
)

//go:embed testdata/suite-v1.json
var suiteFiles embed.FS

var (
	defaultEvaluatorOnce sync.Once
	defaultEvaluator     *Evaluator
	defaultEvaluatorErr  error
)

type RuntimePolicy struct {
	PolicyVersion  string  `json:"policy_version"`
	MaxSkillTokens int     `json:"max_skill_tokens"`
	MaxInputRatio  float64 `json:"max_input_ratio"`
	MaxSkills      int     `json:"max_skills"`
}

type ReleasePackage struct {
	SkillID   int64
	VersionID int64
	Package   agentskill.CompiledPackage
}

type Input struct {
	CapabilityKey string
	Audience      string
	Policy        RuntimePolicy
	Packages      []ReleasePackage
}

type CaseResult struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Passed bool   `json:"passed"`
	Code   string `json:"code"`
}

type Result struct {
	Passed       bool         `json:"passed"`
	SuiteVersion string       `json:"suite_version"`
	SuiteHash    string       `json:"suite_hash"`
	ResultHash   string       `json:"result_hash"`
	Cases        []CaseResult `json:"cases"`
}

type Evaluator struct {
	suite     Suite
	suiteHash string
}

type Suite struct {
	SchemaVersion    int               `json:"schema_version"`
	SuiteVersion     string            `json:"suite_version"`
	CompileCases     []CompileCase     `json:"compile_cases"`
	RankingCases     []RankingCase     `json:"ranking_cases"`
	CompositionCases []CompositionCase `json:"composition_cases"`
}

type CompileCase struct {
	ID                   string `json:"id"`
	Fixture              string `json:"fixture"`
	ExpectedPass         bool   `json:"expected_pass"`
	ExpectedErrorCode    string `json:"expected_error_code,omitempty"`
	ExpectedCompiledHash string `json:"expected_compiled_hash,omitempty"`
}

type RankingCase struct {
	ID                string             `json:"id"`
	Query             string             `json:"query"`
	Candidates        []RankingCandidate `json:"candidates"`
	ExpectedObjectIDs []int64            `json:"expected_object_ids"`
	ExpectedModes     []string           `json:"expected_modes"`
}

type RankingCandidate struct {
	ObjectID           int64    `json:"object_id"`
	SkillID            int64    `json:"skill_id,omitempty"`
	VersionID          int64    `json:"version_id,omitempty"`
	SectionID          int64    `json:"section_id,omitempty"`
	LexicalText        []string `json:"lexical_text,omitempty"`
	MetadataTerms      []string `json:"metadata_terms,omitempty"`
	Priority           int      `json:"priority,omitempty"`
	VectorScore        float64  `json:"vector_score,omitempty"`
	EmbeddingAvailable bool     `json:"embedding_available"`
}

type CompositionCase struct {
	ID           string               `json:"id"`
	Packages     []CompositionPackage `json:"packages"`
	ExpectedPass bool                 `json:"expected_pass"`
}

type CompositionPackage struct {
	AgentType string                     `json:"agent_type"`
	Scenario  string                     `json:"scenario"`
	Role      agentskill.CompositionRole `json:"role"`
}

type resultEnvelope struct {
	SuiteVersion  string            `json:"suite_version"`
	SuiteHash     string            `json:"suite_hash"`
	CapabilityKey string            `json:"capability_key"`
	Audience      string            `json:"audience"`
	Policy        RuntimePolicy     `json:"policy"`
	Packages      []packageIdentity `json:"packages"`
	Cases         []CaseResult      `json:"cases"`
}

type packageIdentity struct {
	SkillID      int64  `json:"skill_id"`
	VersionID    int64  `json:"version_id"`
	CompiledHash string `json:"compiled_hash"`
}

func NewDefaultEvaluator() (*Evaluator, error) {
	defaultEvaluatorOnce.Do(func() {
		raw, err := suiteFiles.ReadFile("testdata/suite-v1.json")
		if err != nil {
			defaultEvaluatorErr = fmt.Errorf("read embedded Agent Skill evaluation suite: %w", err)
			return
		}
		defaultEvaluator, defaultEvaluatorErr = NewEvaluator(raw)
	})
	return defaultEvaluator, defaultEvaluatorErr
}

// NewEvaluator constructs a deterministic evaluator from a versioned suite.
// It is exported so release tooling can inspect the same read-only result as
// the publication gate without involving a model or network provider.
func NewEvaluator(raw []byte) (*Evaluator, error) {
	var suite Suite
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&suite); err != nil {
		return nil, fmt.Errorf("decode Agent Skill evaluation suite: %w", err)
	}
	if err := validateAndNormalizeSuite(&suite); err != nil {
		return nil, err
	}
	canonical, err := canonicalJSON(suite)
	if err != nil {
		return nil, fmt.Errorf("canonicalize Agent Skill evaluation suite: %w", err)
	}
	return &Evaluator{suite: suite, suiteHash: hashBytes(canonical)}, nil
}

func (e *Evaluator) Evaluate(ctx context.Context, input Input) (Result, error) {
	if e == nil {
		return Result{}, errors.New("Agent Skill release evaluator is nil")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	cases := make([]CaseResult, 0, len(e.suite.CompileCases)+len(e.suite.RankingCases)+len(e.suite.CompositionCases)+len(input.Packages)+2)
	for _, testCase := range e.suite.CompileCases {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		cases = append(cases, evaluateCompileCase(testCase))
	}
	for _, testCase := range e.suite.RankingCases {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		cases = append(cases, evaluateRankingCase(testCase))
	}
	for _, testCase := range e.suite.CompositionCases {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		actual := validateComposition(testCase.Packages) == nil
		cases = append(cases, expectationResult(testCase.ID, "composition_fixture", actual == testCase.ExpectedPass))
	}

	packages := append([]ReleasePackage(nil), input.Packages...)
	slices.SortFunc(packages, func(a, b ReleasePackage) int {
		if a.VersionID != b.VersionID {
			return cmp.Compare(a.VersionID, b.VersionID)
		}
		return cmp.Compare(a.SkillID, b.SkillID)
	})
	cases = append(cases, evaluatePolicy(input.Policy))
	for _, item := range packages {
		cases = append(cases, evaluateReleasePackage(item))
	}
	cases = append(cases, evaluateReleaseComposition(packages))
	slices.SortFunc(cases, func(a, b CaseResult) int {
		if a.ID != b.ID {
			return strings.Compare(a.ID, b.ID)
		}
		return strings.Compare(a.Kind, b.Kind)
	})

	passed := true
	for _, item := range cases {
		if !item.Passed {
			passed = false
			break
		}
	}
	identities := make([]packageIdentity, 0, len(packages))
	for _, item := range packages {
		identities = append(identities, packageIdentity{
			SkillID:      item.SkillID,
			VersionID:    item.VersionID,
			CompiledHash: strings.ToLower(strings.TrimSpace(item.Package.CompiledHash)),
		})
	}
	envelope := resultEnvelope{
		SuiteVersion:  e.suite.SuiteVersion,
		SuiteHash:     e.suiteHash,
		CapabilityKey: strings.TrimSpace(input.CapabilityKey),
		Audience:      strings.TrimSpace(input.Audience),
		Policy: RuntimePolicy{
			PolicyVersion:  strings.TrimSpace(input.Policy.PolicyVersion),
			MaxSkillTokens: input.Policy.MaxSkillTokens,
			MaxInputRatio:  input.Policy.MaxInputRatio,
			MaxSkills:      input.Policy.MaxSkills,
		},
		Packages: identities,
		Cases:    cases,
	}
	canonical, err := canonicalJSON(envelope)
	if err != nil {
		return Result{}, fmt.Errorf("canonicalize Agent Skill evaluation result: %w", err)
	}
	return Result{
		Passed:       passed,
		SuiteVersion: e.suite.SuiteVersion,
		SuiteHash:    e.suiteHash,
		ResultHash:   hashBytes(canonical),
		Cases:        append([]CaseResult(nil), cases...),
	}, nil
}

func validateAndNormalizeSuite(suite *Suite) error {
	if suite.SchemaVersion != SuiteSchemaVersion {
		return fmt.Errorf("Agent Skill evaluation suite schema_version must be %d", SuiteSchemaVersion)
	}
	suite.SuiteVersion = strings.TrimSpace(suite.SuiteVersion)
	if suite.SuiteVersion == "" {
		return errors.New("Agent Skill evaluation suite suite_version is required")
	}
	slices.SortFunc(suite.CompileCases, func(a, b CompileCase) int { return strings.Compare(a.ID, b.ID) })
	slices.SortFunc(suite.RankingCases, func(a, b RankingCase) int { return strings.Compare(a.ID, b.ID) })
	slices.SortFunc(suite.CompositionCases, func(a, b CompositionCase) int { return strings.Compare(a.ID, b.ID) })
	if len(suite.CompileCases) == 0 || len(suite.RankingCases) == 0 || len(suite.CompositionCases) == 0 {
		return errors.New("Agent Skill evaluation suite must contain compile, ranking, and composition cases")
	}
	for index := range suite.CompileCases {
		suite.CompileCases[index].ID = strings.TrimSpace(suite.CompileCases[index].ID)
		suite.CompileCases[index].Fixture = strings.TrimSpace(suite.CompileCases[index].Fixture)
		suite.CompileCases[index].ExpectedErrorCode = strings.TrimSpace(suite.CompileCases[index].ExpectedErrorCode)
		suite.CompileCases[index].ExpectedCompiledHash = strings.ToLower(strings.TrimSpace(suite.CompileCases[index].ExpectedCompiledHash))
		if _, _, err := compileFixture(suite.CompileCases[index].Fixture); err != nil {
			return fmt.Errorf("Agent Skill evaluation suite compile case %q: %w", suite.CompileCases[index].ID, err)
		}
		if !suite.CompileCases[index].ExpectedPass && suite.CompileCases[index].ExpectedErrorCode == "" {
			return fmt.Errorf("Agent Skill evaluation suite compile case %q expected_error_code is required", suite.CompileCases[index].ID)
		}
		if hash := suite.CompileCases[index].ExpectedCompiledHash; hash != "" && !validSHA256(hash) {
			return fmt.Errorf("Agent Skill evaluation suite compile case %q expected_compiled_hash must be a SHA-256 digest", suite.CompileCases[index].ID)
		}
	}
	for index := range suite.RankingCases {
		suite.RankingCases[index].ID = strings.TrimSpace(suite.RankingCases[index].ID)
		suite.RankingCases[index].Query = strings.TrimSpace(suite.RankingCases[index].Query)
		if len(suite.RankingCases[index].ExpectedObjectIDs) != len(suite.RankingCases[index].ExpectedModes) {
			return fmt.Errorf("Agent Skill evaluation suite ranking case %q expected IDs and modes must have equal length", suite.RankingCases[index].ID)
		}
	}
	for index := range suite.CompositionCases {
		suite.CompositionCases[index].ID = strings.TrimSpace(suite.CompositionCases[index].ID)
	}
	seen := map[string]struct{}{}
	for _, entry := range appendCaseIdentities(*suite) {
		if entry.id == "" {
			return fmt.Errorf("Agent Skill evaluation suite %s case id is required", entry.kind)
		}
		if _, exists := seen[entry.id]; exists {
			return fmt.Errorf("Agent Skill evaluation suite case id %q is duplicated", entry.id)
		}
		seen[entry.id] = struct{}{}
	}
	return nil
}

type caseIdentity struct {
	id   string
	kind string
}

func appendCaseIdentities(suite Suite) []caseIdentity {
	result := make([]caseIdentity, 0, len(suite.CompileCases)+len(suite.RankingCases)+len(suite.CompositionCases))
	for _, item := range suite.CompileCases {
		result = append(result, caseIdentity{id: strings.TrimSpace(item.ID), kind: "compile"})
	}
	for _, item := range suite.RankingCases {
		result = append(result, caseIdentity{id: strings.TrimSpace(item.ID), kind: "ranking"})
	}
	for _, item := range suite.CompositionCases {
		result = append(result, caseIdentity{id: strings.TrimSpace(item.ID), kind: "composition"})
	}
	return result
}

func evaluateCompileCase(testCase CompileCase) CaseResult {
	first, second, fixtureErr := compileFixture(testCase.Fixture)
	if fixtureErr != nil {
		return failedResult(testCase.ID, "compile_fixture", "fixture_invalid")
	}
	compiled, err := agentskill.Compile(first)
	actualPass := err == nil
	actualCode := ""
	if err != nil {
		actualCode = agentskill.ErrorCode(err)
	}
	passed := actualPass == testCase.ExpectedPass && actualCode == testCase.ExpectedErrorCode
	if passed && testCase.ExpectedCompiledHash != "" {
		passed = compiled != nil && compiled.CompiledHash == testCase.ExpectedCompiledHash
	}
	if passed && second != nil {
		equivalent, equivalentErr := agentskill.Compile(*second)
		passed = equivalentErr == nil &&
			compiled.CompiledHash == equivalent.CompiledHash &&
			compiled.CanonicalJSON == equivalent.CanonicalJSON &&
			compiled.CompiledMarkdown == equivalent.CompiledMarkdown
	}
	return expectationResult(testCase.ID, "compile_fixture", passed)
}

func compileFixture(name string) (agentskill.PackageDraft, *agentskill.PackageDraft, error) {
	base := evaluationDraft()
	switch strings.TrimSpace(name) {
	case "canonical_equivalence":
		first := base
		first.Manifest.RequiredCapabilities = []string{" search ", "calendar", "search"}
		first.Core.ContentMarkdown = "\r\n  Follow the screening policy.\r\n"
		first.Sections = []agentskill.ReferenceSection{
			evaluationSection("details", 20, "Explain evidence."),
			evaluationSection("examples", 10, "Provide examples."),
		}
		second := base
		second.Manifest.RequiredCapabilities = []string{"calendar", "search"}
		second.Core.ContentMarkdown = "Follow the screening policy.\n"
		second.Sections = []agentskill.ReferenceSection{
			evaluationSection("examples", 10, "Provide examples.\n"),
			evaluationSection("details", 20, "Explain evidence.\n"),
		}
		return first, &second, nil
	case "core_over_budget":
		base.Core.ContentMarkdown = strings.Repeat("a", agentskill.MaxCoreTokens*4+1)
		return base, nil, nil
	case "section_over_budget":
		base.Sections = []agentskill.ReferenceSection{
			evaluationSection("oversized", 1, strings.Repeat("a", agentskill.MaxSectionTokens*4+1)),
		}
		return base, nil, nil
	case "package_over_budget":
		base.Sections = make([]agentskill.ReferenceSection, 0, 11)
		for index := 0; index < 11; index++ {
			base.Sections = append(base.Sections, evaluationSection(
				fmt.Sprintf("section_%02d", index),
				index,
				strings.Repeat("a", agentskill.MaxSectionTokens*4),
			))
		}
		return base, nil, nil
	default:
		return agentskill.PackageDraft{}, nil, fmt.Errorf("unknown fixture %q", name)
	}
}

func evaluationDraft() agentskill.PackageDraft {
	return agentskill.PackageDraft{
		Manifest: agentskill.Manifest{
			SchemaVersion:        agentskill.SchemaVersion,
			SkillName:            "candidate-screening",
			DisplayName:          "Candidate Screening",
			Description:          "Screens candidates against role requirements.",
			AgentType:            "hr_recruiting_agent",
			Category:             "screening",
			Scenario:             "candidate_screening",
			Priority:             100,
			RiskLevel:            agentskill.RiskLevelMedium,
			RequiredCapabilities: []string{"search"},
			TriggerKeywords:      []string{"candidate", "screening"},
			SemanticTags:         []string{"recruiting"},
			Composition:          agentskill.Composition{Role: agentskill.CompositionRolePrimary},
			OutputContract:       agentskill.OutputContract{Mode: agentskill.OutputModeNone},
			EvaluationCriteria:   []string{"grounded"},
		},
		Core: agentskill.Core{ContentMarkdown: "Follow the screening policy."},
	}
}

func evaluationSection(key string, ordinal int, content string) agentskill.ReferenceSection {
	return agentskill.ReferenceSection{
		SectionKey:      key,
		Title:           strings.ToUpper(key[:1]) + strings.ReplaceAll(strings.ReplaceAll(key[1:], "_", " "), "-", " "),
		ContentMarkdown: content,
		TriggerTerms:    []string{"candidate"},
		SemanticTags:    []string{"screening"},
		PlannerIntents:  []string{"evaluate"},
		Ordinal:         ordinal,
	}
}

func evaluateRankingCase(testCase RankingCase) CaseResult {
	candidates := make([]agentskill.RankingCandidate, 0, len(testCase.Candidates))
	for _, item := range testCase.Candidates {
		candidates = append(candidates, agentskill.RankingCandidate{
			Document: agentskill.RankingDocument{
				ObjectID:      item.ObjectID,
				SkillID:       item.SkillID,
				VersionID:     item.VersionID,
				SectionID:     item.SectionID,
				LexicalText:   append([]string(nil), item.LexicalText...),
				MetadataTerms: append([]string(nil), item.MetadataTerms...),
				Priority:      item.Priority,
				VectorScore:   item.VectorScore,
			},
			EmbeddingAvailable: item.EmbeddingAvailable,
		})
	}
	ranked := agentskill.RankCandidates(testCase.Query, candidates)
	if len(ranked) != len(testCase.ExpectedObjectIDs) || len(ranked) != len(testCase.ExpectedModes) {
		return failedResult(testCase.ID, "ranking_fixture", "unexpected_count")
	}
	for index, item := range ranked {
		if item.Document.ObjectID != testCase.ExpectedObjectIDs[index] ||
			string(item.Signals.Mode) != testCase.ExpectedModes[index] {
			return failedResult(testCase.ID, "ranking_fixture", "unexpected_rank")
		}
	}
	return expectationResult(testCase.ID, "ranking_fixture", true)
}

func evaluatePolicy(policy RuntimePolicy) CaseResult {
	passed := strings.TrimSpace(policy.PolicyVersion) == PolicyVersion &&
		policy.MaxSkillTokens > 0 &&
		policy.MaxSkillTokens <= 3000 &&
		policy.MaxInputRatio > 0 &&
		policy.MaxInputRatio <= 0.15 &&
		policy.MaxSkills > 0 &&
		policy.MaxSkills <= 2
	return expectationResult("release-policy", "release_policy", passed)
}

func evaluateReleasePackage(item ReleasePackage) CaseResult {
	id := fmt.Sprintf("release-package-%020d", item.VersionID)
	if item.SkillID <= 0 || item.VersionID <= 0 {
		return failedResult(id, "release_package", "invalid_identity")
	}
	draft := agentskill.PackageDraft{
		Manifest: item.Package.Manifest,
		Core:     item.Package.Core.Core,
		Sections: make([]agentskill.ReferenceSection, 0, len(item.Package.Sections)),
	}
	for _, section := range item.Package.Sections {
		draft.Sections = append(draft.Sections, section.ReferenceSection)
	}
	recompiled, err := agentskill.Compile(draft)
	if err != nil {
		return failedResult(id, "release_package", "compile_failed")
	}
	if recompiled.CompiledHash != strings.ToLower(strings.TrimSpace(item.Package.CompiledHash)) ||
		recompiled.CanonicalJSON != item.Package.CanonicalJSON ||
		recompiled.ManifestJSON != item.Package.ManifestJSON ||
		recompiled.CompiledMarkdown != item.Package.CompiledMarkdown ||
		recompiled.EstimatedTokens != item.Package.EstimatedTokens ||
		recompiled.Core.EstimatedTokens != item.Package.Core.EstimatedTokens ||
		!reflect.DeepEqual(recompiled.Sections, item.Package.Sections) {
		return failedResult(id, "release_package", "canonical_mismatch")
	}
	return expectationResult(id, "release_package", true)
}

func evaluateReleaseComposition(packages []ReleasePackage) CaseResult {
	fixtures := make([]CompositionPackage, 0, len(packages))
	seenVersions := make(map[int64]struct{}, len(packages))
	for _, item := range packages {
		if _, exists := seenVersions[item.VersionID]; exists {
			return failedResult("release-composition", "release_composition", "duplicate_version")
		}
		seenVersions[item.VersionID] = struct{}{}
		fixtures = append(fixtures, CompositionPackage{
			AgentType: item.Package.Manifest.AgentType,
			Scenario:  item.Package.Manifest.Scenario,
			Role:      item.Package.Manifest.Composition.Role,
		})
	}
	if err := validateComposition(fixtures); err != nil {
		return failedResult("release-composition", "release_composition", "composition_invalid")
	}
	return expectationResult("release-composition", "release_composition", true)
}

func validateComposition(packages []CompositionPackage) error {
	type counts struct {
		primary    int
		supporting int
	}
	byScenario := map[string]counts{}
	structuredPrimary := map[string]int{}
	for _, item := range packages {
		key := strings.TrimSpace(item.AgentType) + "\x00" + strings.TrimSpace(item.Scenario)
		current := byScenario[key]
		switch item.Role {
		case agentskill.CompositionRolePrimary:
			current.primary++
			if structuredAgentType(item.AgentType) {
				structuredPrimary[strings.TrimSpace(item.AgentType)]++
			}
		case agentskill.CompositionRoleSupporting:
			current.supporting++
		default:
			return errors.New("invalid composition role")
		}
		byScenario[key] = current
	}
	for _, current := range byScenario {
		if current.primary > 1 || current.supporting > 1 || (current.supporting == 1 && current.primary != 1) {
			return errors.New("invalid Primary and Supporting composition")
		}
	}
	for _, count := range structuredPrimary {
		if count > 1 {
			return errors.New("structured agent type has multiple Primary Skills")
		}
	}
	return nil
}

func structuredAgentType(agentType string) bool {
	switch strings.TrimSpace(agentType) {
	case "resume_profile_extractor", "job_requirement_extractor", "candidate_match_evaluator":
		return true
	default:
		return false
	}
}

func expectationResult(id, kind string, passed bool) CaseResult {
	if passed {
		return CaseResult{ID: strings.TrimSpace(id), Kind: kind, Passed: true, Code: "passed"}
	}
	return failedResult(id, kind, "expectation_mismatch")
}

func failedResult(id, kind, code string) CaseResult {
	return CaseResult{ID: strings.TrimSpace(id), Kind: kind, Passed: false, Code: code}
}

func canonicalJSON(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var decoded any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	return json.Marshal(decoded)
}

func hashBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
