package recruiting_intelligence

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"smart-recruit-ai-agent-service/internal/domain/agentskill"
)

const (
	strictRepairOutputMaxRunes = 8192
	strictSummaryMaxRunes      = 512
)

var (
	ErrStrictContractInvalid          = errors.New("strict output contract is invalid")
	ErrStrictPackageLoaderUnavailable = errors.New("strict output package loader is unavailable")
	ErrStrictOutputInvalid            = errors.New("strict structured output is invalid")
	ErrStrictActivationDenied         = errors.New("strict output contract cannot run unattended")
)

type StrictOutputErrorKind string

const (
	StrictOutputContractError StrictOutputErrorKind = "contract"
	StrictOutputValidation    StrictOutputErrorKind = "validation"
	StrictOutputDomain        StrictOutputErrorKind = "domain"
)

// StrictOutputError is intentionally safe for RPCs, logs, and observations.
// Its string form never contains package/version/schema identifiers, schema
// content, model output, recruiting source text, or validator details.
type StrictOutputError struct {
	Kind  StrictOutputErrorKind
	Cause error
}

func (e *StrictOutputError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("recruiting strict output failed (%s)", e.Kind)
}

func (e *StrictOutputError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func IsStrictOutputError(err error) bool {
	var strictErr *StrictOutputError
	if errors.As(err, &strictErr) {
		return true
	}
	var runtimeErr *RuntimeError
	return errors.As(err, &runtimeErr) && runtimeErr.StrictContractApplied
}

func strictDomainError(err error) error {
	if err == nil {
		return nil
	}
	return &StrictOutputError{Kind: StrictOutputDomain, Cause: err}
}

type ReleasedAgentSkillSection struct {
	ID              int64
	SectionKey      string
	Title           string
	Description     string
	ContentMarkdown string
	TriggerTerms    []string
	SemanticTags    []string
	PlannerIntents  []string
	Priority        int
	Ordinal         int
	EstimatedTokens int
	ContentHash     string
}

type ReleasedAgentSkillPackage struct {
	ID                  int64
	SkillID             int64
	Version             string
	ManifestJSON        string
	CoreMarkdown        string
	CompiledMarkdown    string
	CompiledHash        string
	CoreEstimatedTokens int
	Sections            []ReleasedAgentSkillSection
}

type ReleasedAgentSkillPackageLoader interface {
	LoadReleasedAgentSkillPackages(context.Context, []int64) ([]ReleasedAgentSkillPackage, error)
}

type StrictContractResolver struct {
	loader ReleasedAgentSkillPackageLoader
}

func NewStrictContractResolver(loader ReleasedAgentSkillPackageLoader) *StrictContractResolver {
	return &StrictContractResolver{loader: loader}
}

type resolvedStrictContract struct {
	schemaID   string
	schemaJSON string
	schema     *agentskill.StrictOutputSchema
	risk       agentskill.RiskLevel
}

func (r *StrictContractResolver) Resolve(ctx context.Context, agentType string) (*resolvedStrictContract, error) {
	if !CapabilityRuntimeSkillPackageV2Enabled(ctx) {
		return nil, nil
	}
	versionIDs, err := exactPositiveIDs(CapabilityRuntimeAgentSkillVersionIDs(ctx))
	if err != nil {
		return nil, strictContractError(err)
	}
	if len(versionIDs) == 0 {
		return nil, nil
	}
	if r == nil || r.loader == nil {
		return nil, strictContractError(ErrStrictPackageLoaderUnavailable)
	}
	packages, err := r.loader.LoadReleasedAgentSkillPackages(ctx, versionIDs)
	if err != nil {
		return nil, strictContractError(errors.New("package lookup failed"))
	}
	allowed := make(map[int64]struct{}, len(versionIDs))
	for _, id := range versionIDs {
		allowed[id] = struct{}{}
	}
	loaded := make(map[int64]struct{}, len(packages))
	var matching []*agentskill.CompiledPackage
	for _, runtimePackage := range packages {
		if _, ok := allowed[runtimePackage.ID]; !ok {
			return nil, strictContractError(errors.New("unexpected package version"))
		}
		if _, duplicate := loaded[runtimePackage.ID]; duplicate {
			return nil, strictContractError(errors.New("duplicate package version"))
		}
		loaded[runtimePackage.ID] = struct{}{}
		compiled, compileErr := validateReleasedAgentSkillPackage(runtimePackage)
		if compileErr != nil {
			return nil, strictContractError(compileErr)
		}
		manifest := compiled.Manifest
		if manifest.AgentType == agentType && manifest.Composition.Role == agentskill.CompositionRolePrimary {
			matching = append(matching, compiled)
		}
	}
	if len(loaded) != len(versionIDs) {
		return nil, strictContractError(errors.New("released package version missing"))
	}
	if len(matching) == 0 {
		return nil, nil
	}
	if len(matching) != 1 {
		return nil, strictContractError(errors.New("multiple matching primary packages"))
	}
	selected := matching[0]
	manifest := selected.Manifest
	if manifest.OutputContract.Mode != agentskill.OutputModeStrict {
		return nil, nil
	}
	if manifest.ActivationPolicy != agentskill.ActivationPolicyAuto ||
		(manifest.RiskLevel != agentskill.RiskLevelLow && manifest.RiskLevel != agentskill.RiskLevelMedium) {
		return nil, strictContractError(ErrStrictActivationDenied)
	}
	schema := selected.StrictSchema
	if schema == nil {
		return nil, strictContractError(errors.New("compiled strict schema unavailable"))
	}
	return &resolvedStrictContract{
		schemaID:   manifest.OutputContract.SchemaID,
		schemaJSON: string(manifest.OutputContract.Schema),
		schema:     schema,
		risk:       manifest.RiskLevel,
	}, nil
}

func strictContractError(cause error) error {
	return &StrictOutputError{Kind: StrictOutputContractError, Cause: errors.Join(ErrStrictContractInvalid, cause)}
}

func exactPositiveIDs(input []int64) ([]int64, error) {
	if len(input) == 0 {
		return nil, nil
	}
	output := append([]int64(nil), input...)
	seen := make(map[int64]struct{}, len(output))
	for _, id := range output {
		if id <= 0 {
			return nil, errors.New("invalid version id")
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, errors.New("duplicate version id")
		}
		seen[id] = struct{}{}
	}
	sort.Slice(output, func(i, j int) bool { return output[i] < output[j] })
	return output, nil
}

func validateReleasedAgentSkillPackage(runtimePackage ReleasedAgentSkillPackage) (*agentskill.CompiledPackage, error) {
	manifest, storedManifest, err := agentskill.DecodeManifestJSON([]byte(runtimePackage.ManifestJSON))
	if err != nil {
		return nil, err
	}
	draft := agentskill.PackageDraft{
		Manifest: manifest,
		Core:     agentskill.Core{ContentMarkdown: runtimePackage.CoreMarkdown},
		Sections: make([]agentskill.ReferenceSection, 0, len(runtimePackage.Sections)),
	}
	for _, section := range runtimePackage.Sections {
		draft.Sections = append(draft.Sections, agentskill.ReferenceSection{
			SectionKey: section.SectionKey, Title: section.Title, Description: section.Description,
			ContentMarkdown: section.ContentMarkdown, TriggerTerms: append([]string(nil), section.TriggerTerms...),
			SemanticTags: append([]string(nil), section.SemanticTags...), PlannerIntents: append([]string(nil), section.PlannerIntents...),
			Priority: section.Priority, Ordinal: section.Ordinal,
		})
	}
	compiled, err := agentskill.Compile(draft)
	if err != nil {
		return nil, err
	}
	_, canonicalManifest, err := agentskill.DecodeManifestJSON([]byte(compiled.ManifestJSON))
	if err != nil {
		return nil, err
	}
	if runtimePackage.ID <= 0 || runtimePackage.SkillID <= 0 ||
		strings.TrimSpace(runtimePackage.Version) == "" ||
		!bytes.Equal(storedManifest, canonicalManifest) ||
		runtimePackage.CoreMarkdown != compiled.Core.ContentMarkdown ||
		runtimePackage.CompiledMarkdown != compiled.CompiledMarkdown ||
		runtimePackage.CompiledHash != compiled.CompiledHash ||
		runtimePackage.CoreEstimatedTokens != compiled.Core.EstimatedTokens ||
		len(runtimePackage.Sections) != len(compiled.Sections) {
		return nil, errors.New("package canonical fields mismatch")
	}
	for index, expected := range compiled.Sections {
		stored := runtimePackage.Sections[index]
		if stored.ID <= 0 ||
			stored.SectionKey != expected.SectionKey ||
			stored.Title != expected.Title ||
			stored.Description != expected.Description ||
			stored.ContentMarkdown != expected.ContentMarkdown ||
			!reflect.DeepEqual(stored.TriggerTerms, expected.TriggerTerms) ||
			!reflect.DeepEqual(stored.SemanticTags, expected.SemanticTags) ||
			!reflect.DeepEqual(stored.PlannerIntents, expected.PlannerIntents) ||
			stored.Priority != expected.Priority ||
			stored.Ordinal != expected.Ordinal ||
			stored.EstimatedTokens != expected.EstimatedTokens ||
			stored.ContentHash != expected.ContentHash {
			return nil, errors.New("package section integrity mismatch")
		}
	}
	return compiled, nil
}

func strictSystemDirective(contract *resolvedStrictContract) string {
	if contract == nil {
		return ""
	}
	schemaID, _ := json.Marshal(contract.schemaID)
	return "\n\n<agent-skill-strict-output>\n" +
		"Return exactly one JSON value and no markdown fence or surrounding prose.\n" +
		"Schema ID (JSON string): " + string(schemaID) + "\n" +
		"JSON Schema: " + contract.schemaJSON + "\n" +
		"</agent-skill-strict-output>"
}

func strictRepairPrompt(previousOutput, summary string) string {
	return "Repair the previous response so it is exactly one JSON value matching the system JSON Schema. " +
		"Return JSON only.\nValidation summary: " + truncateRunes(summary, strictSummaryMaxRunes) +
		"\nPrevious response:\n" + truncateRunes(previousOutput, strictRepairOutputMaxRunes)
}

func truncateRunes(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

type StrictValidationOutcome struct {
	Result string
	Reason string
	Risk   string
}

type StrictValidationCollector struct {
	mu       sync.Mutex
	outcomes []StrictValidationOutcome
}

type strictValidationCollectorContextKey struct{}

func WithStrictValidationCollector(ctx context.Context) (context.Context, *StrictValidationCollector) {
	collector := &StrictValidationCollector{}
	return context.WithValue(ctx, strictValidationCollectorContextKey{}, collector), collector
}

func (c *StrictValidationCollector) add(outcome StrictValidationOutcome) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.outcomes = append(c.outcomes, outcome)
	c.mu.Unlock()
}

func (c *StrictValidationCollector) Outcomes() []StrictValidationOutcome {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]StrictValidationOutcome(nil), c.outcomes...)
}

func recordStrictValidationOutcome(ctx context.Context, contract *resolvedStrictContract, valid bool) {
	collector, _ := ctx.Value(strictValidationCollectorContextKey{}).(*StrictValidationCollector)
	if collector == nil {
		return
	}
	outcome := StrictValidationOutcome{Result: "failed", Reason: "strict_invalid", Risk: "unknown"}
	if contract != nil {
		outcome.Risk = string(contract.risk)
	}
	if valid {
		outcome.Result = "passed"
		outcome.Reason = "strict_valid"
	}
	collector.add(outcome)
}
