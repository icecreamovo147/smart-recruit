package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
)

func (s *MCPService) ListMCPToolPolicies(ctx context.Context, req *pb.ListMCPToolPoliciesRequest) (*pb.ListMCPToolPoliciesResponse, error) {
	log := logger.GetRequestLogger(ctx)
	log.Info("[logic][mcp_policy] ListMCPToolPolicies started", zap.Int64("server_id", req.GetServerId()))
	page := req.GetPage()
	if page <= 0 {
		page = 1
	}
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}
	policies, total, err := s.mcpRepo.ListToolPolicies(ctx, req.GetServerId(), page, pageSize)
	if err != nil {
		log.Error("[logic][mcp_policy] list MCP tool policies failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "list MCP tool policies failed")
	}
	list := make([]*pb.MCPToolPolicyInfo, 0, len(policies))
	for i := range policies {
		list = append(list, mcpToolPolicyToInfo(&policies[i]))
	}
	log.Info("[logic][mcp_policy] ListMCPToolPolicies succeeded", zap.Int("total", int(total)), zap.Int("returned", len(list)))
	return &pb.ListMCPToolPoliciesResponse{Code: 0, Msg: "ok", Total: total, List: list}, nil
}

func (s *MCPService) CreateMCPToolPolicy(ctx context.Context, req *pb.CreateMCPToolPolicyRequest) (*pb.MCPToolPolicyResponse, error) {
	log := logger.GetRequestLogger(ctx)
	log.Info("[logic][mcp_policy] CreateMCPToolPolicy started",
		zap.Int64("server_id", req.GetServerId()),
		zap.String("tool_name", req.GetToolName()),
		zap.String("effect", req.GetEffect()))
	policy, err := buildMCPToolPolicyFromCreate(req)
	if err != nil {
		log.Warn("[logic][mcp_policy] CreateMCPToolPolicy validation failed", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if _, err := s.mcpRepo.GetServerByID(ctx, policy.ServerID); err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Warn("[logic][mcp_policy] CreateMCPToolPolicy server not found")
			return nil, status.Error(codes.NotFound, "MCP server not found")
		}
		log.Error("[logic][mcp_policy] CreateMCPToolPolicy get server failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get MCP server failed")
	}
	if err := s.mcpRepo.CreateToolPolicy(ctx, policy); err != nil {
		log.Error("[logic][mcp_policy] create MCP tool policy failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "create MCP tool policy failed")
	}
	log.Info("[logic][mcp_policy] CreateMCPToolPolicy succeeded",
		zap.Int64("policy_id", policy.ID),
		zap.String("tool_name", policy.ToolName))
	return &pb.MCPToolPolicyResponse{Code: 0, Msg: "ok", Policy: mcpToolPolicyToInfo(policy)}, nil
}

func (s *MCPService) UpdateMCPToolPolicy(ctx context.Context, req *pb.UpdateMCPToolPolicyRequest) (*pb.MCPToolPolicyResponse, error) {
	log := logger.GetRequestLogger(ctx)
	log.Info("[logic][mcp_policy] UpdateMCPToolPolicy started", zap.Int64("policy_id", req.GetId()))
	if req.GetId() <= 0 {
		log.Warn("[logic][mcp_policy] UpdateMCPToolPolicy invalid id")
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	existing, err := s.mcpRepo.GetToolPolicyByID(ctx, req.GetId())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Warn("[logic][mcp_policy] UpdateMCPToolPolicy policy not found")
			return nil, status.Error(codes.NotFound, "MCP tool policy not found")
		}
		log.Error("[logic][mcp_policy] get MCP tool policy failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get MCP tool policy failed")
	}
	policy, err := buildMCPToolPolicyFromUpdate(existing, req)
	if err != nil {
		log.Warn("[logic][mcp_policy] UpdateMCPToolPolicy validation failed", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if _, err := s.mcpRepo.GetServerByID(ctx, policy.ServerID); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "MCP server not found")
		}
		log.Error("[logic][mcp_policy] get MCP server failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get MCP server failed")
	}
	if err := s.mcpRepo.UpdateToolPolicy(ctx, policy); err != nil {
		log.Error("[logic][mcp_policy] update MCP tool policy failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "update MCP tool policy failed")
	}
	updated, err := s.mcpRepo.GetToolPolicyByID(ctx, policy.ID)
	if err != nil {
		log.Error("[logic][mcp_policy] get updated MCP tool policy failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get updated MCP tool policy failed")
	}
	log.Info("[logic][mcp_policy] UpdateMCPToolPolicy succeeded",
		zap.Int64("policy_id", updated.ID),
		zap.String("tool_name", updated.ToolName))
	return &pb.MCPToolPolicyResponse{Code: 0, Msg: "ok", Policy: mcpToolPolicyToInfo(updated)}, nil
}

func (s *MCPService) DeleteMCPToolPolicy(ctx context.Context, req *pb.DeleteMCPToolPolicyRequest) (*pb.CommonResponse, error) {
	log := logger.GetRequestLogger(ctx)
	log.Info("[logic][mcp_policy] DeleteMCPToolPolicy started", zap.Int64("policy_id", req.GetId()))
	if req.GetId() <= 0 {
		log.Warn("[logic][mcp_policy] DeleteMCPToolPolicy invalid id")
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if err := s.mcpRepo.DeleteToolPolicy(ctx, req.GetId()); err != nil {
		log.Error("[logic][mcp_policy] delete MCP tool policy failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "delete MCP tool policy failed")
	}
	log.Info("[logic][mcp_policy] DeleteMCPToolPolicy succeeded", zap.Int64("policy_id", req.GetId()))
	return &pb.CommonResponse{Code: 0, Msg: "ok"}, nil
}

func buildMCPToolPolicyFromCreate(req *pb.CreateMCPToolPolicyRequest) (*model.MCPToolPolicy, error) {
	policy := &model.MCPToolPolicy{
		ServerID:               req.GetServerId(),
		ToolName:               strings.TrimSpace(req.GetToolName()),
		Effect:                 strings.ToLower(strings.TrimSpace(req.GetEffect())),
		RiskLevel:              strings.ToLower(strings.TrimSpace(req.GetRiskLevel())),
		RateLimitWindowSeconds: req.GetRateLimitWindowSeconds(),
		RateLimitMaxCalls:      req.GetRateLimitMaxCalls(),
		IsEnabled:              1,
		CreatedByHRID:          positiveInt64Ptr(req.GetOperatorHrId()),
		UpdatedByHRID:          positiveInt64Ptr(req.GetOperatorHrId()),
	}
	if req.GetIsEnabledSet() {
		policy.IsEnabled = mcpBoolToInt32(req.GetIsEnabled())
	}
	policy.RequireConfirmation = mcpBoolToInt32(req.GetRequireConfirmation())
	assignPolicyJSON(policy, req.GetAllowedRolesJson(), req.GetAllowedScopesJson(), req.GetRequiredArgsJson(), req.GetDeniedArgsJson(), req.GetArgRulesJson(), req.GetRedactFieldsJson())
	return policy, validateMCPToolPolicy(policy)
}

func buildMCPToolPolicyFromUpdate(existing *model.MCPToolPolicy, req *pb.UpdateMCPToolPolicyRequest) (*model.MCPToolPolicy, error) {
	policy := *existing
	if req.GetServerId() > 0 {
		policy.ServerID = req.GetServerId()
	}
	if strings.TrimSpace(req.GetToolName()) != "" {
		policy.ToolName = strings.TrimSpace(req.GetToolName())
	}
	if strings.TrimSpace(req.GetEffect()) != "" {
		policy.Effect = strings.ToLower(strings.TrimSpace(req.GetEffect()))
	}
	if strings.TrimSpace(req.GetRiskLevel()) != "" {
		policy.RiskLevel = strings.ToLower(strings.TrimSpace(req.GetRiskLevel()))
	}
	if req.GetRequireConfirmationSet() {
		policy.RequireConfirmation = mcpBoolToInt32(req.GetRequireConfirmation())
	}
	if req.GetAllowedRolesJson() != "" {
		policy.AllowedRolesJSON = strPtrOrNil(req.GetAllowedRolesJson())
	}
	if req.GetAllowedScopesJson() != "" {
		policy.AllowedScopesJSON = strPtrOrNil(req.GetAllowedScopesJson())
	}
	if req.GetRequiredArgsJson() != "" {
		policy.RequiredArgsJSON = strPtrOrNil(req.GetRequiredArgsJson())
	}
	if req.GetDeniedArgsJson() != "" {
		policy.DeniedArgsJSON = strPtrOrNil(req.GetDeniedArgsJson())
	}
	if req.GetArgRulesJson() != "" {
		policy.ArgRulesJSON = strPtrOrNil(req.GetArgRulesJson())
	}
	if req.GetRedactFieldsJson() != "" {
		policy.RedactFieldsJSON = strPtrOrNil(req.GetRedactFieldsJson())
	}
	if req.GetRateLimitWindowSecondsSet() {
		policy.RateLimitWindowSeconds = req.GetRateLimitWindowSeconds()
	}
	if req.GetRateLimitMaxCallsSet() {
		policy.RateLimitMaxCalls = req.GetRateLimitMaxCalls()
	}
	if req.GetIsEnabledSet() {
		policy.IsEnabled = mcpBoolToInt32(req.GetIsEnabled())
	}
	policy.UpdatedByHRID = positiveInt64Ptr(req.GetOperatorHrId())
	return &policy, validateMCPToolPolicy(&policy)
}

func assignPolicyJSON(policy *model.MCPToolPolicy, allowedRoles, allowedScopes, requiredArgs, deniedArgs, argRules, redactFields string) {
	policy.AllowedRolesJSON = strPtrOrNil(allowedRoles)
	policy.AllowedScopesJSON = strPtrOrNil(allowedScopes)
	policy.RequiredArgsJSON = strPtrOrNil(requiredArgs)
	policy.DeniedArgsJSON = strPtrOrNil(deniedArgs)
	policy.ArgRulesJSON = strPtrOrNil(argRules)
	policy.RedactFieldsJSON = strPtrOrNil(redactFields)
}

func validateMCPToolPolicy(policy *model.MCPToolPolicy) error {
	if policy.ServerID <= 0 {
		return errInvalid("server_id is required")
	}
	if strings.TrimSpace(policy.ToolName) == "" {
		return errInvalid("tool_name is required")
	}
	if policy.Effect == "" {
		policy.Effect = "allow"
	}
	if policy.Effect != "allow" && policy.Effect != "deny" {
		return errInvalid("effect must be allow or deny")
	}
	if policy.RiskLevel == "" {
		policy.RiskLevel = "medium"
	}
	if policy.RateLimitWindowSeconds < 0 || policy.RateLimitMaxCalls < 0 {
		return errInvalid("rate limit values must be non-negative")
	}
	for label, raw := range map[string]*string{
		"allowed_roles_json":  policy.AllowedRolesJSON,
		"allowed_scopes_json": policy.AllowedScopesJSON,
		"required_args_json":  policy.RequiredArgsJSON,
		"denied_args_json":    policy.DeniedArgsJSON,
		"redact_fields_json":  policy.RedactFieldsJSON,
	} {
		if err := validateJSONArray(label, raw); err != nil {
			return err
		}
	}
	return validateJSONObject("arg_rules_json", policy.ArgRulesJSON)
}

func validateJSONArray(label string, raw *string) error {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	var values []string
	if err := json.Unmarshal([]byte(*raw), &values); err != nil {
		return errInvalid(label + " must be a JSON string array")
	}
	return nil
}

func validateJSONObject(label string, raw *string) error {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	var values map[string]any
	if err := json.Unmarshal([]byte(*raw), &values); err != nil {
		return errInvalid(label + " must be a JSON object")
	}
	return nil
}

type invalidPolicyError string

func (e invalidPolicyError) Error() string { return string(e) }

func errInvalid(message string) error { return invalidPolicyError(message) }

func mcpToolPolicyToInfo(policy *model.MCPToolPolicy) *pb.MCPToolPolicyInfo {
	if policy == nil {
		return nil
	}
	return &pb.MCPToolPolicyInfo{
		Id:                     policy.ID,
		ServerId:               policy.ServerID,
		ToolName:               policy.ToolName,
		Effect:                 policy.Effect,
		RiskLevel:              policy.RiskLevel,
		RequireConfirmation:    policy.RequireConfirmation == 1,
		AllowedRolesJson:       valueOrEmpty(policy.AllowedRolesJSON),
		AllowedScopesJson:      valueOrEmpty(policy.AllowedScopesJSON),
		RequiredArgsJson:       valueOrEmpty(policy.RequiredArgsJSON),
		DeniedArgsJson:         valueOrEmpty(policy.DeniedArgsJSON),
		ArgRulesJson:           valueOrEmpty(policy.ArgRulesJSON),
		RedactFieldsJson:       valueOrEmpty(policy.RedactFieldsJSON),
		RateLimitWindowSeconds: policy.RateLimitWindowSeconds,
		RateLimitMaxCalls:      policy.RateLimitMaxCalls,
		IsEnabled:              policy.IsEnabled == 1,
		CreatedByHrId:          int64Value(policy.CreatedByHRID),
		UpdatedByHrId:          int64Value(policy.UpdatedByHRID),
		CreatedAt:              formatMCPPolicyTime(policy.CreatedAt),
		UpdatedAt:              formatMCPPolicyTime(policy.UpdatedAt),
	}
}

func mcpBoolToInt32(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

func positiveInt64Ptr(value int64) *int64 {
	if value <= 0 {
		return nil
	}
	return &value
}

func int64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func formatMCPPolicyTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}
