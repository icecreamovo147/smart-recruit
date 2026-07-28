package model

import "time"

const (
	MCPTransportStdio = "stdio"
	MCPTransportSSE   = "sse"
	MCPTransportHTTP  = "http"

	MCPPolicyDecisionAllow                = "allow"
	MCPPolicyDecisionDeny                 = "deny"
	MCPPolicyDecisionConfirmationRequired = "confirmation_required"
	MCPPolicyDecisionRateLimited          = "rate_limited"

	EmbeddingStatusAvailable   = "available"
	EmbeddingStatusUnavailable = "unavailable"

	SkillRuntimePrompt   = "prompt"
	SkillRuntimeTool     = "tool"
	SkillRuntimeWorkflow = "workflow"
	SkillRuntimeHTTP     = "http"

	MatchStatusStrongMatch  = "strong_match"
	MatchStatusMatch        = "match"
	MatchStatusPartialMatch = "partial_match"
	MatchStatusWeakEvidence = "weak_evidence"
	MatchStatusMissing      = "missing"
	MatchStatusConflict     = "conflict"
	RequirementMustHave     = "must_have"
	RequirementNiceToHave   = "nice_to_have"
	RequirementSoftSkill    = "soft_skill"
	RequirementCoreSkill    = "core_skill"
	RequirementExperience   = "experience"
	RecommendationStrong    = "strong_recommend"
	RecommendationRecommend = "recommend"
	RecommendationReview    = "review"
	RecommendationNot       = "not_recommend"
	RecommendationStrongNot = "strong_not_recommend"
)

type MCPServerConfig struct {
	ID                  uint64
	Name                string
	Transport           string
	Command             string
	URL                 string
	TimeoutSeconds      int
	Enabled             bool
	AllowPrivateNetwork bool
	AllowedCommands     []string
}

type MCPToolPolicy struct {
	ID                     uint64
	ServerID               uint64
	ToolName               string
	Enabled                bool
	Effect                 string
	RiskLevel              string
	AllowedRoles           []string
	AllowedScopes          []string
	RequiredArgs           []string
	DeniedArgs             []string
	ArgRules               map[string]MCPArgRule
	RedactFields           []string
	RequireConfirmation    bool
	RateLimitWindowSeconds int
	RateLimitMaxCalls      int
}

type MCPArgRule struct {
	Required bool
	Deny     bool
	Enum     []string
	Regex    string
	Min      *float64
	Max      *float64
}

type MCPPolicyContext struct {
	ServerID             uint64
	ToolName             string
	CallerRole           string
	CallerScope          string
	ConfirmationApproved bool
	RecentCalls          int64
	Args                 map[string]any
	Now                  time.Time
}

type MCPPolicyEvaluation struct {
	PolicyID     uint64
	Decision     string
	Reason       string
	RedactFields []string
}

type SkillManifest struct {
	Name         string
	DisplayName  string
	Description  string
	Version      string
	Instruction  string
	RuntimeType  string
	InputSchema  string
	OutputSchema string
	Tools        []SkillToolManifest
}

type SkillToolManifest struct {
	Name          string
	RuntimeType   string
	InputSchema   string
	RuntimeConfig string
}

type SkillVersion struct {
	SkillID    uint64
	Version    int64
	Manifest   SkillManifest
	Content    string
	Active     bool
	ChangedBy  uint64
	ChangeNote string
	CreatedAt  time.Time
}

type EmbeddingProviderConfig struct {
	ID                     uint64
	Name                   string
	ProviderType           string
	Endpoint               string
	Enabled                bool
	HasEncryptedCredential bool
}

type EmbeddingModelConfig struct {
	ID          uint64
	ProviderID  uint64
	ModelName   string
	Dimension   int
	TokenLimit  int
	Enabled     bool
	Default     bool
	BatchSize   int
	TimeoutSecs int
}

type EmbeddingRuntimeState struct {
	Status         string
	ProviderID     uint64
	ModelID        uint64
	ModelName      string
	Dimension      int
	FallbackUsed   bool
	FallbackReason string
}

type CandidateRequirement struct {
	ID       string
	Label    string
	Priority string
	Category string
	Weight   float64
	Knockout bool
}

type CandidateRequirementResult struct {
	RequirementID string
	Status        string
	Score         float64
	Confidence    float64
	Risk          string
}

type CandidateMatchProfile struct {
	Requirements []CandidateRequirement
}

type CandidateMatchAggregation struct {
	OverallScore   float64
	Recommendation string
	Summary        string
	Dimensions     []ScoreDimension
	Risks          []string
	FallbackUsed   bool
}

type ScoreDimension struct {
	Name   string
	Label  string
	Weight float64
	Score  float64
}
