package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

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

type aiSkillRecord struct {
	ID               int64          `gorm:"primaryKey"`
	Name             string         `gorm:"column:name"`
	DisplayName      string         `gorm:"column:display_name"`
	Description      sql.NullString `gorm:"column:description"`
	SourceType       string         `gorm:"column:source_type"`
	SourceURI        sql.NullString `gorm:"column:source_uri"`
	CurrentVersionID sql.NullInt64  `gorm:"column:current_version_id"`
	IsEnabled        bool           `gorm:"column:is_enabled"`
	CreatedAt        time.Time      `gorm:"column:created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at"`
}

func (aiSkillRecord) TableName() string { return "ai_skills" }

type aiSkillVersionRecord struct {
	ID               int64          `gorm:"primaryKey"`
	SkillID          int64          `gorm:"column:skill_id"`
	Version          string         `gorm:"column:version"`
	ManifestJSON     string         `gorm:"column:manifest_json"`
	Instruction      sql.NullString `gorm:"column:instruction"`
	InputSchemaJSON  sql.NullString `gorm:"column:input_schema_json"`
	OutputSchemaJSON sql.NullString `gorm:"column:output_schema_json"`
	RuntimeType      string         `gorm:"column:runtime_type"`
	CreatedAt        time.Time      `gorm:"column:created_at"`
}

func (aiSkillVersionRecord) TableName() string { return "ai_skill_versions" }

type aiSkillToolRecord struct {
	ID                int64          `gorm:"primaryKey"`
	SkillVersionID    int64          `gorm:"column:skill_version_id"`
	ToolName          string         `gorm:"column:tool_name"`
	Description       sql.NullString `gorm:"column:description"`
	InputSchemaJSON   sql.NullString `gorm:"column:input_schema_json"`
	RuntimeConfigJSON sql.NullString `gorm:"column:runtime_config_json"`
	IsEnabled         bool           `gorm:"column:is_enabled"`
	CreatedAt         time.Time      `gorm:"column:created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at"`
}

func (aiSkillToolRecord) TableName() string { return "ai_skill_tools" }

type agentSkillVersionRecord struct {
	ID              int64          `gorm:"primaryKey"`
	SkillID         int64          `gorm:"column:skill_id"`
	Version         string         `gorm:"column:version"`
	FlowJSON        sql.NullString `gorm:"column:flow_json"`
	SkillMD         string         `gorm:"column:skill_md"`
	FrontmatterJSON sql.NullString `gorm:"column:frontmatter_json"`
	BodyMarkdown    sql.NullString `gorm:"column:body_markdown"`
	ChangeNote      sql.NullString `gorm:"column:change_note"`
	CreatedBy       sql.NullInt64  `gorm:"column:created_by"`
	CreatedAt       time.Time      `gorm:"column:created_at"`
}

func (agentSkillVersionRecord) TableName() string { return "agent_skill_versions" }

func (s *NativeStore) CreateMCPServer(ctx context.Context, req *pb.CreateMCPServerRequest) (*pb.MCPServerResponse, error) {
	transport := strings.ToLower(strings.TrimSpace(req.GetTransport()))
	if err := validateMCPConfig(transport, req.GetCommandOrUrl(), int(req.GetTimeoutSeconds())); err != nil {
		return &pb.MCPServerResponse{Code: governanceBadRequest, Msg: err.Error()}, nil
	}
	row := mcpServerRecord{Name: strings.TrimSpace(req.GetName()), Description: nullStringFrom(req.GetDescription(), true), Transport: transport, CommandOrURL: strings.TrimSpace(req.GetCommandOrUrl()), Args: nullStringFrom(req.GetArgs(), true), EnvVars: nullStringFrom(req.GetEnvVars(), true), TimeoutSeconds: defaultInt32(req.GetTimeoutSeconds(), 30), IsEnabled: false, Status: "disconnected"}
	if row.Name == "" {
		return &pb.MCPServerResponse{Code: governanceBadRequest, Msg: "server name is required"}, nil
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return s.getMCPServerResponse(ctx, row.ID)
}

func (s *NativeStore) UpdateMCPServer(ctx context.Context, req *pb.UpdateMCPServerRequest) (*pb.MCPServerResponse, error) {
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
			return &pb.MCPServerResponse{Code: governanceNotFound, Msg: "mcp server not found"}, nil
		}
		return &pb.MCPServerResponse{Code: governanceBadRequest, Msg: err.Error()}, nil
	}
	if len(updates) > 0 {
		result := s.db.WithContext(ctx).Model(&mcpServerRecord{}).Where("id = ?", req.GetId()).Updates(updates)
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 0 {
			return &pb.MCPServerResponse{Code: governanceNotFound, Msg: "mcp server not found"}, nil
		}
	}
	return s.getMCPServerResponse(ctx, req.GetId())
}

func (s *NativeStore) DeleteMCPServer(ctx context.Context, req *pb.DeleteMCPServerRequest) (*pb.CommonResponse, error) {
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
	return &pb.ListMCPToolPoliciesResponse{Code: governanceOK, Msg: "success", Total: total, List: items}, nil
}

func (s *NativeStore) CreateMCPToolPolicy(ctx context.Context, req *pb.CreateMCPToolPolicyRequest) (*pb.MCPToolPolicyResponse, error) {
	if req.GetServerId() <= 0 || strings.TrimSpace(req.GetToolName()) == "" {
		return &pb.MCPToolPolicyResponse{Code: governanceBadRequest, Msg: "server_id and tool_name are required"}, nil
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
		return &pb.MCPToolPolicyResponse{Code: governanceNotFound, Msg: "mcp tool policy not found"}, nil
	}
	return s.getMCPPolicyResponse(ctx, req.GetId())
}

func (s *NativeStore) DeleteMCPToolPolicy(ctx context.Context, req *pb.DeleteMCPToolPolicyRequest) (*pb.CommonResponse, error) {
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
	return &pb.ListMCPToolLogsResponse{Code: governanceOK, Msg: "success", Total: total, List: items}, nil
}

func (s *NativeStore) TestMCPConnection(ctx context.Context, req *pb.TestMCPConnectionRequest) (*pb.TestMCPConnectionResponse, error) {
	resp := &pb.TestMCPConnectionResponse{Code: governanceUnsupported, Msg: "mcp runtime validation is not configured", Success: false, Detail: "native MCP governance store can validate persisted configuration but has no MCP client runner bound"}
	if req.GetServerId() > 0 {
		_ = s.db.WithContext(ctx).Model(&mcpServerRecord{}).Where("id = ?", req.GetServerId()).Updates(map[string]any{"status": "disconnected", "last_error": resp.Detail}).Error
	}
	return resp, nil
}

func (s *NativeStore) ListMCPTools(context.Context, *pb.ListMCPToolsRequest) (*pb.ListMCPToolsResponse, error) {
	return &pb.ListMCPToolsResponse{Code: governanceUnsupported, Msg: "mcp tool discovery is not configured in native runtime"}, nil
}

func (s *NativeStore) CallMCPTool(context.Context, *pb.CallMCPToolRequest) (*pb.CallMCPToolResponse, error) {
	return &pb.CallMCPToolResponse{Code: governanceUnsupported, Msg: "mcp tool execution is not configured in native runtime", ErrorMsg: "mcp client runner is not bound"}, nil
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

func (s *NativeStore) ListSkills(ctx context.Context, req *pb.ListSkillsRequest) (*pb.ListSkillsResponse, error) {
	page, pageSize := pageDefaults(req.GetPage(), req.GetPageSize())
	var total int64
	query := s.db.WithContext(ctx).Model(&aiSkillRecord{})
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []aiSkillRecord
	if err := query.Order("id ASC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]*pb.SkillInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, aiSkillToPB(row))
	}
	return &pb.ListSkillsResponse{Code: governanceOK, Msg: "success", Total: total, List: items}, nil
}

func (s *NativeStore) CreateSkill(ctx context.Context, req *pb.CreateSkillRequest) (*pb.SkillResponse, error) {
	if strings.TrimSpace(req.GetName()) == "" || strings.TrimSpace(req.GetDisplayName()) == "" {
		return &pb.SkillResponse{Code: governanceBadRequest, Msg: "name and display_name are required"}, nil
	}
	row := aiSkillRecord{Name: strings.TrimSpace(req.GetName()), DisplayName: strings.TrimSpace(req.GetDisplayName()), Description: nullStringFrom(req.GetDescription(), true), SourceType: defaultString(strings.TrimSpace(req.GetSourceType()), "local"), SourceURI: nullStringFrom(req.GetSourceUri(), true), IsEnabled: true}
	if req.GetIsEnabledSet() {
		row.IsEnabled = req.GetIsEnabled()
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return s.getSkillResponse(ctx, row.ID)
}

func (s *NativeStore) UpdateSkill(ctx context.Context, req *pb.UpdateSkillRequest) (*pb.SkillResponse, error) {
	updates := map[string]any{}
	putString(updates, "display_name", req.GetDisplayName())
	putString(updates, "description", req.GetDescription())
	putString(updates, "source_type", req.GetSourceType())
	putString(updates, "source_uri", req.GetSourceUri())
	if req.GetIsEnabledSet() {
		updates["is_enabled"] = req.GetIsEnabled()
	}
	if len(updates) > 0 {
		result := s.db.WithContext(ctx).Model(&aiSkillRecord{}).Where("id = ?", req.GetId()).Updates(updates)
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 0 {
			return &pb.SkillResponse{Code: governanceNotFound, Msg: "skill not found"}, nil
		}
	}
	return s.getSkillResponse(ctx, req.GetId())
}

func (s *NativeStore) CreateSkillVersion(ctx context.Context, req *pb.CreateSkillVersionRequest) (*pb.SkillVersionResponse, error) {
	manifest, err := parseSkillManifest(req.GetManifestJson())
	if err != nil {
		return &pb.SkillVersionResponse{Code: governanceBadRequest, Msg: err.Error()}, nil
	}
	row := aiSkillVersionRecord{SkillID: req.GetSkillId(), Version: defaultString(manifest.Version, fmt.Sprintf("v%d", time.Now().Unix())), ManifestJSON: req.GetManifestJson(), Instruction: nullStringFrom(manifest.Instruction, true), InputSchemaJSON: nullStringFrom(manifest.InputSchemaJSON, true), OutputSchemaJSON: nullStringFrom(manifest.OutputSchemaJSON, true), RuntimeType: defaultString(manifest.RuntimeType, model.SkillRuntimePrompt)}
	var tools []*pb.SkillToolInfo
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		for _, tool := range manifest.Tools {
			if strings.TrimSpace(tool.Name) == "" {
				continue
			}
			toolRow := aiSkillToolRecord{SkillVersionID: row.ID, ToolName: strings.TrimSpace(tool.Name), Description: nullStringFrom(tool.Description, true), InputSchemaJSON: nullStringFrom(tool.InputSchemaJSON, true), RuntimeConfigJSON: nullStringFrom(tool.RuntimeConfigJSON, true), IsEnabled: true}
			if err := tx.Create(&toolRow).Error; err != nil {
				return err
			}
			tools = append(tools, skillToolToPB(toolRow, ""))
		}
		if req.GetActivate() {
			return tx.Model(&aiSkillRecord{}).Where("id = ?", req.GetSkillId()).Update("current_version_id", row.ID).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pb.SkillVersionResponse{Code: governanceOK, Msg: "success", Version: skillVersionToPB(row), Tools: tools}, nil
}

func (s *NativeStore) ListSkillVersions(ctx context.Context, req *pb.ListSkillVersionsRequest) (*pb.ListSkillVersionsResponse, error) {
	var rows []aiSkillVersionRecord
	if err := s.db.WithContext(ctx).Where("skill_id = ?", req.GetSkillId()).Order("created_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]*pb.SkillVersionInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, skillVersionToPB(row))
	}
	return &pb.ListSkillVersionsResponse{Code: governanceOK, Msg: "success", List: items}, nil
}

func (s *NativeStore) ActivateSkillVersion(ctx context.Context, req *pb.ActivateSkillVersionRequest) (*pb.SkillResponse, error) {
	result := s.db.WithContext(ctx).Model(&aiSkillRecord{}).Where("id = ?", req.GetSkillId()).Update("current_version_id", req.GetVersionId())
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return &pb.SkillResponse{Code: governanceNotFound, Msg: "skill not found"}, nil
	}
	return s.getSkillResponse(ctx, req.GetSkillId())
}

func (s *NativeStore) ListSkillTools(ctx context.Context, req *pb.ListSkillToolsRequest) (*pb.ListSkillToolsResponse, error) {
	query := s.db.WithContext(ctx).Model(&aiSkillToolRecord{})
	if req.GetSkillVersionId() > 0 {
		query = query.Where("skill_version_id = ?", req.GetSkillVersionId())
	} else if req.GetSkillId() > 0 {
		query = query.Joins("JOIN ai_skill_versions v ON v.id = ai_skill_tools.skill_version_id").Where("v.skill_id = ?", req.GetSkillId())
	}
	if req.GetEnabledOnly() {
		query = query.Where("ai_skill_tools.is_enabled = ?", true)
	}
	var rows []aiSkillToolRecord
	if err := query.Order("ai_skill_tools.id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]*pb.SkillToolInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, skillToolToPB(row, ""))
	}
	return &pb.ListSkillToolsResponse{Code: governanceOK, Msg: "success", List: items}, nil
}

func (s *NativeStore) UpdateSkillTool(ctx context.Context, req *pb.UpdateSkillToolRequest) (*pb.SkillToolResponse, error) {
	updates := map[string]any{}
	putString(updates, "description", req.GetDescription())
	if strings.TrimSpace(req.GetRuntimeConfigJson()) != "" {
		updates["runtime_config_json"] = nullStringFrom(req.GetRuntimeConfigJson(), true)
	}
	if req.GetIsEnabledSet() {
		updates["is_enabled"] = req.GetIsEnabled()
	}
	if len(updates) > 0 {
		result := s.db.WithContext(ctx).Model(&aiSkillToolRecord{}).Where("id = ?", req.GetToolId()).Updates(updates)
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 0 {
			return &pb.SkillToolResponse{Code: governanceNotFound, Msg: "skill tool not found"}, nil
		}
	}
	return s.getSkillToolResponse(ctx, req.GetToolId())
}

func (s *NativeStore) GetAgentSkill(ctx context.Context, req *pb.GetAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	return s.getAgentSkillResponse(ctx, req.GetId())
}

func (s *NativeStore) CreateAgentSkill(ctx context.Context, req *pb.CreateAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	if strings.TrimSpace(req.GetName()) == "" || strings.TrimSpace(req.GetDisplayName()) == "" {
		return &pb.AgentSkillResponse{Code: governanceBadRequest, Msg: "name and display_name are required"}, nil
	}
	row := agentSkillRecord{Name: strings.TrimSpace(req.GetName()), DisplayName: strings.TrimSpace(req.GetDisplayName()), Description: nullStringFrom(req.GetDescription(), true), IsEnabled: true, IsManualInvocable: true, TriggerKeywords: jsonListNull(req.GetTriggerKeywords()), AgentType: defaultString(strings.TrimSpace(req.GetAgentType()), "hr_recruiting_agent"), Category: defaultString(strings.TrimSpace(req.GetCategory()), "general"), Scenario: strings.TrimSpace(req.GetScenario()), Priority: int(req.GetPriority()), RiskLevel: defaultString(strings.TrimSpace(req.GetRiskLevel()), "medium"), RequiredCapabilities: jsonListNull(req.GetRequiredCapabilities()), OutputSchema: nullStringFrom(req.GetOutputSchema(), true), EvaluationCriteria: jsonListNull(req.GetEvaluationCriteria()), SemanticTags: jsonListNull(req.GetSemanticTags())}
	if req.GetIsEnabledSet() {
		row.IsEnabled = req.GetIsEnabled()
	}
	if req.GetIsManualInvocableSet() {
		row.IsManualInvocable = req.GetIsManualInvocable()
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if strings.TrimSpace(req.GetVersion()) != "" || strings.TrimSpace(req.GetSkillMd()) != "" {
			version := agentSkillVersionRecord{SkillID: row.ID, Version: defaultString(strings.TrimSpace(req.GetVersion()), "v1"), FlowJSON: nullStringFrom(req.GetFlowJson(), true), SkillMD: defaultString(req.GetSkillMd(), renderAgentSkillMarkdown(req.GetName(), req.GetDescription(), req.GetFlowJson())), FrontmatterJSON: nullStringFrom("{}", true), BodyMarkdown: nullStringFrom(req.GetDescription(), true), ChangeNote: nullStringFrom(req.GetChangeNote(), true), CreatedBy: nullInt64From(req.GetActorUserId())}
			if err := tx.Create(&version).Error; err != nil {
				return err
			}
			if req.GetActivate() {
				return tx.Model(&agentSkillRecord{}).Where("id = ?", row.ID).Update("current_version_id", version.ID).Error
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.getAgentSkillResponse(ctx, row.ID)
}

func (s *NativeStore) UpdateAgentSkill(ctx context.Context, req *pb.UpdateAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	updates := map[string]any{"updated_by": nullInt64From(req.GetActorUserId())}
	if req.GetDisplayNameSet() {
		updates["display_name"] = strings.TrimSpace(req.GetDisplayName())
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
	if req.GetTriggerKeywordsSet() {
		updates["trigger_keywords"] = jsonListNull(req.GetTriggerKeywords())
	}
	if req.GetAgentTypeSet() {
		updates["agent_type"] = defaultString(strings.TrimSpace(req.GetAgentType()), "hr_recruiting_agent")
	}
	if req.GetCategorySet() {
		updates["category"] = defaultString(strings.TrimSpace(req.GetCategory()), "general")
	}
	if req.GetScenarioSet() {
		updates["scenario"] = strings.TrimSpace(req.GetScenario())
	}
	if req.GetPrioritySet() {
		updates["priority"] = req.GetPriority()
	}
	if req.GetRiskLevelSet() {
		updates["risk_level"] = defaultString(strings.TrimSpace(req.GetRiskLevel()), "medium")
	}
	if req.GetRequiredCapabilitiesSet() {
		updates["required_capabilities"] = jsonListNull(req.GetRequiredCapabilities())
	}
	if req.GetOutputSchemaSet() {
		updates["output_schema"] = nullStringFrom(req.GetOutputSchema(), true)
	}
	if req.GetEvaluationCriteriaSet() {
		updates["evaluation_criteria"] = jsonListNull(req.GetEvaluationCriteria())
	}
	if req.GetSemanticTagsSet() {
		updates["semantic_tags"] = jsonListNull(req.GetSemanticTags())
	}
	result := s.db.WithContext(ctx).Model(&agentSkillRecord{}).Where("id = ?", req.GetId()).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return &pb.AgentSkillResponse{Code: governanceNotFound, Msg: "agent skill not found"}, nil
	}
	return s.getAgentSkillResponse(ctx, req.GetId())
}

func (s *NativeStore) CreateAgentSkillVersion(ctx context.Context, req *pb.CreateAgentSkillVersionRequest) (*pb.AgentSkillVersionResponse, error) {
	version := agentSkillVersionRecord{SkillID: req.GetSkillId(), Version: defaultString(strings.TrimSpace(req.GetVersion()), fmt.Sprintf("v%d", time.Now().Unix())), FlowJSON: nullStringFrom(req.GetFlowJson(), true), SkillMD: defaultString(req.GetSkillMd(), renderAgentSkillMarkdown(fmt.Sprintf("skill-%d", req.GetSkillId()), req.GetChangeNote(), req.GetFlowJson())), FrontmatterJSON: nullStringFrom("{}", true), BodyMarkdown: nullStringFrom(req.GetChangeNote(), true), ChangeNote: nullStringFrom(req.GetChangeNote(), true), CreatedBy: nullInt64From(req.GetActorUserId())}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
		if req.GetActivate() {
			return tx.Model(&agentSkillRecord{}).Where("id = ?", req.GetSkillId()).Update("current_version_id", version.ID).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pb.AgentSkillVersionResponse{Code: governanceOK, Msg: "success", Version: agentSkillVersionToPB(version)}, nil
}

func (s *NativeStore) ListAgentSkillVersions(ctx context.Context, req *pb.ListAgentSkillVersionsRequest) (*pb.ListAgentSkillVersionsResponse, error) {
	var rows []agentSkillVersionRecord
	if err := s.db.WithContext(ctx).Where("skill_id = ?", req.GetSkillId()).Order("created_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]*pb.AgentSkillVersionInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, agentSkillVersionToPB(row))
	}
	return &pb.ListAgentSkillVersionsResponse{Code: governanceOK, Msg: "success", List: items}, nil
}

func (s *NativeStore) ActivateAgentSkillVersion(ctx context.Context, req *pb.ActivateAgentSkillVersionRequest) (*pb.AgentSkillResponse, error) {
	result := s.db.WithContext(ctx).Model(&agentSkillRecord{}).Where("id = ?", req.GetSkillId()).Updates(map[string]any{"current_version_id": req.GetVersionId(), "updated_by": nullInt64From(req.GetActorUserId())})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return &pb.AgentSkillResponse{Code: governanceNotFound, Msg: "agent skill not found"}, nil
	}
	return s.getAgentSkillResponse(ctx, req.GetSkillId())
}

func (s *NativeStore) UpdateAgentSkillStatus(ctx context.Context, req *pb.UpdateAgentSkillStatusRequest) (*pb.AgentSkillResponse, error) {
	result := s.db.WithContext(ctx).Model(&agentSkillRecord{}).Where("id = ?", req.GetId()).Updates(map[string]any{"is_enabled": req.GetIsEnabled(), "updated_by": nullInt64From(req.GetActorUserId())})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return &pb.AgentSkillResponse{Code: governanceNotFound, Msg: "agent skill not found"}, nil
	}
	return s.getAgentSkillResponse(ctx, req.GetId())
}

func (s *NativeStore) PreviewAgentSkill(_ context.Context, req *pb.PreviewAgentSkillRequest) (*pb.PreviewAgentSkillResponse, error) {
	body := renderAgentSkillMarkdown(req.GetName(), req.GetDescription(), req.GetFlowJson())
	frontmatter := fmt.Sprintf(`{"name":%q,"description":%q}`, strings.TrimSpace(req.GetName()), strings.TrimSpace(req.GetDescription()))
	return &pb.PreviewAgentSkillResponse{Code: governanceOK, Msg: "success", SkillMd: body, FrontmatterJson: frontmatter, BodyMarkdown: body}, nil
}

func (s *NativeStore) DebugSemanticRetrieval(context.Context, *pb.DebugSemanticRetrievalRequest) (*pb.DebugSemanticRetrievalResponse, error) {
	return &pb.DebugSemanticRetrievalResponse{Code: governanceUnsupported, Msg: "semantic retrieval debug is not configured in native runtime", EmbeddingAvailable: false, FallbackReason: "embedding query runner is not bound"}, nil
}

func (s *NativeStore) getMCPServerResponse(ctx context.Context, id int64) (*pb.MCPServerResponse, error) {
	var row mcpServerRecord
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.MCPServerResponse{Code: governanceNotFound, Msg: "mcp server not found"}, nil
		}
		return nil, err
	}
	return &pb.MCPServerResponse{Code: governanceOK, Msg: "success", Server: mcpServerToPB(row)}, nil
}

func (s *NativeStore) getMCPPolicyResponse(ctx context.Context, id int64) (*pb.MCPToolPolicyResponse, error) {
	var row mcpToolPolicyRecord
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.MCPToolPolicyResponse{Code: governanceNotFound, Msg: "mcp tool policy not found"}, nil
		}
		return nil, err
	}
	return &pb.MCPToolPolicyResponse{Code: governanceOK, Msg: "success", Policy: mcpPolicyToPB(row)}, nil
}

func (s *NativeStore) getSkillResponse(ctx context.Context, id int64) (*pb.SkillResponse, error) {
	var row aiSkillRecord
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.SkillResponse{Code: governanceNotFound, Msg: "skill not found"}, nil
		}
		return nil, err
	}
	return &pb.SkillResponse{Code: governanceOK, Msg: "success", Skill: aiSkillToPB(row)}, nil
}

func (s *NativeStore) getSkillToolResponse(ctx context.Context, id int64) (*pb.SkillToolResponse, error) {
	var row aiSkillToolRecord
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.SkillToolResponse{Code: governanceNotFound, Msg: "skill tool not found"}, nil
		}
		return nil, err
	}
	return &pb.SkillToolResponse{Code: governanceOK, Msg: "success", Tool: skillToolToPB(row, "")}, nil
}

func (s *NativeStore) getAgentSkillResponse(ctx context.Context, id int64) (*pb.AgentSkillResponse, error) {
	var row agentSkillRecord
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.AgentSkillResponse{Code: governanceNotFound, Msg: "agent skill not found"}, nil
		}
		return nil, err
	}
	return &pb.AgentSkillResponse{Code: governanceOK, Msg: "success", Skill: agentSkillToPB(row)}, nil
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

func aiSkillToPB(row aiSkillRecord) *pb.SkillInfo {
	return &pb.SkillInfo{Id: row.ID, Name: row.Name, DisplayName: row.DisplayName, Description: nullString(row.Description), SourceType: row.SourceType, SourceUri: nullString(row.SourceURI), CurrentVersionId: nullInt64(row.CurrentVersionID), IsEnabled: row.IsEnabled, CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt)}
}

func skillVersionToPB(row aiSkillVersionRecord) *pb.SkillVersionInfo {
	return &pb.SkillVersionInfo{Id: row.ID, SkillId: row.SkillID, Version: row.Version, ManifestJson: row.ManifestJSON, Instruction: nullString(row.Instruction), InputSchemaJson: nullString(row.InputSchemaJSON), OutputSchemaJson: nullString(row.OutputSchemaJSON), RuntimeType: row.RuntimeType, CreatedAt: formatTime(row.CreatedAt)}
}

func skillToolToPB(row aiSkillToolRecord, skillName string) *pb.SkillToolInfo {
	key := row.ToolName
	runtimeName := row.ToolName
	if strings.TrimSpace(skillName) != "" {
		key = skillName + ":" + row.ToolName
		runtimeName = "skill_" + skillName + "_" + row.ToolName
	}
	return &pb.SkillToolInfo{Id: row.ID, SkillVersionId: row.SkillVersionID, ToolName: row.ToolName, Description: nullString(row.Description), InputSchemaJson: nullString(row.InputSchemaJSON), RuntimeConfigJson: redactSensitiveJSON(nullString(row.RuntimeConfigJSON)), IsEnabled: row.IsEnabled, CapabilityKey: key, RuntimeToolName: runtimeName, CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt)}
}

func agentSkillToPB(row agentSkillRecord) *pb.AgentSkillInfo {
	return &pb.AgentSkillInfo{Id: row.ID, Name: row.Name, DisplayName: row.DisplayName, Description: nullString(row.Description), CurrentVersionId: nullInt64(row.CurrentVersionID), IsEnabled: row.IsEnabled, IsManualInvocable: row.IsManualInvocable, TriggerKeywords: jsonStringList(row.TriggerKeywords), CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt), AgentType: row.AgentType, Category: row.Category, Scenario: row.Scenario, Priority: int32(row.Priority), RiskLevel: row.RiskLevel, RequiredCapabilities: jsonStringList(row.RequiredCapabilities), OutputSchema: nullString(row.OutputSchema), EvaluationCriteria: jsonStringList(row.EvaluationCriteria), SemanticTags: jsonStringList(row.SemanticTags)}
}

func agentSkillVersionToPB(row agentSkillVersionRecord) *pb.AgentSkillVersionInfo {
	return &pb.AgentSkillVersionInfo{Id: row.ID, SkillId: row.SkillID, Version: row.Version, FlowJson: nullString(row.FlowJSON), SkillMd: row.SkillMD, FrontmatterJson: nullString(row.FrontmatterJSON), BodyMarkdown: nullString(row.BodyMarkdown), ChangeNote: nullString(row.ChangeNote), CreatedAt: formatTime(row.CreatedAt)}
}

type parsedSkillManifest struct {
	Version          string            `json:"version"`
	Instruction      string            `json:"instruction"`
	InputSchemaJSON  string            `json:"input_schema_json"`
	OutputSchemaJSON string            `json:"output_schema_json"`
	RuntimeType      string            `json:"runtime_type"`
	Tools            []parsedSkillTool `json:"tools"`
}

type parsedSkillTool struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	InputSchemaJSON   string `json:"input_schema_json"`
	RuntimeConfigJSON string `json:"runtime_config_json"`
}

func parseSkillManifest(value string) (parsedSkillManifest, error) {
	if strings.TrimSpace(value) == "" {
		return parsedSkillManifest{}, fmt.Errorf("manifest_json is required")
	}
	var manifest parsedSkillManifest
	if err := json.Unmarshal([]byte(value), &manifest); err != nil {
		return parsedSkillManifest{}, fmt.Errorf("manifest_json is invalid: %w", err)
	}
	return manifest, nil
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

func renderAgentSkillMarkdown(name, description, flowJSON string) string {
	return fmt.Sprintf("# %s\n\n%s\n\n```json\n%s\n```\n", strings.TrimSpace(name), strings.TrimSpace(description), strings.TrimSpace(flowJSON))
}
