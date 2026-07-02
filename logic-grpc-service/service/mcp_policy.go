package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

const (
	mcpPolicyDecisionAllow        = "allow"
	mcpPolicyDecisionDeny         = "deny"
	mcpPolicyDecisionConfirmation = "confirmation_required"
	mcpPolicyDecisionRateLimited  = "rate_limited"
)

type mcpPolicyEvaluation struct {
	Policy       *model.MCPToolPolicy
	Decision     string
	Reason       string
	RedactFields []string
	SnapshotJSON string
}

type mcpArgRule struct {
	Required bool     `json:"required"`
	Deny     bool     `json:"deny"`
	Enum     []string `json:"enum"`
	Regex    string   `json:"regex"`
	Min      *float64 `json:"min"`
	Max      *float64 `json:"max"`
}

func (s *MCPService) evaluateToolPolicy(ctx context.Context, serverID int64, toolName string, args map[string]any, callerRole, callerScope string, confirmationApproved bool) (mcpPolicyEvaluation, error) {
	policy, err := s.mcpRepo.GetEnabledToolPolicy(ctx, serverID, toolName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return mcpPolicyEvaluation{Decision: mcpPolicyDecisionAllow, Reason: "no_policy"}, nil
		}
		return mcpPolicyEvaluation{}, fmt.Errorf("get MCP tool policy: %w", err)
	}

	eval := mcpPolicyEvaluation{
		Policy:       policy,
		Decision:     mcpPolicyDecisionAllow,
		Reason:       "policy_allow",
		RedactFields: parseJSONStringArray(policy.RedactFieldsJSON),
		SnapshotJSON: mcpPolicySnapshotJSON(policy),
	}

	if strings.EqualFold(strings.TrimSpace(policy.Effect), mcpPolicyDecisionDeny) {
		eval.Decision = mcpPolicyDecisionDeny
		eval.Reason = "policy_effect_deny"
		return eval, nil
	}

	if !valueAllowed(callerRole, parseJSONStringArray(policy.AllowedRolesJSON)) {
		eval.Decision = mcpPolicyDecisionDeny
		eval.Reason = "caller_role_not_allowed"
		return eval, nil
	}
	if !valueAllowed(callerScope, parseJSONStringArray(policy.AllowedScopesJSON)) {
		eval.Decision = mcpPolicyDecisionDeny
		eval.Reason = "caller_scope_not_allowed"
		return eval, nil
	}

	if reason := validateMCPPolicyArgs(args, policy); reason != "" {
		eval.Decision = mcpPolicyDecisionDeny
		eval.Reason = reason
		return eval, nil
	}

	if policy.RateLimitWindowSeconds > 0 && policy.RateLimitMaxCalls > 0 {
		since := time.Now().Add(-time.Duration(policy.RateLimitWindowSeconds) * time.Second)
		count, err := s.mcpRepo.CountToolLogsSince(ctx, serverID, toolName, since)
		if err != nil {
			return mcpPolicyEvaluation{}, fmt.Errorf("count MCP tool logs for rate limit: %w", err)
		}
		if count >= int64(policy.RateLimitMaxCalls) {
			eval.Decision = mcpPolicyDecisionRateLimited
			eval.Reason = "rate_limit_exceeded"
			return eval, nil
		}
	}

	if policy.RequireConfirmation == 1 && !confirmationApproved {
		eval.Decision = mcpPolicyDecisionConfirmation
		eval.Reason = "confirmation_required"
		return eval, nil
	}

	return eval, nil
}

func validateMCPPolicyArgs(args map[string]any, policy *model.MCPToolPolicy) string {
	for _, name := range parseJSONStringArray(policy.RequiredArgsJSON) {
		if _, ok := args[name]; !ok {
			return "missing_required_arg:" + name
		}
	}
	for _, name := range parseJSONStringArray(policy.DeniedArgsJSON) {
		if _, ok := args[name]; ok {
			return "denied_arg_present:" + name
		}
	}

	rules := parseMCPArgRules(policy.ArgRulesJSON)
	for name, rule := range rules {
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

func parseMCPArgs(raw string) (map[string]any, any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil, nil
	}
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, nil, err
	}
	obj, ok := parsed.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("args_json must be a JSON object")
	}
	return obj, parsed, nil
}

func parseJSONStringArray(raw *string) []string {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	var values []string
	if err := json.Unmarshal([]byte(*raw), &values); err != nil {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func parseMCPArgRules(raw *string) map[string]mcpArgRule {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	var rules map[string]mcpArgRule
	if err := json.Unmarshal([]byte(*raw), &rules); err != nil {
		return nil
	}
	return rules
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

func mcpPolicySnapshotJSON(policy *model.MCPToolPolicy) string {
	if policy == nil {
		return ""
	}
	return safeJSON(map[string]any{
		"policy_id":                 policy.ID,
		"server_id":                 policy.ServerID,
		"tool_name":                 policy.ToolName,
		"effect":                    policy.Effect,
		"risk_level":                policy.RiskLevel,
		"require_confirmation":      policy.RequireConfirmation == 1,
		"allowed_roles_json":        valueOrEmpty(policy.AllowedRolesJSON),
		"allowed_scopes_json":       valueOrEmpty(policy.AllowedScopesJSON),
		"required_args_json":        valueOrEmpty(policy.RequiredArgsJSON),
		"denied_args_json":          valueOrEmpty(policy.DeniedArgsJSON),
		"arg_rules_json":            valueOrEmpty(policy.ArgRulesJSON),
		"redact_fields_json":        valueOrEmpty(policy.RedactFieldsJSON),
		"rate_limit_window_seconds": policy.RateLimitWindowSeconds,
		"rate_limit_max_calls":      policy.RateLimitMaxCalls,
	})
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
