package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/netip"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"smart-recruit-ai-agent-service/internal/domain/model"
)

var (
	ErrMCPServerInvalid       = errors.New("mcp server config invalid")
	ErrMCPPolicyInvalid       = errors.New("mcp policy invalid")
	ErrSkillManifestInvalid   = errors.New("skill manifest invalid")
	ErrEmbeddingConfigInvalid = errors.New("embedding config invalid")
	ErrEmbeddingUnavailable   = errors.New("embedding unavailable")
	ErrCandidateMatchInvalid  = errors.New("candidate match invalid")
)

var skillNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,127}$`)

func ValidateMCPServerConfig(server model.MCPServerConfig) error {
	switch strings.ToLower(strings.TrimSpace(server.Transport)) {
	case model.MCPTransportStdio:
		if strings.TrimSpace(server.Command) == "" {
			return fmt.Errorf("%w: command is required", ErrMCPServerInvalid)
		}
		if len(server.AllowedCommands) > 0 && !commandAllowed(server.Command, server.AllowedCommands) {
			return fmt.Errorf("%w: command_not_allowed", ErrMCPServerInvalid)
		}
	case model.MCPTransportSSE, model.MCPTransportHTTP:
		if err := validateEndpointURL(server.URL, server.AllowPrivateNetwork); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: unsupported transport", ErrMCPServerInvalid)
	}
	if server.TimeoutSeconds < 0 {
		return fmt.Errorf("%w: timeout_seconds must be non-negative", ErrMCPServerInvalid)
	}
	return nil
}

func EvaluateMCPToolPolicy(policy *model.MCPToolPolicy, ctx model.MCPPolicyContext) (model.MCPPolicyEvaluation, error) {
	if policy == nil || !policy.Enabled {
		return model.MCPPolicyEvaluation{Decision: model.MCPPolicyDecisionAllow, Reason: "no_policy"}, nil
	}
	eval := model.MCPPolicyEvaluation{
		PolicyID:     policy.ID,
		Decision:     model.MCPPolicyDecisionAllow,
		Reason:       "policy_allow",
		RedactFields: append([]string(nil), policy.RedactFields...),
	}
	if strings.EqualFold(strings.TrimSpace(policy.Effect), model.MCPPolicyDecisionDeny) {
		eval.Decision = model.MCPPolicyDecisionDeny
		eval.Reason = "policy_effect_deny"
		return eval, nil
	}
	if !valueAllowed(ctx.CallerRole, policy.AllowedRoles) {
		eval.Decision = model.MCPPolicyDecisionDeny
		eval.Reason = "caller_role_not_allowed"
		return eval, nil
	}
	if !valueAllowed(ctx.CallerScope, policy.AllowedScopes) {
		eval.Decision = model.MCPPolicyDecisionDeny
		eval.Reason = "caller_scope_not_allowed"
		return eval, nil
	}
	if reason := validateMCPArgs(ctx.Args, policy); reason != "" {
		eval.Decision = model.MCPPolicyDecisionDeny
		eval.Reason = reason
		return eval, nil
	}
	if policy.RateLimitWindowSeconds > 0 && policy.RateLimitMaxCalls > 0 && ctx.RecentCalls >= int64(policy.RateLimitMaxCalls) {
		eval.Decision = model.MCPPolicyDecisionRateLimited
		eval.Reason = "rate_limit_exceeded"
		return eval, nil
	}
	if policy.RequireConfirmation && !ctx.ConfirmationApproved {
		eval.Decision = model.MCPPolicyDecisionConfirmationRequired
		eval.Reason = "confirmation_required"
		return eval, nil
	}
	return eval, nil
}

func ValidateSkillManifest(manifest model.SkillManifest) error {
	manifest.Name = strings.TrimSpace(manifest.Name)
	runtimeType := normalizeSkillRuntime(manifest.RuntimeType)
	if !skillNamePattern.MatchString(manifest.Name) {
		return fmt.Errorf("%w: name must match %s", ErrSkillManifestInvalid, skillNamePattern.String())
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return fmt.Errorf("%w: version is required", ErrSkillManifestInvalid)
	}
	if runtimeType == "" {
		runtimeType = model.SkillRuntimePrompt
	}
	if !supportedSkillRuntime(runtimeType) {
		return fmt.Errorf("%w: unsupported runtime", ErrSkillManifestInvalid)
	}
	if err := validateOptionalObjectSchema("input_schema", manifest.InputSchema); err != nil {
		return err
	}
	if err := validateOptionalObjectSchema("output_schema", manifest.OutputSchema); err != nil {
		return err
	}
	if len(manifest.Tools) == 0 && runtimeType != model.SkillRuntimePrompt {
		return fmt.Errorf("%w: tools are required for runtime", ErrSkillManifestInvalid)
	}
	seen := map[string]bool{}
	for i, tool := range manifest.Tools {
		toolRuntime := normalizeSkillRuntime(tool.RuntimeType)
		if toolRuntime == "" {
			toolRuntime = runtimeType
		}
		if !skillNamePattern.MatchString(strings.TrimSpace(tool.Name)) {
			return fmt.Errorf("%w: tools[%d].name invalid", ErrSkillManifestInvalid, i)
		}
		if seen[tool.Name] {
			return fmt.Errorf("%w: duplicate tool name %s", ErrSkillManifestInvalid, tool.Name)
		}
		seen[tool.Name] = true
		if !supportedSkillRuntime(toolRuntime) {
			return fmt.Errorf("%w: tools[%d].runtime unsupported", ErrSkillManifestInvalid, i)
		}
		if (toolRuntime == model.SkillRuntimeHTTP || toolRuntime == model.SkillRuntimeTool) && strings.TrimSpace(tool.InputSchema) == "" {
			return fmt.Errorf("%w: tools[%d].input_schema required", ErrSkillManifestInvalid, i)
		}
		if err := validateOptionalObjectSchema(fmt.Sprintf("tools[%d].input_schema", i), tool.InputSchema); err != nil {
			return err
		}
	}
	return nil
}

func NextSkillVersion(current int64, contentChanged bool) (int64, bool) {
	if !contentChanged {
		return current, false
	}
	if current <= 0 {
		return 1, true
	}
	return current + 1, true
}

func SelectAgentSkills(candidates []model.AgentSkill, req model.AgentSkillSelectionRequest) []model.SelectedAgentSkill {
	maxSkills := req.MaxSkills
	if maxSkills <= 0 || maxSkills > 3 {
		maxSkills = 3
	}
	manualSet := map[uint64]bool{}
	for _, id := range req.ManualIDs {
		if id > 0 {
			manualSet[id] = true
		}
	}
	selected := make([]model.SelectedAgentSkill, 0, maxSkills)
	seen := map[uint64]bool{}
	for _, skill := range candidates {
		if len(selected) >= maxSkills {
			break
		}
		if !manualSet[skill.ID] || !skill.Enabled || !skill.ManualInvocable || !agentTypeMatches(skill.AgentType, req.AgentType) || !capabilitiesAvailable(skill.RequiredCapabilities, req.AvailableCapabilities) {
			continue
		}
		selected = append(selected, model.SelectedAgentSkill{ID: skill.ID, Name: skill.Name, Manual: true, Score: float64(skill.Priority), Reason: "manual selection", RiskLevel: skill.RiskLevel})
		seen[skill.ID] = true
	}
	if len(manualSet) > 0 {
		return selected
	}
	if !agentSkillAutoEligible(req.Question) {
		return nil
	}
	auto := make([]model.SelectedAgentSkill, 0, len(candidates))
	for _, skill := range candidates {
		if seen[skill.ID] || !skill.Enabled || !agentTypeMatches(skill.AgentType, req.AgentType) || !capabilitiesAvailable(skill.RequiredCapabilities, req.AvailableCapabilities) {
			continue
		}
		score := float64(skill.Priority)
		if semantic, ok := req.SemanticScores[skill.ID]; ok {
			score += semantic * 100
		}
		if strings.Contains(strings.ToLower(req.Question), strings.ToLower(skill.Category)) && skill.Category != "" {
			score += 10
		}
		auto = append(auto, model.SelectedAgentSkill{ID: skill.ID, Name: skill.Name, Score: score, Reason: "auto selection", RiskLevel: skill.RiskLevel})
	}
	sort.SliceStable(auto, func(i, j int) bool {
		if auto[i].Score == auto[j].Score {
			return auto[i].ID < auto[j].ID
		}
		return auto[i].Score > auto[j].Score
	})
	for i := range auto {
		auto[i].PoolRank = i + 1
		if len(selected) < maxSkills {
			selected = append(selected, auto[i])
		}
	}
	return selected
}

func ValidateEmbeddingProvider(provider model.EmbeddingProviderConfig) error {
	if strings.TrimSpace(provider.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrEmbeddingConfigInvalid)
	}
	if strings.TrimSpace(provider.ProviderType) == "" {
		return fmt.Errorf("%w: provider_type is required", ErrEmbeddingConfigInvalid)
	}
	if strings.TrimSpace(provider.Endpoint) == "" {
		return fmt.Errorf("%w: endpoint is required", ErrEmbeddingConfigInvalid)
	}
	if !provider.HasEncryptedCredential {
		return fmt.Errorf("%w: encrypted credential is required", ErrEmbeddingConfigInvalid)
	}
	return nil
}

func ValidateEmbeddingModel(provider model.EmbeddingProviderConfig, embeddingModel model.EmbeddingModelConfig) error {
	if provider.ID == 0 || embeddingModel.ProviderID != provider.ID {
		return fmt.Errorf("%w: provider mismatch", ErrEmbeddingConfigInvalid)
	}
	if !provider.Enabled {
		return fmt.Errorf("%w: provider disabled", ErrEmbeddingUnavailable)
	}
	if strings.TrimSpace(embeddingModel.ModelName) == "" {
		return fmt.Errorf("%w: model_name is required", ErrEmbeddingConfigInvalid)
	}
	if embeddingModel.Dimension <= 0 {
		return fmt.Errorf("%w: dimension must be positive", ErrEmbeddingConfigInvalid)
	}
	if !embeddingModel.Enabled {
		return fmt.Errorf("%w: model disabled", ErrEmbeddingUnavailable)
	}
	return nil
}

func ResolveEmbeddingRuntime(provider *model.EmbeddingProviderConfig, embeddingModel *model.EmbeddingModelConfig, fallbackReason string) model.EmbeddingRuntimeState {
	if provider == nil || embeddingModel == nil {
		return model.EmbeddingRuntimeState{Status: model.EmbeddingStatusUnavailable, FallbackUsed: true, FallbackReason: defaultString(fallbackReason, "embedding_config_missing")}
	}
	if err := ValidateEmbeddingModel(*provider, *embeddingModel); err != nil {
		return model.EmbeddingRuntimeState{Status: model.EmbeddingStatusUnavailable, ProviderID: provider.ID, ModelID: embeddingModel.ID, ModelName: embeddingModel.ModelName, Dimension: embeddingModel.Dimension, FallbackUsed: true, FallbackReason: err.Error()}
	}
	return model.EmbeddingRuntimeState{Status: model.EmbeddingStatusAvailable, ProviderID: provider.ID, ModelID: embeddingModel.ID, ModelName: embeddingModel.ModelName, Dimension: embeddingModel.Dimension}
}

func AggregateCandidateMatch(profile model.CandidateMatchProfile, results []model.CandidateRequirementResult, fallbackUsed bool) (model.CandidateMatchAggregation, error) {
	if len(profile.Requirements) == 0 {
		return model.CandidateMatchAggregation{OverallScore: 0, Recommendation: model.RecommendationNot, Summary: "暂无评估数据", Risks: []string{"no_data"}, FallbackUsed: fallbackUsed}, nil
	}
	resultByID := map[string]model.CandidateRequirementResult{}
	for _, result := range results {
		resultByID[result.RequirementID] = result
	}
	mustHave := scoreRequirements(profile.Requirements, resultByID, func(r model.CandidateRequirement) bool { return r.Priority == model.RequirementMustHave }, 0)
	coreSkill := scoreRequirements(profile.Requirements, resultByID, func(r model.CandidateRequirement) bool { return r.Category == model.RequirementCoreSkill }, 50)
	experience := scoreRequirements(profile.Requirements, resultByID, func(r model.CandidateRequirement) bool { return r.Category == model.RequirementExperience }, 30)
	growth := scoreRequirements(profile.Requirements, resultByID, func(r model.CandidateRequirement) bool {
		return r.Priority == model.RequirementNiceToHave || r.Priority == model.RequirementSoftSkill
	}, 50)
	knockoutMissing := 0
	missing := 0
	for _, req := range profile.Requirements {
		result, ok := resultByID[req.ID]
		if !ok || result.Status == model.MatchStatusMissing || result.Status == model.MatchStatusConflict {
			if req.Knockout {
				knockoutMissing++
			}
			missing++
		}
	}
	riskPenalty := math.Min(float64(missing*5+knockoutMissing*10), 20)
	overall := roundScore(mustHave*0.40 + coreSkill*0.25 + experience*0.20 + growth*0.10 - riskPenalty)
	if knockoutMissing > 0 {
		overall = math.Min(overall, 40)
	}
	recommendation := recommendationForScore(overall, profile.Requirements, resultByID, knockoutMissing > 0)
	risks := []string{}
	if knockoutMissing > 0 {
		risks = append(risks, "knockout_missing")
	}
	if missing > 0 {
		risks = append(risks, "missing_requirements")
	}
	return model.CandidateMatchAggregation{
		OverallScore:   overall,
		Recommendation: recommendation,
		Summary:        fmt.Sprintf("综合评分 %.1f 分。必备要求和核心技能已按本地策略聚合。", overall),
		FallbackUsed:   fallbackUsed,
		Risks:          risks,
		Dimensions: []model.ScoreDimension{
			{Name: "must_have", Label: "必备要求满足度", Weight: 0.40, Score: mustHave},
			{Name: "core_skills", Label: "核心技能匹配", Weight: 0.25, Score: coreSkill},
			{Name: "experience", Label: "项目/经验证据", Weight: 0.20, Score: experience},
			{Name: "growth", Label: "成长潜力与加分项", Weight: 0.10, Score: growth},
		},
	}, nil
}

func validateEndpointURL(rawURL string, allowPrivate bool) error {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%w: url must be absolute", ErrMCPServerInvalid)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: url scheme must be http or https", ErrMCPServerInvalid)
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return fmt.Errorf("%w: url host is required", ErrMCPServerInvalid)
	}
	if allowPrivate {
		return nil
	}
	if isPrivateHost(host) {
		return fmt.Errorf("%w: private_network_denied", ErrMCPServerInvalid)
	}
	return nil
}

func commandAllowed(command string, allowed []string) bool {
	base := filepath.Base(strings.Fields(command)[0])
	for _, item := range allowed {
		if base == item || command == item {
			return true
		}
	}
	return false
}

func isPrivateHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if addr, ok := netip.AddrFromSlice(ip); ok && (addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast()) {
			return true
		}
	}
	return false
}

func validateMCPArgs(args map[string]any, policy *model.MCPToolPolicy) string {
	for _, name := range policy.RequiredArgs {
		if _, ok := args[name]; !ok {
			return "missing_required_arg:" + name
		}
	}
	for _, name := range policy.DeniedArgs {
		if _, ok := args[name]; ok {
			return "denied_arg_present:" + name
		}
	}
	for name, rule := range policy.ArgRules {
		value, exists := args[name]
		if rule.Required && !exists {
			return "missing_required_arg:" + name
		}
		if !exists {
			continue
		}
		if rule.Deny {
			return "denied_arg_present:" + name
		}
		if len(rule.Enum) > 0 && !stringInSet(fmt.Sprint(value), rule.Enum) {
			return "arg_enum_mismatch:" + name
		}
		if rule.Regex != "" {
			matched, err := regexp.MatchString(rule.Regex, fmt.Sprint(value))
			if err != nil || !matched {
				return "arg_regex_mismatch:" + name
			}
		}
		if rule.Min != nil || rule.Max != nil {
			number, ok := numericValue(value)
			if !ok {
				return "arg_numeric_mismatch:" + name
			}
			if rule.Min != nil && number < *rule.Min {
				return "arg_min_mismatch:" + name
			}
			if rule.Max != nil && number > *rule.Max {
				return "arg_max_mismatch:" + name
			}
		}
	}
	return ""
}

func valueAllowed(value string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, item := range allowed {
		if item == "*" || strings.EqualFold(item, value) {
			return true
		}
	}
	return false
}

func stringInSet(value string, allowed []string) bool {
	for _, item := range allowed {
		if item == value {
			return true
		}
	}
	return false
}

func numericValue(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func normalizeSkillRuntime(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func supportedSkillRuntime(v string) bool {
	switch v {
	case model.SkillRuntimePrompt, model.SkillRuntimeTool, model.SkillRuntimeWorkflow, model.SkillRuntimeHTTP:
		return true
	default:
		return false
	}
}

func validateOptionalObjectSchema(field, raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var schema map[string]any
	if err := json.Unmarshal([]byte(raw), &schema); err != nil {
		return fmt.Errorf("%w: %s must be JSON object schema", ErrSkillManifestInvalid, field)
	}
	if len(schema) == 0 {
		return fmt.Errorf("%w: %s must not be empty", ErrSkillManifestInvalid, field)
	}
	if typ, ok := schema["type"].(string); !ok || typ != "object" {
		return fmt.Errorf("%w: %s.type must be object", ErrSkillManifestInvalid, field)
	}
	return nil
}

func agentTypeMatches(skillAgentType, requested string) bool {
	return strings.TrimSpace(skillAgentType) == "" || strings.EqualFold(skillAgentType, requested)
}

func capabilitiesAvailable(required []string, available map[string]bool) bool {
	for _, capability := range required {
		if !available[capability] {
			return false
		}
	}
	return true
}

func agentSkillAutoEligible(question string) bool {
	normalized := strings.ToLower(strings.TrimSpace(question))
	if meaningfulRuneCount(normalized) <= 4 {
		return false
	}
	action := hasAnyTerm(normalized, []string{"筛", "匹配", "评估", "分析", "推荐", "搜索", "安排", "生成", "比较", "screen", "match", "evaluate", "analyze", "recommend", "search"})
	object := hasAnyTerm(normalized, []string{"简历", "候选", "面试", "职位", "岗位", "招聘", "resume", "candidate", "interview", "job", "position", "recruiting"})
	return action && object
}

func hasAnyTerm(text string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func meaningfulRuneCount(text string) int {
	count := 0
	for _, r := range text {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		count++
	}
	return count
}

func scoreRequirements(requirements []model.CandidateRequirement, results map[string]model.CandidateRequirementResult, predicate func(model.CandidateRequirement) bool, defaultScore float64) float64 {
	totalWeight := 0.0
	weightedScore := 0.0
	matched := false
	for _, req := range requirements {
		if !predicate(req) {
			continue
		}
		result, ok := results[req.ID]
		if !ok {
			continue
		}
		weight := req.Weight
		if weight <= 0 {
			weight = 1
		}
		weightedScore += result.Score * weight
		totalWeight += weight
		matched = true
	}
	if !matched || totalWeight == 0 {
		return defaultScore
	}
	return roundScore(weightedScore / totalWeight)
}

func recommendationForScore(score float64, requirements []model.CandidateRequirement, results map[string]model.CandidateRequirementResult, knockout bool) string {
	if knockout {
		return model.RecommendationStrongNot
	}
	passed, total := 0, 0
	for _, req := range requirements {
		if req.Priority != model.RequirementMustHave {
			continue
		}
		total++
		if result, ok := results[req.ID]; ok && (result.Status == model.MatchStatusStrongMatch || result.Status == model.MatchStatusMatch) {
			passed++
		}
	}
	ratio := 0.0
	if total > 0 {
		ratio = float64(passed) / float64(total)
	}
	if score >= 85 && ratio >= 0.8 {
		return model.RecommendationStrong
	}
	if score >= 70 && ratio >= 0.6 {
		return model.RecommendationRecommend
	}
	if score >= 50 {
		return model.RecommendationReview
	}
	if score >= 30 {
		return model.RecommendationNot
	}
	return model.RecommendationStrongNot
}

func roundScore(score float64) float64 {
	return math.Round(math.Max(0, math.Min(100, score))*10) / 10
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
