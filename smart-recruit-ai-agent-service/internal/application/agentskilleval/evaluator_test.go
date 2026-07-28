package agentskilleval

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"smart-recruit-ai-agent-service/internal/domain/agentskill"
)

const (
	goldenSuiteHash  = "00b1e6f3f12027bde1d56ebd2feb64b7e7c63bffaafe79340a8ede00da1607a5"
	goldenResultHash = "f12d6fdff1d2891143fd4b03c2267ab7f8bedf9b16d41521a54ff037c66c22df"
)

func TestDefaultEvaluatorGoldenHashesAndDeterminism(t *testing.T) {
	evaluator, err := NewDefaultEvaluator()
	if err != nil {
		t.Fatalf("NewDefaultEvaluator() error = %v", err)
	}
	input := validInput(t)
	first, err := evaluator.Evaluate(context.Background(), input)
	if err != nil {
		t.Fatalf("Evaluate(first) error = %v", err)
	}
	second, err := evaluator.Evaluate(context.Background(), input)
	if err != nil {
		t.Fatalf("Evaluate(second) error = %v", err)
	}
	if !first.Passed {
		t.Fatalf("default suite failed: %+v", first.Cases)
	}
	if first.SuiteHash != goldenSuiteHash {
		t.Fatalf("suite hash = %s, want %s", first.SuiteHash, goldenSuiteHash)
	}
	if first.ResultHash != goldenResultHash {
		t.Fatalf("result hash = %s, want %s", first.ResultHash, goldenResultHash)
	}
	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("results differ:\n%s\n%s", firstJSON, secondJSON)
	}
}

func TestEvaluatorCanonicalizesPackageOrderAndDetectsMutation(t *testing.T) {
	evaluator, err := NewDefaultEvaluator()
	if err != nil {
		t.Fatalf("NewDefaultEvaluator() error = %v", err)
	}
	input := validInput(t)
	supporting := compilePackage(t, "candidate-evidence", agentskill.CompositionRoleSupporting)
	input.Packages = append(input.Packages, ReleasePackage{SkillID: 2, VersionID: 20, Package: supporting})
	first, err := evaluator.Evaluate(context.Background(), input)
	if err != nil || !first.Passed {
		t.Fatalf("Evaluate(first) = %+v, %v", first, err)
	}

	input.Packages[0], input.Packages[1] = input.Packages[1], input.Packages[0]
	reordered, err := evaluator.Evaluate(context.Background(), input)
	if err != nil || !reordered.Passed {
		t.Fatalf("Evaluate(reordered) = %+v, %v", reordered, err)
	}
	if reordered.ResultHash != first.ResultHash {
		t.Fatalf("package order changed result hash: %s != %s", reordered.ResultHash, first.ResultHash)
	}

	input.Packages[0].VersionID = 21
	renumbered, err := evaluator.Evaluate(context.Background(), input)
	if err != nil || !renumbered.Passed {
		t.Fatalf("Evaluate(renumbered) = %+v, %v", renumbered, err)
	}
	if renumbered.ResultHash == first.ResultHash {
		t.Fatal("exact version ID mutation did not change result hash")
	}

	input.Packages[0].VersionID = 20
	input.Packages[0].Package.CompiledHash = strings.Repeat("f", 64)
	mutated, err := evaluator.Evaluate(context.Background(), input)
	if err != nil {
		t.Fatalf("Evaluate(mutated) error = %v", err)
	}
	if mutated.Passed || mutated.ResultHash == first.ResultHash {
		t.Fatalf("mutated result = %+v, want failed with a distinct hash", mutated)
	}
}

func TestEvaluatorReturnsPassedFalseWhenSuiteExpectationMutates(t *testing.T) {
	raw, err := suiteFiles.ReadFile("testdata/suite-v1.json")
	if err != nil {
		t.Fatalf("read suite: %v", err)
	}
	mutated := strings.Replace(string(raw), `"expected_object_ids": [1]`, `"expected_object_ids": [2]`, 1)
	evaluator, err := NewEvaluator([]byte(mutated))
	if err != nil {
		t.Fatalf("NewEvaluator(mutated) error = %v", err)
	}
	result, err := evaluator.Evaluate(context.Background(), validInput(t))
	if err != nil {
		t.Fatalf("Evaluate(mutated suite) error = %v", err)
	}
	if result.Passed {
		t.Fatalf("mutated suite unexpectedly passed: %+v", result.Cases)
	}
	if !containsFailedCase(result.Cases, "ranking-hybrid") {
		t.Fatalf("mutated suite failure not reported: %+v", result.Cases)
	}
}

func TestEvaluatorSuiteHashIgnoresJSONFormattingAndCaseOrder(t *testing.T) {
	raw, err := suiteFiles.ReadFile("testdata/suite-v1.json")
	if err != nil {
		t.Fatalf("read suite: %v", err)
	}
	first, err := NewEvaluator(raw)
	if err != nil {
		t.Fatalf("NewEvaluator(first) error = %v", err)
	}
	var suite Suite
	if err := json.Unmarshal(raw, &suite); err != nil {
		t.Fatalf("decode suite: %v", err)
	}
	for left, right := 0, len(suite.RankingCases)-1; left < right; left, right = left+1, right-1 {
		suite.RankingCases[left], suite.RankingCases[right] = suite.RankingCases[right], suite.RankingCases[left]
	}
	reordered, err := json.MarshalIndent(suite, "", "    ")
	if err != nil {
		t.Fatalf("encode reordered suite: %v", err)
	}
	second, err := NewEvaluator(reordered)
	if err != nil {
		t.Fatalf("NewEvaluator(second) error = %v", err)
	}
	firstResult, _ := first.Evaluate(context.Background(), validInput(t))
	secondResult, _ := second.Evaluate(context.Background(), validInput(t))
	if firstResult.SuiteHash != secondResult.SuiteHash {
		t.Fatalf("suite hashes differ: %s != %s", firstResult.SuiteHash, secondResult.SuiteHash)
	}
}

func TestEvaluatorRejectsReleaseCompositionConflict(t *testing.T) {
	evaluator, err := NewDefaultEvaluator()
	if err != nil {
		t.Fatalf("NewDefaultEvaluator() error = %v", err)
	}
	input := validInput(t)
	second := compilePackage(t, "candidate-ranking", agentskill.CompositionRolePrimary)
	input.Packages = append(input.Packages, ReleasePackage{SkillID: 2, VersionID: 20, Package: second})
	result, err := evaluator.Evaluate(context.Background(), input)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.Passed || !containsFailedCase(result.Cases, "release-composition") {
		t.Fatalf("composition conflict result = %+v", result)
	}
}

func TestEvaluatorHonorsCanceledContext(t *testing.T) {
	evaluator, err := NewDefaultEvaluator()
	if err != nil {
		t.Fatalf("NewDefaultEvaluator() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := evaluator.Evaluate(ctx, validInput(t)); err == nil {
		t.Fatal("Evaluate(canceled) error = nil")
	}
}

func validInput(t *testing.T) Input {
	t.Helper()
	compiled := compilePackage(t, "candidate-screening", agentskill.CompositionRolePrimary)
	return Input{
		CapabilityKey: "ai.chat",
		Audience:      "tenant_hr",
		Policy: RuntimePolicy{
			PolicyVersion:  PolicyVersion,
			MaxSkillTokens: 3000,
			MaxInputRatio:  0.15,
			MaxSkills:      2,
		},
		Packages: []ReleasePackage{{SkillID: 1, VersionID: 10, Package: compiled}},
	}
}

func compilePackage(t *testing.T, name string, role agentskill.CompositionRole) agentskill.CompiledPackage {
	t.Helper()
	draft := evaluationDraft()
	draft.Manifest.SkillName = name
	draft.Manifest.DisplayName = name
	draft.Manifest.Composition.Role = role
	compiled, err := agentskill.Compile(draft)
	if err != nil {
		t.Fatalf("Compile(%s) error = %v", name, err)
	}
	return *compiled
}

func containsFailedCase(cases []CaseResult, id string) bool {
	for _, item := range cases {
		if item.ID == id && !item.Passed {
			return true
		}
	}
	return false
}
