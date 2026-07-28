package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"smart-recruit-ai-agent-service/internal/application/contextbudget"
	"smart-recruit-ai-agent-service/internal/domain/agentskill"
	"smart-recruit-ai-agent-service/internal/domain/model"
	"smart-recruit-ai-agent-service/internal/domain/policy"
	mcpinfra "smart-recruit-ai-agent-service/internal/infrastructure/mcp"
	"smart-recruit-proto/recruitment/pb"
)

const (
	governanceOK          int32 = 0
	governanceBadRequest  int32 = 400
	governanceNotFound    int32 = 404
	governanceUnavailable int32 = 500
	governanceUnsupported int32 = 501
)

var errAgentSkillNameMismatch = errors.New("agent skill package name does not match registry name")

type mcpToolPolicyRecord struct {
	ID                     int64          `gorm:"primaryKey"`
	ServerID               int64          `gorm:"column:server_id"`
	ToolName               string         `gorm:"column:tool_name"`
	Effect                 string         `gorm:"column:effect"`
	RiskLevel              string         `gorm:"column:risk_level"`
	RequireConfirmation    bool           `gorm:"column:require_confirmation"`
	AllowedRolesJSON       sql.NullString `gorm:"column:allowed_roles_json"`
	AllowedScopesJSON      sql.NullString `gorm:"column:allowed_scopes_json"`
	RequiredArgsJSON       sql.NullString `gorm:"column:required_args_json"`
	DeniedArgsJSON         sql.NullString `gorm:"column:denied_args_json"`
	ArgRulesJSON           sql.NullString `gorm:"column:arg_rules_json"`
	RedactFieldsJSON       sql.NullString `gorm:"column:redact_fields_json"`
	RateLimitWindowSeconds int            `gorm:"column:rate_limit_window_seconds"`
	RateLimitMaxCalls      int            `gorm:"column:rate_limit_max_calls"`
	IsEnabled              bool           `gorm:"column:is_enabled"`
	CreatedByHRID          sql.NullInt64  `gorm:"column:created_by_hr_id"`
	UpdatedByHRID          sql.NullInt64  `gorm:"column:updated_by_hr_id"`
	CreatedAt              time.Time      `gorm:"column:created_at"`
	UpdatedAt              time.Time      `gorm:"column:updated_at"`
}

func (mcpToolPolicyRecord) TableName() string { return "mcp_tool_policies" }

type mcpToolLogRecord struct {
	ID             int64          `gorm:"primaryKey"`
	TenantID       *int64         `gorm:"column:tenant_id"`
	ServerID       int64          `gorm:"column:server_id"`
	ToolName       string         `gorm:"column:tool_name"`
	ArgsJSON       sql.NullString `gorm:"column:args_json"`
	ResultContent  sql.NullString `gorm:"column:result_content"`
	DurationMs     int            `gorm:"column:duration_ms"`
	ErrorMsg       sql.NullString `gorm:"column:error_msg"`
	CalledByHRID   sql.NullInt64  `gorm:"column:called_by_hr_id"`
	SessionID      sql.NullInt64  `gorm:"column:session_id"`
	PolicyID       sql.NullInt64  `gorm:"column:policy_id"`
	PolicyDecision string         `gorm:"column:policy_decision"`
	PolicyReason   sql.NullString `gorm:"column:policy_reason"`
	CreatedAt      time.Time      `gorm:"column:created_at"`
}

func (mcpToolLogRecord) TableName() string { return "mcp_tool_logs" }

type agentSkillVersionRecord struct {
	ID                  int64          `gorm:"primaryKey"`
	SkillID             int64          `gorm:"column:skill_id"`
	Version             string         `gorm:"column:version"`
	ManifestJSON        string         `gorm:"column:manifest_json"`
	CoreMarkdown        string         `gorm:"column:core_markdown"`
	CompiledMarkdown    string         `gorm:"column:compiled_markdown"`
	AuthoringJSON       sql.NullString `gorm:"column:authoring_json"`
	CompiledHash        string         `gorm:"column:compiled_hash"`
	CoreEstimatedTokens int            `gorm:"column:core_estimated_tokens"`
	ChangeNote          sql.NullString `gorm:"column:change_note"`
	CreatedBy           sql.NullInt64  `gorm:"column:created_by"`
	CreatedAt           time.Time      `gorm:"column:created_at"`
}

func (agentSkillVersionRecord) TableName() string { return "agent_skill_versions" }

type agentSkillSectionRecord struct {
	ID                 int64          `gorm:"primaryKey"`
	SkillVersionID     int64          `gorm:"column:skill_version_id"`
	SectionKey         string         `gorm:"column:section_key"`
	Title              string         `gorm:"column:title"`
	Description        sql.NullString `gorm:"column:description"`
	ContentMarkdown    string         `gorm:"column:content_markdown"`
	TriggerTermsJSON   sql.NullString `gorm:"column:trigger_terms_json"`
	SemanticTagsJSON   sql.NullString `gorm:"column:semantic_tags_json"`
	PlannerIntentsJSON sql.NullString `gorm:"column:planner_intents_json"`
	Priority           int            `gorm:"column:priority"`
	Ordinal            int            `gorm:"column:ordinal"`
	EstimatedTokens    int            `gorm:"column:estimated_tokens"`
	ContentHash        string         `gorm:"column:content_hash"`
	CreatedAt          time.Time      `gorm:"column:created_at"`
}

func (agentSkillSectionRecord) TableName() string { return "agent_skill_version_sections" }

func (s *NativeStore) CreateMCPServer(ctx context.Context, req *pb.CreateMCPServerRequest) (*pb.MCPServerResponse, error) {
	transport := strings.ToLower(strings.TrimSpace(req.GetTransport()))
	if err := validateMCPConfig(transport, req.GetCommandOrUrl(), int(req.GetTimeoutSeconds())); err != nil {
		return &pb.MCPServerResponse{Code: governanceBadRequest, Msg: "common.invalid_request"}, nil
	}
	row := mcpServerRecord{Name: strings.TrimSpace(req.GetName()), Description: nullStringFrom(req.GetDescription(), true), Transport: transport, CommandOrURL: strings.TrimSpace(req.GetCommandOrUrl()), Args: nullStringFrom(req.GetArgs(), true), EnvVars: nullStringFrom(req.GetEnvVars(), true), TimeoutSeconds: defaultInt32(req.GetTimeoutSeconds(), 30), IsEnabled: false, Status: "disconnected"}
	if row.Name == "" {
		return &pb.MCPServerResponse{Code: governanceBadRequest, Msg: "common.invalid_request"}, nil
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return s.getMCPServerResponse(ctx, row.ID)
}

func (s *NativeStore) UpdateMCPServer(ctx context.Context, req *pb.UpdateMCPServerRequest) (*pb.MCPServerResponse, error) {
	if err := s.assertMCPServerNotReleased(ctx, req.GetId()); err != nil {
		return &pb.MCPServerResponse{Code: governanceBadRequest, Msg: "common.invalid_request"}, nil
	}
	updates := map[string]any{}
	putString(updates, "name", req.GetName())
	putString(updates, "description", req.GetDescription())
	putString(updates, "transport", strings.ToLower(req.GetTransport()))
	putString(updates, "command_or_url", req.GetCommandOrUrl())
	if strings.TrimSpace(req.GetArgs()) != "" {
		updates["args"] = nullStringFrom(req.GetArgs(), true)
	}
	if strings.TrimSpace(req.GetEnvVars()) != "" {
		updates["env_vars"] = nullStringFrom(req.GetEnvVars(), true)
	}
	if req.GetTimeoutSecondsSet() {
		updates["timeout_seconds"] = req.GetTimeoutSeconds()
	}
	if req.GetIsEnabledSet() {
		updates["is_enabled"] = req.GetIsEnabled()
	}
	if err := s.validateMCPServerUpdate(ctx, req.GetId(), updates); err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.MCPServerResponse{Code: governanceNotFound, Msg: "common.not_found"}, nil
		}
		return &pb.MCPServerResponse{Code: governanceBadRequest, Msg: "common.invalid_request"}, nil
	}
	if len(updates) > 0 {
		result := s.db.WithContext(ctx).Model(&mcpServerRecord{}).Where("id = ?", req.GetId()).Updates(updates)
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 0 {
			return &pb.MCPServerResponse{Code: governanceNotFound, Msg: "common.not_found"}, nil
		}
	}
	return s.getMCPServerResponse(ctx, req.GetId())
}

func (s *NativeStore) DeleteMCPServer(ctx context.Context, req *pb.DeleteMCPServerRequest) (*pb.CommonResponse, error) {
	if err := s.assertMCPServerNotReleased(ctx, req.GetId()); err != nil {
		return &pb.CommonResponse{Code: governanceBadRequest, Msg: "common.invalid_request"}, nil
	}
	return rowsCommon(s.db.WithContext(ctx).Delete(&mcpServerRecord{}, req.GetId()), "success", "mcp server not found")
}

func (s *NativeStore) ListMCPToolPolicies(ctx context.Context, req *pb.ListMCPToolPoliciesRequest) (*pb.ListMCPToolPoliciesResponse, error) {
	page, pageSize := pageDefaults(req.GetPage(), req.GetPageSize())
	query := s.db.WithContext(ctx).Model(&mcpToolPolicyRecord{})
	if req.GetServerId() > 0 {
		query = query.Where("server_id = ?", req.GetServerId())
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []mcpToolPolicyRecord
	if err := query.Order("server_id ASC, tool_name ASC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]*pb.MCPToolPolicyInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, mcpPolicyToPB(row))
	}
	return &pb.ListMCPToolPoliciesResponse{Code: governanceOK, Msg: "common.success", Total: total, List: items}, nil
}

func (s *NativeStore) CreateMCPToolPolicy(ctx context.Context, req *pb.CreateMCPToolPolicyRequest) (*pb.MCPToolPolicyResponse, error) {
	if req.GetServerId() <= 0 || strings.TrimSpace(req.GetToolName()) == "" {
		return &pb.MCPToolPolicyResponse{Code: governanceBadRequest, Msg: "common.invalid_request"}, nil
	}
	row := mcpToolPolicyRecord{ServerID: req.GetServerId(), ToolName: strings.TrimSpace(req.GetToolName()), Effect: defaultString(strings.TrimSpace(req.GetEffect()), model.MCPPolicyDecisionAllow), RiskLevel: defaultString(strings.TrimSpace(req.GetRiskLevel()), "medium"), RequireConfirmation: req.GetRequireConfirmation(), AllowedRolesJSON: nullStringFrom(req.GetAllowedRolesJson(), true), AllowedScopesJSON: nullStringFrom(req.GetAllowedScopesJson(), true), RequiredArgsJSON: nullStringFrom(req.GetRequiredArgsJson(), true), DeniedArgsJSON: nullStringFrom(req.GetDeniedArgsJson(), true), ArgRulesJSON: nullStringFrom(req.GetArgRulesJson(), true), RedactFieldsJSON: nullStringFrom(req.GetRedactFieldsJson(), true), RateLimitWindowSeconds: int(req.GetRateLimitWindowSeconds()), RateLimitMaxCalls: int(req.GetRateLimitMaxCalls()), IsEnabled: true, CreatedByHRID: nullInt64From(req.GetOperatorHrId()), UpdatedByHRID: nullInt64From(req.GetOperatorHrId())}
	if req.GetIsEnabledSet() {
		row.IsEnabled = req.GetIsEnabled()
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return s.getMCPPolicyResponse(ctx, row.ID)
}

func (s *NativeStore) UpdateMCPToolPolicy(ctx context.Context, req *pb.UpdateMCPToolPolicyRequest) (*pb.MCPToolPolicyResponse, error) {
	if err := s.assertNotReleasedConfiguration(ctx, "mcp_policy", req.GetId()); err != nil {
		return &pb.MCPToolPolicyResponse{Code: governanceBadRequest, Msg: "common.invalid_request"}, nil
	}
	updates := map[string]any{"updated_by_hr_id": nullInt64From(req.GetOperatorHrId())}
	if req.GetServerId() > 0 {
		updates["server_id"] = req.GetServerId()
	}
	putString(updates, "tool_name", req.GetToolName())
	putString(updates, "effect", req.GetEffect())
	putString(updates, "risk_level", req.GetRiskLevel())
	if req.GetRequireConfirmationSet() {
		updates["require_confirmation"] = req.GetRequireConfirmation()
	}
	putMCPJSON(updates, "allowed_roles_json", req.GetAllowedRolesJson())
	putMCPJSON(updates, "allowed_scopes_json", req.GetAllowedScopesJson())
	putMCPJSON(updates, "required_args_json", req.GetRequiredArgsJson())
	putMCPJSON(updates, "denied_args_json", req.GetDeniedArgsJson())
	putMCPJSON(updates, "arg_rules_json", req.GetArgRulesJson())
	putMCPJSON(updates, "redact_fields_json", req.GetRedactFieldsJson())
	if req.GetRateLimitWindowSecondsSet() {
		updates["rate_limit_window_seconds"] = req.GetRateLimitWindowSeconds()
	}
	if req.GetRateLimitMaxCallsSet() {
		updates["rate_limit_max_calls"] = req.GetRateLimitMaxCalls()
	}
	if req.GetIsEnabledSet() {
		updates["is_enabled"] = req.GetIsEnabled()
	}
	result := s.db.WithContext(ctx).Model(&mcpToolPolicyRecord{}).Where("id = ?", req.GetId()).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return &pb.MCPToolPolicyResponse{Code: governanceNotFound, Msg: "common.not_found"}, nil
	}
	return s.getMCPPolicyResponse(ctx, req.GetId())
}

func (s *NativeStore) DeleteMCPToolPolicy(ctx context.Context, req *pb.DeleteMCPToolPolicyRequest) (*pb.CommonResponse, error) {
	if err := s.assertNotReleasedConfiguration(ctx, "mcp_policy", req.GetId()); err != nil {
		return &pb.CommonResponse{Code: governanceBadRequest, Msg: "common.invalid_request"}, nil
	}
	return rowsCommon(s.db.WithContext(ctx).Delete(&mcpToolPolicyRecord{}, req.GetId()), "success", "mcp tool policy not found")
}

func (s *NativeStore) ListMCPToolLogs(ctx context.Context, req *pb.ListMCPToolLogsRequest) (*pb.ListMCPToolLogsResponse, error) {
	page, pageSize := pageDefaults(req.GetPage(), req.GetPageSize())
	query := s.db.WithContext(ctx).Model(&mcpToolLogRecord{})
	if req.GetServerId() > 0 {
		query = query.Where("server_id = ?", req.GetServerId())
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []mcpToolLogRecord
	if err := query.Order("created_at DESC, id DESC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]*pb.MCPToolLogInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, mcpLogToPB(row))
	}
	return &pb.ListMCPToolLogsResponse{Code: governanceOK, Msg: "common.success", Total: total, List: items}, nil
}

func (s *NativeStore) TestMCPConnection(ctx context.Context, req *pb.TestMCPConnectionRequest) (*pb.TestMCPConnectionResponse, error) {
	resp := &pb.TestMCPConnectionResponse{Code: governanceUnsupported, Msg: "common.operation_failed", Success: false, Detail: "native MCP governance store can validate persisted configuration but has no MCP client runner bound"}
	if req.GetServerId() > 0 {
		_ = s.db.WithContext(ctx).Model(&mcpServerRecord{}).Where("id = ?", req.GetServerId()).Updates(map[string]any{"status": "disconnected", "last_error": resp.Detail}).Error
	}
	return resp, nil
}

func (s *NativeStore) ListMCPTools(context.Context, *pb.ListMCPToolsRequest) (*pb.ListMCPToolsResponse, error) {
	return &pb.ListMCPToolsResponse{Code: governanceUnsupported, Msg: "common.operation_failed"}, nil
}

func (s *NativeStore) CallMCPTool(context.Context, *pb.CallMCPToolRequest) (*pb.CallMCPToolResponse, error) {
	return &pb.CallMCPToolResponse{Code: governanceUnsupported, Msg: "common.operation_failed", ErrorMsg: "mcp client runner is not bound"}, nil
}

func (s *NativeStore) GetMCPRuntimeServer(ctx context.Context, serverID int64) (mcpinfra.ServerConfig, bool, error) {
	var row mcpServerRecord
	if err := s.db.WithContext(ctx).First(&row, serverID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return mcpinfra.ServerConfig{}, false, nil
		}
		return mcpinfra.ServerConfig{}, false, err
	}
	return mcpinfra.ServerConfig{
		ID:                  row.ID,
		Name:                row.Name,
		Transport:           row.Transport,
		CommandOrURL:        row.CommandOrURL,
		ArgsJSON:            nullString(row.Args),
		EnvVarsJSON:         nullString(row.EnvVars),
		TimeoutSeconds:      row.TimeoutSeconds,
		Enabled:             row.IsEnabled,
		AllowPrivateNetwork: false,
		AllowedCommands:     []string{"node", "npx", "python", "python3"},
	}, true, nil
}

func (s *NativeStore) UpdateMCPRuntimeStatus(ctx context.Context, serverID int64, status string, toolCount int, lastError string) error {
	updates := map[string]any{"status": strings.TrimSpace(status), "last_error": nullStringFrom(lastError, true)}
	if toolCount >= 0 {
		updates["tool_count"] = toolCount
	}
	return s.db.WithContext(ctx).Model(&mcpServerRecord{}).Where("id = ?", serverID).Updates(updates).Error
}

func (s *NativeStore) GetMCPRuntimeToolPolicy(ctx context.Context, serverID int64, toolName string) (*model.MCPToolPolicy, bool, error) {
	var row mcpToolPolicyRecord
	if err := s.db.WithContext(ctx).Where("server_id = ? AND tool_name = ? AND is_enabled = ?", serverID, strings.TrimSpace(toolName), true).Order("id DESC").First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	policyModel := &model.MCPToolPolicy{
		ID:                     uint64(row.ID),
		ServerID:               uint64(row.ServerID),
		ToolName:               row.ToolName,
		Enabled:                row.IsEnabled,
		Effect:                 row.Effect,
		RiskLevel:              row.RiskLevel,
		AllowedRoles:           stringListFromJSON(row.AllowedRolesJSON),
		AllowedScopes:          stringListFromJSON(row.AllowedScopesJSON),
		RequiredArgs:           stringListFromJSON(row.RequiredArgsJSON),
		DeniedArgs:             stringListFromJSON(row.DeniedArgsJSON),
		ArgRules:               argRulesFromJSON(row.ArgRulesJSON),
		RedactFields:           stringListFromJSON(row.RedactFieldsJSON),
		RequireConfirmation:    row.RequireConfirmation,
		RateLimitWindowSeconds: row.RateLimitWindowSeconds,
		RateLimitMaxCalls:      row.RateLimitMaxCalls,
	}
	return policyModel, true, nil
}

func (s *NativeStore) CountRecentMCPToolCalls(ctx context.Context, serverID int64, toolName string, since time.Time) (int64, error) {
	var total int64
	err := s.db.WithContext(ctx).Model(&mcpToolLogRecord{}).
		Where("server_id = ? AND tool_name = ? AND created_at >= ?", serverID, strings.TrimSpace(toolName), since).
		Count(&total).Error
	return total, err
}

func (s *NativeStore) AppendMCPToolLog(ctx context.Context, log mcpinfra.ToolLog) error {
	row := mcpToolLogRecord{
		ServerID:       log.ServerID,
		ToolName:       strings.TrimSpace(log.ToolName),
		ArgsJSON:       nullStringFrom(log.ArgsJSON, true),
		ResultContent:  nullStringFrom(log.ResultContent, true),
		DurationMs:     int(log.DurationMs),
		ErrorMsg:       nullStringFrom(log.ErrorMsg, true),
		CalledByHRID:   nullInt64From(log.CalledByHRID),
		SessionID:      nullInt64From(log.SessionID),
		PolicyID:       nullInt64From(log.PolicyID),
		PolicyDecision: strings.TrimSpace(log.PolicyDecision),
		PolicyReason:   nullStringFrom(log.PolicyReason, true),
	}
	return s.db.WithContext(ctx).Create(&row).Error
}

func (s *NativeStore) GetAgentSkill(ctx context.Context, req *pb.GetAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	return s.getAgentSkillResponse(ctx, req.GetId())
}

func (s *NativeStore) CreateAgentSkill(ctx context.Context, req *pb.CreateAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	compiled, authoringJSON, compileErr := compileAgentSkillPackage(req.GetPackage())
	if compileErr != nil {
		return &pb.AgentSkillResponse{Code: governanceBadRequest, Msg: agentSkillCompileErrorMessage(compileErr)}, nil
	}
	row := agentSkillRecord{
		Name:              compiled.Manifest.SkillName,
		DisplayName:       compiled.Manifest.DisplayName,
		Description:       nullStringFrom(compiled.Manifest.Description, true),
		IsEnabled:         true,
		IsManualInvocable: true,
		CreatedBy:         nullInt64From(req.GetActorUserId()),
		UpdatedBy:         nullInt64From(req.GetActorUserId()),
	}
	if req.GetIsEnabledSet() {
		row.IsEnabled = req.GetIsEnabled()
	}
	if req.GetIsManualInvocableSet() {
		row.IsManualInvocable = req.GetIsManualInvocable()
	}
	versionName := defaultString(strings.TrimSpace(req.GetVersion()), "v1")
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		version, err := persistCompiledAgentSkillVersion(tx, row.ID, versionName, req.GetChangeNote(), req.GetActorUserId(), authoringJSON, compiled)
		if err != nil {
			return err
		}
		if req.GetActivate() {
			return tx.Model(&agentSkillRecord{}).Where("id = ?", row.ID).
				Updates(map[string]any{"current_version_id": version.ID, "updated_by": nullInt64From(req.GetActorUserId())}).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.getAgentSkillResponse(ctx, row.ID)
}

func (s *NativeStore) UpdateAgentSkill(ctx context.Context, req *pb.UpdateAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	if req.GetId() <= 0 {
		return &pb.AgentSkillResponse{Code: governanceBadRequest, Msg: "common.invalid_request"}, nil
	}
	var existing agentSkillRecord
	if err := s.db.WithContext(ctx).First(&existing, req.GetId()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &pb.AgentSkillResponse{Code: governanceNotFound, Msg: "common.not_found"}, nil
		}
		return nil, err
	}
	updates := map[string]any{"updated_by": nullInt64From(req.GetActorUserId())}
	if req.GetDisplayNameSet() {
		displayName := strings.TrimSpace(req.GetDisplayName())
		if displayName == "" {
			return &pb.AgentSkillResponse{Code: governanceBadRequest, Msg: "common.invalid_request"}, nil
		}
		updates["display_name"] = displayName
	}
	if req.GetDescriptionSet() {
		updates["description"] = nullStringFrom(req.GetDescription(), true)
	}
	if req.GetIsEnabledSet() {
		updates["is_enabled"] = req.GetIsEnabled()
	}
	if req.GetIsManualInvocableSet() {
		updates["is_manual_invocable"] = req.GetIsManualInvocable()
	}
	result := s.db.WithContext(ctx).Model(&agentSkillRecord{}).Where("id = ?", req.GetId()).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	return s.getAgentSkillResponse(ctx, req.GetId())
}

func (s *NativeStore) CreateAgentSkillVersion(ctx context.Context, req *pb.CreateAgentSkillVersionRequest) (*pb.AgentSkillVersionResponse, error) {
	versionName := strings.TrimSpace(req.GetVersion())
	if req.GetSkillId() <= 0 || versionName == "" {
		return &pb.AgentSkillVersionResponse{Code: governanceBadRequest, Msg: "common.invalid_request"}, nil
	}
	compiled, authoringJSON, compileErr := compileAgentSkillPackage(req.GetPackage())
	if compileErr != nil {
		return &pb.AgentSkillVersionResponse{Code: governanceBadRequest, Msg: agentSkillCompileErrorMessage(compileErr)}, nil
	}
	var version agentSkillVersionRecord
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var registry agentSkillRecord
		if err := tx.First(&registry, req.GetSkillId()).Error; err != nil {
			return err
		}
		if compiled.Manifest.SkillName != registry.Name {
			return errAgentSkillNameMismatch
		}
		persisted, err := persistCompiledAgentSkillVersion(tx, registry.ID, versionName, req.GetChangeNote(), req.GetActorUserId(), authoringJSON, compiled)
		if err != nil {
			return err
		}
		version = persisted
		if req.GetActivate() {
			return tx.Model(&agentSkillRecord{}).Where("id = ?", req.GetSkillId()).
				Updates(map[string]any{"current_version_id": version.ID, "updated_by": nullInt64From(req.GetActorUserId())}).Error
		}
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &pb.AgentSkillVersionResponse{Code: governanceNotFound, Msg: "common.not_found"}, nil
	}
	if errors.Is(err, errAgentSkillNameMismatch) {
		return &pb.AgentSkillVersionResponse{Code: governanceBadRequest, Msg: agentskill.CodePackageInvalid}, nil
	}
	if err != nil {
		return nil, err
	}
	sections, err := s.loadAgentSkillVersionSections(ctx, []int64{version.ID})
	if err != nil {
		return nil, err
	}
	info, err := agentSkillVersionToPB(version, sections[version.ID])
	if err != nil {
		return nil, err
	}
	return &pb.AgentSkillVersionResponse{Code: governanceOK, Msg: "common.success", Version: info}, nil
}

func (s *NativeStore) ListAgentSkillVersions(ctx context.Context, req *pb.ListAgentSkillVersionsRequest) (*pb.ListAgentSkillVersionsResponse, error) {
	var rows []agentSkillVersionRecord
	if err := s.db.WithContext(ctx).Where("skill_id = ?", req.GetSkillId()).Order("created_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	versionIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		versionIDs = append(versionIDs, row.ID)
	}
	sections, err := s.loadAgentSkillVersionSections(ctx, versionIDs)
	if err != nil {
		return nil, err
	}
	items := make([]*pb.AgentSkillVersionInfo, 0, len(rows))
	for _, row := range rows {
		item, err := agentSkillVersionToPB(row, sections[row.ID])
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return &pb.ListAgentSkillVersionsResponse{Code: governanceOK, Msg: "common.success", List: items}, nil
}

func (s *NativeStore) ActivateAgentSkillVersion(ctx context.Context, req *pb.ActivateAgentSkillVersionRequest) (*pb.AgentSkillResponse, error) {
	if req.GetSkillId() <= 0 || req.GetVersionId() <= 0 {
		return &pb.AgentSkillResponse{Code: governanceBadRequest, Msg: "common.invalid_request"}, nil
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var registry agentSkillRecord
		if err := tx.First(&registry, req.GetSkillId()).Error; err != nil {
			return err
		}
		var version agentSkillVersionRecord
		if err := tx.Where("id = ? AND skill_id = ?", req.GetVersionId(), req.GetSkillId()).First(&version).Error; err != nil {
			return err
		}
		result := tx.Model(&agentSkillRecord{}).Where("id = ?", req.GetSkillId()).Updates(map[string]any{"current_version_id": version.ID, "updated_by": nullInt64From(req.GetActorUserId())})
		return result.Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &pb.AgentSkillResponse{Code: governanceBadRequest, Msg: "common.invalid_request"}, nil
	}
	if err != nil {
		return nil, err
	}
	return s.getAgentSkillResponse(ctx, req.GetSkillId())
}

func (s *NativeStore) UpdateAgentSkillStatus(ctx context.Context, req *pb.UpdateAgentSkillStatusRequest) (*pb.AgentSkillResponse, error) {
	if req.GetId() <= 0 {
		return &pb.AgentSkillResponse{Code: governanceBadRequest, Msg: "common.invalid_request"}, nil
	}
	var existing agentSkillRecord
	if err := s.db.WithContext(ctx).First(&existing, req.GetId()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &pb.AgentSkillResponse{Code: governanceNotFound, Msg: "common.not_found"}, nil
		}
		return nil, err
	}
	result := s.db.WithContext(ctx).Model(&agentSkillRecord{}).Where("id = ?", req.GetId()).Updates(map[string]any{"is_enabled": req.GetIsEnabled(), "updated_by": nullInt64From(req.GetActorUserId())})
	if result.Error != nil {
		return nil, result.Error
	}
	return s.getAgentSkillResponse(ctx, req.GetId())
}

func (s *NativeStore) PreviewAgentSkill(_ context.Context, req *pb.PreviewAgentSkillRequest) (*pb.PreviewAgentSkillResponse, error) {
	compiled, authoringJSON, err := compileAgentSkillPackage(req.GetPackage())
	if err != nil {
		return &pb.PreviewAgentSkillResponse{Code: governanceBadRequest, Msg: agentSkillCompileErrorMessage(err)}, nil
	}
	return &pb.PreviewAgentSkillResponse{
		Code:    governanceOK,
		Msg:     "common.success",
		Package: compiledAgentSkillPackageToPB(compiled, authoringJSON, nil),
	}, nil
}

func (s *NativeStore) DebugSemanticRetrieval(context.Context, *pb.DebugSemanticRetrievalRequest) (*pb.DebugSemanticRetrievalResponse, error) {
	return &pb.DebugSemanticRetrievalResponse{Code: governanceUnsupported, Msg: "common.operation_failed", EmbeddingAvailable: false, FallbackReason: "embedding query runner is not bound"}, nil
}

func (s *NativeStore) getMCPServerResponse(ctx context.Context, id int64) (*pb.MCPServerResponse, error) {
	var row mcpServerRecord
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.MCPServerResponse{Code: governanceNotFound, Msg: "common.not_found"}, nil
		}
		return nil, err
	}
	return &pb.MCPServerResponse{Code: governanceOK, Msg: "common.success", Server: mcpServerToPB(row)}, nil
}

func (s *NativeStore) getMCPPolicyResponse(ctx context.Context, id int64) (*pb.MCPToolPolicyResponse, error) {
	var row mcpToolPolicyRecord
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.MCPToolPolicyResponse{Code: governanceNotFound, Msg: "common.not_found"}, nil
		}
		return nil, err
	}
	return &pb.MCPToolPolicyResponse{Code: governanceOK, Msg: "common.success", Policy: mcpPolicyToPB(row)}, nil
}

func (s *NativeStore) getAgentSkillResponse(ctx context.Context, id int64) (*pb.AgentSkillResponse, error) {
	var row agentSkillRecord
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.AgentSkillResponse{Code: governanceNotFound, Msg: "common.not_found"}, nil
		}
		return nil, err
	}
	summaries, err := s.loadAgentSkillCurrentVersionSummaries(ctx, []agentSkillRecord{row})
	if err != nil {
		return nil, err
	}
	return &pb.AgentSkillResponse{
		Code:  governanceOK,
		Msg:   "common.success",
		Skill: agentSkillToPB(row, summaries[row.ID]),
	}, nil
}

func (s *NativeStore) validateMCPServerUpdate(ctx context.Context, id int64, updates map[string]any) error {
	var current mcpServerRecord
	if err := s.db.WithContext(ctx).First(&current, id).Error; err != nil {
		return err
	}
	transport := current.Transport
	commandOrURL := current.CommandOrURL
	timeout := current.TimeoutSeconds
	if value, ok := updates["transport"].(string); ok && strings.TrimSpace(value) != "" {
		transport = value
	}
	if value, ok := updates["command_or_url"].(string); ok && strings.TrimSpace(value) != "" {
		commandOrURL = value
	}
	if value, ok := updates["timeout_seconds"].(int32); ok {
		timeout = int(value)
	}
	return validateMCPConfig(transport, commandOrURL, timeout)
}

func validateMCPConfig(transport, commandOrURL string, timeout int) error {
	cfg := model.MCPServerConfig{Transport: strings.ToLower(strings.TrimSpace(transport)), TimeoutSeconds: timeout}
	if cfg.Transport == model.MCPTransportStdio {
		cfg.Command = commandOrURL
	} else {
		cfg.URL = commandOrURL
	}
	return policy.ValidateMCPServerConfig(cfg)
}

func mcpServerToPB(row mcpServerRecord) *pb.MCPServerInfo {
	return &pb.MCPServerInfo{Id: row.ID, Name: row.Name, Description: nullString(row.Description), Transport: row.Transport, CommandOrUrl: row.CommandOrURL, Args: redactSensitiveJSON(nullString(row.Args)), EnvVars: redactHeaders(nullString(row.EnvVars)), TimeoutSeconds: int32(row.TimeoutSeconds), IsEnabled: row.IsEnabled, Status: row.Status, ToolCount: int32(row.ToolCount), LastError: nullString(row.LastError), CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt)}
}

func mcpPolicyToPB(row mcpToolPolicyRecord) *pb.MCPToolPolicyInfo {
	return &pb.MCPToolPolicyInfo{Id: row.ID, ServerId: row.ServerID, ToolName: row.ToolName, Effect: row.Effect, RiskLevel: row.RiskLevel, RequireConfirmation: row.RequireConfirmation, AllowedRolesJson: nullString(row.AllowedRolesJSON), AllowedScopesJson: nullString(row.AllowedScopesJSON), RequiredArgsJson: nullString(row.RequiredArgsJSON), DeniedArgsJson: redactSensitiveJSON(nullString(row.DeniedArgsJSON)), ArgRulesJson: redactSensitiveJSON(nullString(row.ArgRulesJSON)), RedactFieldsJson: nullString(row.RedactFieldsJSON), RateLimitWindowSeconds: int32(row.RateLimitWindowSeconds), RateLimitMaxCalls: int32(row.RateLimitMaxCalls), IsEnabled: row.IsEnabled, CreatedByHrId: nullInt64(row.CreatedByHRID), UpdatedByHrId: nullInt64(row.UpdatedByHRID), CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt)}
}

func mcpLogToPB(row mcpToolLogRecord) *pb.MCPToolLogInfo {
	return &pb.MCPToolLogInfo{Id: row.ID, ServerId: row.ServerID, ToolName: row.ToolName, ArgsJson: redactSensitiveJSON(nullString(row.ArgsJSON)), ResultContent: truncateForLog(redactSensitiveText(nullString(row.ResultContent))), DurationMs: int32(row.DurationMs), ErrorMsg: redactSensitiveText(nullString(row.ErrorMsg)), CalledByHrId: nullInt64(row.CalledByHRID), SessionId: nullInt64(row.SessionID), PolicyId: nullInt64(row.PolicyID), PolicyDecision: row.PolicyDecision, PolicyReason: redactSensitiveText(nullString(row.PolicyReason)), CreatedAt: formatTime(row.CreatedAt)}
}

func agentSkillToPB(row agentSkillRecord, current *pb.AgentSkillVersionSummary) *pb.AgentSkillInfo {
	return &pb.AgentSkillInfo{
		Id:                row.ID,
		Name:              row.Name,
		DisplayName:       row.DisplayName,
		Description:       nullString(row.Description),
		CurrentVersionId:  nullInt64(row.CurrentVersionID),
		IsEnabled:         row.IsEnabled,
		IsManualInvocable: row.IsManualInvocable,
		CreatedAt:         formatTime(row.CreatedAt),
		UpdatedAt:         formatTime(row.UpdatedAt),
		CurrentVersion:    current,
	}
}

func agentSkillVersionToPB(row agentSkillVersionRecord, sections []agentSkillSectionRecord) (*pb.AgentSkillVersionInfo, error) {
	var manifest agentskill.Manifest
	if err := json.Unmarshal([]byte(row.ManifestJSON), &manifest); err != nil {
		return nil, fmt.Errorf("decode agent skill version %d manifest: %w", row.ID, err)
	}
	sectionInfos := make([]*pb.AgentSkillSectionInfo, 0, len(sections))
	for _, section := range sections {
		sectionInfos = append(sectionInfos, agentSkillSectionToPB(section))
	}
	return &pb.AgentSkillVersionInfo{
		Id:         row.ID,
		SkillId:    row.SkillID,
		Version:    row.Version,
		ChangeNote: nullString(row.ChangeNote),
		CreatedAt:  formatTime(row.CreatedAt),
		Package: &pb.AgentSkillPackageInfo{
			Manifest:               agentSkillManifestToPB(manifest),
			CoreMarkdown:           row.CoreMarkdown,
			Sections:               sectionInfos,
			CompiledMarkdown:       row.CompiledMarkdown,
			AuthoringJson:          nullString(row.AuthoringJSON),
			CompiledHash:           row.CompiledHash,
			CoreEstimatedTokens:    int32(row.CoreEstimatedTokens),
			PackageEstimatedTokens: int32(contextbudget.EstimateTokensConservative(row.CompiledMarkdown)),
		},
	}, nil
}

func (s *NativeStore) loadAgentSkillCurrentVersionSummaries(ctx context.Context, skills []agentSkillRecord) (map[int64]*pb.AgentSkillVersionSummary, error) {
	ids := make([]int64, 0, len(skills))
	for _, skill := range skills {
		if skill.CurrentVersionID.Valid && skill.CurrentVersionID.Int64 > 0 {
			ids = append(ids, skill.CurrentVersionID.Int64)
		}
	}
	result := make(map[int64]*pb.AgentSkillVersionSummary, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var versions []agentSkillVersionRecord
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&versions).Error; err != nil {
		return nil, err
	}
	versionsByID := make(map[int64]agentSkillVersionRecord, len(versions))
	for _, version := range versions {
		versionsByID[version.ID] = version
	}
	for _, skill := range skills {
		if !skill.CurrentVersionID.Valid {
			continue
		}
		version, ok := versionsByID[skill.CurrentVersionID.Int64]
		if !ok || version.SkillID != skill.ID {
			continue
		}
		summary, err := agentSkillVersionSummaryToPB(version)
		if err != nil {
			return nil, err
		}
		result[skill.ID] = summary
	}
	return result, nil
}

func (s *NativeStore) loadAgentSkillVersionSections(ctx context.Context, versionIDs []int64) (map[int64][]agentSkillSectionRecord, error) {
	result := make(map[int64][]agentSkillSectionRecord, len(versionIDs))
	if len(versionIDs) == 0 {
		return result, nil
	}
	var sections []agentSkillSectionRecord
	if err := s.db.WithContext(ctx).
		Where("skill_version_id IN ?", versionIDs).
		Order("skill_version_id ASC, ordinal ASC, section_key ASC, id ASC").
		Find(&sections).Error; err != nil {
		return nil, err
	}
	for _, section := range sections {
		result[section.SkillVersionID] = append(result[section.SkillVersionID], section)
	}
	return result, nil
}

func agentSkillVersionSummaryToPB(row agentSkillVersionRecord) (*pb.AgentSkillVersionSummary, error) {
	var manifest agentskill.Manifest
	if err := json.Unmarshal([]byte(row.ManifestJSON), &manifest); err != nil {
		return nil, fmt.Errorf("decode agent skill version %d manifest summary: %w", row.ID, err)
	}
	return &pb.AgentSkillVersionSummary{
		VersionId:              row.ID,
		Version:                row.Version,
		CompiledHash:           row.CompiledHash,
		AgentType:              manifest.AgentType,
		Category:               manifest.Category,
		Scenario:               manifest.Scenario,
		Priority:               int32(manifest.Priority),
		Risk:                   agentSkillRiskToPB(manifest.RiskLevel),
		ActivationPolicy:       agentSkillActivationPolicyToPB(manifest.ActivationPolicy),
		CompositionRole:        agentSkillCompositionRoleToPB(manifest.Composition.Role),
		CoreEstimatedTokens:    int32(row.CoreEstimatedTokens),
		PackageEstimatedTokens: int32(contextbudget.EstimateTokensConservative(row.CompiledMarkdown)),
	}, nil
}

func compileAgentSkillPackage(input *pb.AgentSkillPackageDraft) (*agentskill.CompiledPackage, string, error) {
	if input == nil || input.GetManifest() == nil {
		return nil, "", &agentskill.CompileError{
			Code:    agentskill.CodePackageInvalid,
			Field:   "package",
			Message: "is required",
		}
	}
	authoringJSON := strings.TrimSpace(input.GetAuthoringJson())
	if authoringJSON != "" && !json.Valid([]byte(authoringJSON)) {
		return nil, "", &agentskill.CompileError{
			Code:    agentskill.CodePackageInvalid,
			Field:   "package.authoring_json",
			Message: "must be valid JSON",
		}
	}
	draft := agentskill.PackageDraft{
		Manifest: agentSkillManifestFromPB(input.GetManifest()),
		Core:     agentskill.Core{ContentMarkdown: input.GetCoreMarkdown()},
		Sections: make([]agentskill.ReferenceSection, 0, len(input.GetSections())),
	}
	for _, section := range input.GetSections() {
		if section == nil {
			return nil, "", &agentskill.CompileError{
				Code:    agentskill.CodeSectionInvalid,
				Field:   "package.sections",
				Message: "must not contain null sections",
			}
		}
		draft.Sections = append(draft.Sections, agentskill.ReferenceSection{
			SectionKey:      section.GetSectionKey(),
			Title:           section.GetTitle(),
			Description:     section.GetDescription(),
			ContentMarkdown: section.GetContentMarkdown(),
			TriggerTerms:    append([]string(nil), section.GetTriggerTerms()...),
			SemanticTags:    append([]string(nil), section.GetSemanticTags()...),
			PlannerIntents:  append([]string(nil), section.GetPlannerIntents()...),
			Priority:        int(section.GetPriority()),
			Ordinal:         int(section.GetOrdinal()),
		})
	}
	compiled, err := agentskill.Compile(draft)
	return compiled, authoringJSON, err
}

func agentSkillManifestFromPB(input *pb.AgentSkillManifest) agentskill.Manifest {
	var schema json.RawMessage
	if contract := input.GetOutputContract(); contract != nil && strings.TrimSpace(contract.GetSchemaJson()) != "" {
		schema = json.RawMessage(contract.GetSchemaJson())
	}
	return agentskill.Manifest{
		SchemaVersion:        int(input.GetSchemaVersion()),
		SkillName:            input.GetSkillName(),
		DisplayName:          input.GetDisplayName(),
		Description:          input.GetDescription(),
		AgentType:            input.GetAgentType(),
		Category:             input.GetCategory(),
		Scenario:             input.GetScenario(),
		Priority:             int(input.GetPriority()),
		RiskLevel:            agentSkillRiskFromPB(input.GetRisk()),
		ActivationPolicy:     agentSkillActivationPolicyFromPB(input.GetActivationPolicy()),
		RequiredCapabilities: append([]string(nil), input.GetRequiredCapabilities()...),
		TriggerKeywords:      append([]string(nil), input.GetTriggerKeywords()...),
		SemanticTags:         append([]string(nil), input.GetSemanticTags()...),
		Composition: agentskill.Composition{
			Role: agentSkillCompositionRoleFromPB(input.GetComposition().GetRole()),
		},
		OutputContract: agentskill.OutputContract{
			Mode:     agentSkillOutputModeFromPB(input.GetOutputContract().GetMode()),
			SchemaID: input.GetOutputContract().GetSchemaId(),
			Schema:   schema,
		},
		EvaluationCriteria: append([]string(nil), input.GetEvaluationCriteria()...),
	}
}

func persistCompiledAgentSkillVersion(
	tx *gorm.DB,
	skillID int64,
	versionName, changeNote string,
	actorUserID int64,
	authoringJSON string,
	compiled *agentskill.CompiledPackage,
) (agentSkillVersionRecord, error) {
	version := agentSkillVersionRecord{
		SkillID:             skillID,
		Version:             versionName,
		ManifestJSON:        compiled.ManifestJSON,
		CoreMarkdown:        compiled.Core.ContentMarkdown,
		CompiledMarkdown:    compiled.CompiledMarkdown,
		AuthoringJSON:       nullStringFrom(authoringJSON, false),
		CompiledHash:        compiled.CompiledHash,
		CoreEstimatedTokens: compiled.Core.EstimatedTokens,
		ChangeNote:          nullStringFrom(changeNote, true),
		CreatedBy:           nullInt64From(actorUserID),
	}
	if err := tx.Create(&version).Error; err != nil {
		return agentSkillVersionRecord{}, err
	}
	if len(compiled.Sections) == 0 {
		return version, nil
	}
	sections := make([]agentSkillSectionRecord, 0, len(compiled.Sections))
	for _, section := range compiled.Sections {
		sections = append(sections, agentSkillSectionRecord{
			SkillVersionID:     version.ID,
			SectionKey:         section.SectionKey,
			Title:              section.Title,
			Description:        nullStringFrom(section.Description, true),
			ContentMarkdown:    section.ContentMarkdown,
			TriggerTermsJSON:   jsonListNull(section.TriggerTerms),
			SemanticTagsJSON:   jsonListNull(section.SemanticTags),
			PlannerIntentsJSON: jsonListNull(section.PlannerIntents),
			Priority:           section.Priority,
			Ordinal:            section.Ordinal,
			EstimatedTokens:    section.EstimatedTokens,
			ContentHash:        section.ContentHash,
		})
	}
	if err := tx.Create(&sections).Error; err != nil {
		return agentSkillVersionRecord{}, err
	}
	return version, nil
}

func compiledAgentSkillPackageToPB(compiled *agentskill.CompiledPackage, authoringJSON string, sectionIDs map[string]int64) *pb.AgentSkillPackageInfo {
	sections := make([]*pb.AgentSkillSectionInfo, 0, len(compiled.Sections))
	for _, section := range compiled.Sections {
		sections = append(sections, &pb.AgentSkillSectionInfo{
			Id:              sectionIDs[section.SectionKey],
			SectionKey:      section.SectionKey,
			Title:           section.Title,
			Description:     section.Description,
			ContentMarkdown: section.ContentMarkdown,
			TriggerTerms:    append([]string(nil), section.TriggerTerms...),
			SemanticTags:    append([]string(nil), section.SemanticTags...),
			PlannerIntents:  append([]string(nil), section.PlannerIntents...),
			Priority:        int32(section.Priority),
			Ordinal:         int32(section.Ordinal),
			EstimatedTokens: int32(section.EstimatedTokens),
			ContentHash:     section.ContentHash,
		})
	}
	return &pb.AgentSkillPackageInfo{
		Manifest:               agentSkillManifestToPB(compiled.Manifest),
		CoreMarkdown:           compiled.Core.ContentMarkdown,
		Sections:               sections,
		CompiledMarkdown:       compiled.CompiledMarkdown,
		AuthoringJson:          authoringJSON,
		CompiledHash:           compiled.CompiledHash,
		CoreEstimatedTokens:    int32(compiled.Core.EstimatedTokens),
		PackageEstimatedTokens: int32(compiled.EstimatedTokens),
	}
}

func agentSkillSectionToPB(section agentSkillSectionRecord) *pb.AgentSkillSectionInfo {
	return &pb.AgentSkillSectionInfo{
		Id:              section.ID,
		SectionKey:      section.SectionKey,
		Title:           section.Title,
		Description:     nullString(section.Description),
		ContentMarkdown: section.ContentMarkdown,
		TriggerTerms:    jsonStringList(section.TriggerTermsJSON),
		SemanticTags:    jsonStringList(section.SemanticTagsJSON),
		PlannerIntents:  jsonStringList(section.PlannerIntentsJSON),
		Priority:        int32(section.Priority),
		Ordinal:         int32(section.Ordinal),
		EstimatedTokens: int32(section.EstimatedTokens),
		ContentHash:     section.ContentHash,
	}
}

func agentSkillManifestToPB(manifest agentskill.Manifest) *pb.AgentSkillManifest {
	return &pb.AgentSkillManifest{
		SchemaVersion:        int32(manifest.SchemaVersion),
		SkillName:            manifest.SkillName,
		DisplayName:          manifest.DisplayName,
		Description:          manifest.Description,
		AgentType:            manifest.AgentType,
		Category:             manifest.Category,
		Scenario:             manifest.Scenario,
		Priority:             int32(manifest.Priority),
		Risk:                 agentSkillRiskToPB(manifest.RiskLevel),
		ActivationPolicy:     agentSkillActivationPolicyToPB(manifest.ActivationPolicy),
		RequiredCapabilities: append([]string(nil), manifest.RequiredCapabilities...),
		TriggerKeywords:      append([]string(nil), manifest.TriggerKeywords...),
		SemanticTags:         append([]string(nil), manifest.SemanticTags...),
		Composition: &pb.AgentSkillComposition{
			Role: agentSkillCompositionRoleToPB(manifest.Composition.Role),
		},
		OutputContract: &pb.AgentSkillOutputContract{
			Mode:       agentSkillOutputModeToPB(manifest.OutputContract.Mode),
			SchemaId:   manifest.OutputContract.SchemaID,
			SchemaJson: string(manifest.OutputContract.Schema),
		},
		EvaluationCriteria: append([]string(nil), manifest.EvaluationCriteria...),
	}
}

func agentSkillCompileErrorMessage(err error) string {
	if code := agentskill.ErrorCode(err); code != "" {
		return code
	}
	return "common.invalid_request"
}

func agentSkillRiskFromPB(value pb.AgentSkillRiskLevel) agentskill.RiskLevel {
	switch value {
	case pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_LOW:
		return agentskill.RiskLevelLow
	case pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_MEDIUM:
		return agentskill.RiskLevelMedium
	case pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_HIGH:
		return agentskill.RiskLevelHigh
	case pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_CRITICAL:
		return agentskill.RiskLevelCritical
	default:
		return ""
	}
}

func agentSkillRiskToPB(value agentskill.RiskLevel) pb.AgentSkillRiskLevel {
	switch value {
	case agentskill.RiskLevelLow:
		return pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_LOW
	case agentskill.RiskLevelMedium:
		return pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_MEDIUM
	case agentskill.RiskLevelHigh:
		return pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_HIGH
	case agentskill.RiskLevelCritical:
		return pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_CRITICAL
	default:
		return pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_UNSPECIFIED
	}
}

func agentSkillActivationPolicyFromPB(value pb.AgentSkillActivationPolicy) agentskill.ActivationPolicy {
	switch value {
	case pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_AUTO:
		return agentskill.ActivationPolicyAuto
	case pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_CONFIRM:
		return agentskill.ActivationPolicyConfirm
	case pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_MANUAL_ONLY:
		return agentskill.ActivationPolicyManualOnly
	default:
		return ""
	}
}

func agentSkillActivationPolicyToPB(value agentskill.ActivationPolicy) pb.AgentSkillActivationPolicy {
	switch value {
	case agentskill.ActivationPolicyAuto:
		return pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_AUTO
	case agentskill.ActivationPolicyConfirm:
		return pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_CONFIRM
	case agentskill.ActivationPolicyManualOnly:
		return pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_MANUAL_ONLY
	default:
		return pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_UNSPECIFIED
	}
}

func agentSkillCompositionRoleFromPB(value pb.AgentSkillCompositionRole) agentskill.CompositionRole {
	switch value {
	case pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_PRIMARY:
		return agentskill.CompositionRolePrimary
	case pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_SUPPORTING:
		return agentskill.CompositionRoleSupporting
	default:
		return ""
	}
}

func agentSkillCompositionRoleToPB(value agentskill.CompositionRole) pb.AgentSkillCompositionRole {
	switch value {
	case agentskill.CompositionRolePrimary:
		return pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_PRIMARY
	case agentskill.CompositionRoleSupporting:
		return pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_SUPPORTING
	default:
		return pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_UNSPECIFIED
	}
}

func agentSkillOutputModeFromPB(value pb.AgentSkillOutputMode) agentskill.OutputMode {
	switch value {
	case pb.AgentSkillOutputMode_AGENT_SKILL_OUTPUT_MODE_NONE:
		return agentskill.OutputModeNone
	case pb.AgentSkillOutputMode_AGENT_SKILL_OUTPUT_MODE_ADVISORY:
		return agentskill.OutputModeAdvisory
	case pb.AgentSkillOutputMode_AGENT_SKILL_OUTPUT_MODE_STRICT:
		return agentskill.OutputModeStrict
	default:
		return ""
	}
}

func agentSkillOutputModeToPB(value agentskill.OutputMode) pb.AgentSkillOutputMode {
	switch value {
	case agentskill.OutputModeNone:
		return pb.AgentSkillOutputMode_AGENT_SKILL_OUTPUT_MODE_NONE
	case agentskill.OutputModeAdvisory:
		return pb.AgentSkillOutputMode_AGENT_SKILL_OUTPUT_MODE_ADVISORY
	case agentskill.OutputModeStrict:
		return pb.AgentSkillOutputMode_AGENT_SKILL_OUTPUT_MODE_STRICT
	default:
		return pb.AgentSkillOutputMode_AGENT_SKILL_OUTPUT_MODE_UNSPECIFIED
	}
}

func putMCPJSON(updates map[string]any, key, value string) {
	if strings.TrimSpace(value) != "" {
		updates[key] = nullStringFrom(value, true)
	}
}

func stringListFromJSON(value sql.NullString) []string {
	return jsonStringList(value)
}

func argRulesFromJSON(value sql.NullString) map[string]model.MCPArgRule {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}
	var rules map[string]model.MCPArgRule
	if err := json.Unmarshal([]byte(value.String), &rules); err != nil {
		return nil
	}
	return rules
}

func jsonListNull(items []string) sql.NullString {
	cleaned := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			cleaned = append(cleaned, strings.TrimSpace(item))
		}
	}
	if len(cleaned) == 0 {
		return sql.NullString{}
	}
	data, _ := json.Marshal(cleaned)
	return sql.NullString{String: string(data), Valid: true}
}

func redactSensitiveJSON(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	var parsed any
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		return redactSensitiveText(value)
	}
	redacted := redactJSONValue(parsed)
	data, err := json.Marshal(redacted)
	if err != nil {
		return `{"redacted":true}`
	}
	return string(data)
}

func redactJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			if isSensitiveKey(key) {
				result[key] = "[redacted]"
				continue
			}
			result[key] = redactJSONValue(item)
		}
		return result
	case []any:
		for i, item := range typed {
			typed[i] = redactJSONValue(item)
		}
	}
	return value
}

func redactSensitiveText(value string) string {
	lower := strings.ToLower(value)
	for _, marker := range []string{"api_key", "apikey", "token", "secret", "password", "authorization", "credential", "cookie"} {
		if strings.Contains(lower, marker) {
			return "[redacted]"
		}
	}
	return value
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	return strings.Contains(lower, "key") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") || strings.Contains(lower, "authorization") || strings.Contains(lower, "credential") || strings.Contains(lower, "cookie")
}

func truncateForLog(value string) string {
	const maxLen = 2048
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen] + "...[truncated]"
}
