package agentskill

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"

	"smart-recruit-ai-agent-service/internal/application/contextbudget"
)

const (
	MaxCoreTokens       = 800
	MaxSectionTokens    = 1200
	MaxSections         = 20
	MaxPackageTokens    = 12000
	maxSkillNameLength  = 128
	maxSectionKeyLength = 128
)

var (
	skillNamePattern  = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
	sectionKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,127}$`)
)

type canonicalPackage struct {
	Manifest Manifest           `json:"manifest"`
	Core     Core               `json:"core"`
	Sections []ReferenceSection `json:"sections"`
}

func Compile(draft PackageDraft) (*CompiledPackage, error) {
	manifest, err := normalizeManifest(draft.Manifest)
	if err != nil {
		return nil, err
	}

	core := Core{ContentMarkdown: normalizeMarkdown(draft.Core.ContentMarkdown)}
	if core.ContentMarkdown == "" {
		return nil, compileError(CodePackageInvalid, "core.content_markdown", "is required")
	}
	coreTokens := contextbudget.EstimateTokensConservative(core.ContentMarkdown)
	if coreTokens > MaxCoreTokens {
		return nil, compileError(
			CodeCoreBudgetExceeded,
			"core.content_markdown",
			fmt.Sprintf("estimated tokens %d exceed limit %d", coreTokens, MaxCoreTokens),
		)
	}

	sections, _, err := normalizeSections(draft.Sections)
	if err != nil {
		return nil, err
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, compileError(CodePackageInvalid, "manifest", "cannot encode canonical manifest")
	}
	compiledMarkdown := renderCompiledMarkdown(manifest, string(manifestBytes), core.ContentMarkdown, sections)
	totalTokens := estimateCompiledArtifactTokens(compiledMarkdown)
	if totalTokens > MaxPackageTokens {
		return nil, compileError(
			CodePackageInvalid,
			"package",
			fmt.Sprintf("estimated compiled artifact tokens %d exceed limit %d", totalTokens, MaxPackageTokens),
		)
	}
	envelope := canonicalPackage{
		Manifest: manifest,
		Core:     core,
		Sections: normalizedReferenceSections(sections),
	}
	canonicalBytes, err := json.Marshal(envelope)
	if err != nil {
		return nil, compileError(CodePackageInvalid, "package", "cannot encode canonical package")
	}
	sum := sha256.Sum256(canonicalBytes)

	return &CompiledPackage{
		Manifest:         manifest,
		Core:             CompiledCore{Core: core, EstimatedTokens: coreTokens},
		Sections:         sections,
		ManifestJSON:     string(manifestBytes),
		CanonicalJSON:    string(canonicalBytes),
		CompiledMarkdown: compiledMarkdown,
		CompiledHash:     hex.EncodeToString(sum[:]),
		EstimatedTokens:  totalTokens,
	}, nil
}

func estimateCompiledArtifactTokens(compiledMarkdown string) int {
	return contextbudget.EstimateTokensConservative(compiledMarkdown)
}

func normalizedReferenceSections(sections []CompiledSection) []ReferenceSection {
	output := make([]ReferenceSection, 0, len(sections))
	for _, section := range sections {
		output = append(output, section.ReferenceSection)
	}
	return output
}

func normalizeManifest(input Manifest) (Manifest, error) {
	output := input
	output.SkillName = strings.TrimSpace(output.SkillName)
	output.DisplayName = strings.TrimSpace(output.DisplayName)
	output.Description = strings.TrimSpace(output.Description)
	output.AgentType = strings.TrimSpace(output.AgentType)
	output.Category = strings.TrimSpace(output.Category)
	output.Scenario = strings.TrimSpace(output.Scenario)
	output.RequiredCapabilities = normalizeSet(output.RequiredCapabilities)
	output.TriggerKeywords = normalizeSet(output.TriggerKeywords)
	output.SemanticTags = normalizeSet(output.SemanticTags)
	output.EvaluationCriteria = normalizeSet(output.EvaluationCriteria)
	output.OutputContract.SchemaID = strings.TrimSpace(output.OutputContract.SchemaID)

	if output.SchemaVersion != SchemaVersion {
		return Manifest{}, compileError(CodePackageInvalid, "manifest.schema_version", "must be 2")
	}
	if output.SkillName == "" {
		return Manifest{}, compileError(CodePackageInvalid, "manifest.skill_name", "is required")
	}
	if len(output.SkillName) > maxSkillNameLength || !skillNamePattern.MatchString(output.SkillName) {
		return Manifest{}, compileError(
			CodePackageInvalid,
			"manifest.skill_name",
			"must start with a lowercase letter and contain only lowercase letters, digits, underscores, or hyphens",
		)
	}
	if output.DisplayName == "" {
		return Manifest{}, compileError(CodePackageInvalid, "manifest.display_name", "is required")
	}
	if output.AgentType == "" {
		return Manifest{}, compileError(CodePackageInvalid, "manifest.agent_type", "is required")
	}

	derivedPolicy, ok := activationPolicyForRisk(output.RiskLevel)
	if !ok {
		return Manifest{}, compileError(CodePackageInvalid, "manifest.risk_level", "must be low, medium, high, or critical")
	}
	if output.ActivationPolicy != "" && output.ActivationPolicy != derivedPolicy {
		return Manifest{}, compileError(
			CodePackageInvalid,
			"manifest.activation_policy",
			fmt.Sprintf("must be %q for risk level %q", derivedPolicy, output.RiskLevel),
		)
	}
	output.ActivationPolicy = derivedPolicy

	switch output.Composition.Role {
	case CompositionRolePrimary, CompositionRoleSupporting:
	default:
		return Manifest{}, compileError(CodePackageInvalid, "manifest.composition.role", "must be primary or supporting")
	}

	contract, contractDefined, err := normalizeOutputContract(output.OutputContract)
	if err != nil {
		return Manifest{}, err
	}
	if output.Composition.Role == CompositionRoleSupporting && contractDefined {
		return Manifest{}, compileError(
			CodeCompositionConflict,
			"manifest.output_contract",
			"supporting skills cannot define an output contract",
		)
	}
	output.OutputContract = contract
	return output, nil
}

func normalizeOutputContract(input OutputContract) (OutputContract, bool, error) {
	output := input
	if output.Mode == "" {
		output.Mode = OutputModeNone
	}
	switch output.Mode {
	case OutputModeNone, OutputModeAdvisory, OutputModeStrict:
	default:
		return OutputContract{}, false, compileError(
			CodePackageInvalid,
			"manifest.output_contract.mode",
			"must be none, advisory, or strict",
		)
	}

	rawSchema := bytes.TrimSpace(output.Schema)
	schemaDefined := len(rawSchema) > 0 && !bytes.Equal(rawSchema, []byte("null"))
	if output.Mode == OutputModeNone {
		if output.SchemaID != "" || schemaDefined {
			return OutputContract{}, false, compileError(
				CodePackageInvalid,
				"manifest.output_contract",
				"mode none cannot define schema_id or schema",
			)
		}
		output.Schema = nil
		return output, false, nil
	}
	if !schemaDefined {
		return OutputContract{}, false, compileError(
			CodePackageInvalid,
			"manifest.output_contract.schema",
			"is required for advisory or strict mode",
		)
	}
	schema, err := normalizeJSONObject(rawSchema)
	if err != nil {
		return OutputContract{}, false, compileError(
			CodePackageInvalid,
			"manifest.output_contract.schema",
			err.Error(),
		)
	}
	if output.Mode == OutputModeStrict && output.SchemaID == "" {
		return OutputContract{}, false, compileError(
			CodePackageInvalid,
			"manifest.output_contract.schema_id",
			"is required for strict mode",
		)
	}
	output.Schema = schema
	return output, true, nil
}

func normalizeSections(input []ReferenceSection) ([]CompiledSection, int, error) {
	if len(input) > MaxSections {
		return nil, 0, compileError(
			CodeSectionInvalid,
			"sections",
			fmt.Sprintf("count %d exceeds limit %d", len(input), MaxSections),
		)
	}

	seen := make(map[string]struct{}, len(input))
	output := make([]CompiledSection, 0, len(input))
	totalTokens := 0
	for i, section := range input {
		field := fmt.Sprintf("sections[%d]", i)
		section.SectionKey = strings.TrimSpace(section.SectionKey)
		section.Title = strings.TrimSpace(section.Title)
		section.Description = strings.TrimSpace(section.Description)
		section.ContentMarkdown = normalizeMarkdown(section.ContentMarkdown)
		section.TriggerTerms = normalizeSet(section.TriggerTerms)
		section.SemanticTags = normalizeSet(section.SemanticTags)
		section.PlannerIntents = normalizeSet(section.PlannerIntents)

		if section.SectionKey == "" ||
			len(section.SectionKey) > maxSectionKeyLength ||
			!sectionKeyPattern.MatchString(section.SectionKey) {
			return nil, 0, compileError(
				CodeSectionInvalid,
				field+".section_key",
				"must start with a lowercase letter and contain only lowercase letters, digits, underscores, or hyphens",
			)
		}
		if _, exists := seen[section.SectionKey]; exists {
			return nil, 0, compileError(CodeSectionInvalid, field+".section_key", "must be unique")
		}
		seen[section.SectionKey] = struct{}{}
		if section.Title == "" {
			return nil, 0, compileError(CodeSectionInvalid, field+".title", "is required")
		}
		if section.Ordinal < 0 {
			return nil, 0, compileError(CodeSectionInvalid, field+".ordinal", "must not be negative")
		}
		if section.ContentMarkdown == "" {
			return nil, 0, compileError(CodeSectionInvalid, field+".content_markdown", "is required")
		}

		tokens := contextbudget.EstimateTokensConservative(section.ContentMarkdown)
		if tokens > MaxSectionTokens {
			return nil, 0, compileError(
				CodeSectionInvalid,
				field+".content_markdown",
				fmt.Sprintf("estimated tokens %d exceed limit %d", tokens, MaxSectionTokens),
			)
		}
		sum := sha256.Sum256([]byte(section.ContentMarkdown))
		output = append(output, CompiledSection{
			ReferenceSection: section,
			EstimatedTokens:  tokens,
			ContentHash:      hex.EncodeToString(sum[:]),
		})
		totalTokens += tokens
	}

	slices.SortFunc(output, func(a, b CompiledSection) int {
		if a.Ordinal != b.Ordinal {
			return cmp.Compare(a.Ordinal, b.Ordinal)
		}
		return strings.Compare(a.SectionKey, b.SectionKey)
	})
	return output, totalTokens, nil
}

func activationPolicyForRisk(risk RiskLevel) (ActivationPolicy, bool) {
	switch risk {
	case RiskLevelLow, RiskLevelMedium:
		return ActivationPolicyAuto, true
	case RiskLevelHigh:
		return ActivationPolicyConfirm, true
	case RiskLevelCritical:
		return ActivationPolicyManualOnly, true
	default:
		return "", false
	}
}

func normalizeSet(values []string) []string {
	unique := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			unique[value] = struct{}{}
		}
	}
	output := make([]string, 0, len(unique))
	for value := range unique {
		output = append(output, value)
	}
	slices.Sort(output)
	return output
}

func normalizeMarkdown(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return value + "\n"
}

func normalizeJSONObject(raw []byte) (json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("must be valid JSON: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, err
	}
	if _, ok := value.(map[string]any); !ok {
		return nil, fmt.Errorf("must be a JSON object")
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("cannot encode canonical JSON")
	}
	return canonical, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("must contain exactly one JSON value: %w", err)
	}
	return fmt.Errorf("must contain exactly one JSON value")
}

func renderCompiledMarkdown(manifest Manifest, manifestJSON, core string, sections []CompiledSection) string {
	var builder strings.Builder
	builder.WriteString("<!-- agent-skill-package-manifest\n")
	builder.WriteString(manifestJSON)
	builder.WriteString("\n-->\n\n# ")
	builder.WriteString(manifest.DisplayName)
	builder.WriteString("\n\n## Core\n\n")
	builder.WriteString(core)
	if len(sections) > 0 {
		builder.WriteString("\n## Reference Sections\n")
	}
	for _, section := range sections {
		builder.WriteString("\n### ")
		builder.WriteString(section.Title)
		builder.WriteString(" (`")
		builder.WriteString(section.SectionKey)
		builder.WriteString("`)\n\n")
		if section.Description != "" {
			builder.WriteString(section.Description)
			builder.WriteString("\n\n")
		}
		builder.WriteString(section.ContentMarkdown)
	}
	return normalizeMarkdown(builder.String())
}
