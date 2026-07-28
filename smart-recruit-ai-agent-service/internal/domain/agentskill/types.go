package agentskill

import "encoding/json"

const SchemaVersion = 2

type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

type ActivationPolicy string

const (
	ActivationPolicyAuto       ActivationPolicy = "auto"
	ActivationPolicyConfirm    ActivationPolicy = "confirm"
	ActivationPolicyManualOnly ActivationPolicy = "manual_only"
)

type CompositionRole string

const (
	CompositionRolePrimary    CompositionRole = "primary"
	CompositionRoleSupporting CompositionRole = "supporting"
)

type OutputMode string

const (
	OutputModeNone     OutputMode = "none"
	OutputModeAdvisory OutputMode = "advisory"
	OutputModeStrict   OutputMode = "strict"
)

type Composition struct {
	Role CompositionRole `json:"role"`
}

type OutputContract struct {
	Mode     OutputMode      `json:"mode"`
	SchemaID string          `json:"schema_id,omitempty"`
	Schema   json.RawMessage `json:"schema,omitempty"`
}

type Manifest struct {
	SchemaVersion        int              `json:"schema_version"`
	SkillName            string           `json:"skill_name"`
	DisplayName          string           `json:"display_name"`
	Description          string           `json:"description,omitempty"`
	AgentType            string           `json:"agent_type"`
	Category             string           `json:"category,omitempty"`
	Scenario             string           `json:"scenario,omitempty"`
	Priority             int              `json:"priority"`
	RiskLevel            RiskLevel        `json:"risk_level"`
	ActivationPolicy     ActivationPolicy `json:"activation_policy"`
	RequiredCapabilities []string         `json:"required_capabilities"`
	TriggerKeywords      []string         `json:"trigger_keywords"`
	SemanticTags         []string         `json:"semantic_tags"`
	Composition          Composition      `json:"composition"`
	OutputContract       OutputContract   `json:"output_contract"`
	EvaluationCriteria   []string         `json:"evaluation_criteria"`
}

type Core struct {
	ContentMarkdown string `json:"content_markdown"`
}

type ReferenceSection struct {
	SectionKey      string   `json:"section_key"`
	Title           string   `json:"title"`
	Description     string   `json:"description,omitempty"`
	ContentMarkdown string   `json:"content_markdown"`
	TriggerTerms    []string `json:"trigger_terms"`
	SemanticTags    []string `json:"semantic_tags"`
	PlannerIntents  []string `json:"planner_intents"`
	Priority        int      `json:"priority"`
	Ordinal         int      `json:"ordinal"`
}

type PackageDraft struct {
	Manifest Manifest           `json:"manifest"`
	Core     Core               `json:"core"`
	Sections []ReferenceSection `json:"sections"`
}

type CompiledCore struct {
	Core
	EstimatedTokens int `json:"estimated_tokens"`
}

type CompiledSection struct {
	ReferenceSection
	EstimatedTokens int    `json:"estimated_tokens"`
	ContentHash     string `json:"content_hash"`
}

type CompiledPackage struct {
	Manifest         Manifest          `json:"manifest"`
	Core             CompiledCore      `json:"core"`
	Sections         []CompiledSection `json:"sections"`
	ManifestJSON     string            `json:"manifest_json"`
	CanonicalJSON    string            `json:"canonical_json"`
	CompiledMarkdown string            `json:"compiled_markdown"`
	CompiledHash     string            `json:"compiled_hash"`
	EstimatedTokens  int               `json:"estimated_tokens"`
}
