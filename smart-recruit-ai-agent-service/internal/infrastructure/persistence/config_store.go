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
	"gorm.io/gorm/clause"

	embeddinginfra "smart-recruit-ai-agent-service/internal/infrastructure/provider"
	"smart-recruit-proto/recruitment/pb"
)

const (
	configOK          int32 = 0
	configBadRequest  int32 = 400
	configNotFound    int32 = 404
	configUnavailable int32 = 500
	configUnsupported int32 = 501
)

type llmModelRecord struct {
	ID                  int64     `gorm:"primaryKey"`
	ProviderID          int64     `gorm:"column:provider_id"`
	ModelName           string    `gorm:"column:model_name"`
	DisplayName         string    `gorm:"column:display_name"`
	Temperature         float64   `gorm:"column:temperature"`
	TopP                float64   `gorm:"column:top_p"`
	MaxTokens           int       `gorm:"column:max_tokens"`
	ContextWindowTokens int       `gorm:"column:context_window_tokens"`
	MaxConcurrency      int       `gorm:"column:max_concurrency"`
	TimeoutSeconds      int       `gorm:"column:timeout_seconds"`
	IsEnabled           bool      `gorm:"column:is_enabled"`
	IsDefault           bool      `gorm:"column:is_default"`
	CreatedAt           time.Time `gorm:"column:created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at"`
}

func (llmModelRecord) TableName() string { return "llm_models" }

type promptVersionRecord struct {
	ID         int64          `gorm:"primaryKey"`
	TemplateID int64          `gorm:"column:template_id"`
	Version    int            `gorm:"column:version"`
	Content    string         `gorm:"column:content"`
	ChangedBy  sql.NullInt64  `gorm:"column:changed_by"`
	ChangeNote sql.NullString `gorm:"column:change_note"`
	CreatedAt  time.Time      `gorm:"column:created_at"`
}

func (promptVersionRecord) TableName() string { return "prompt_versions" }

type agentConfigRecord struct {
	ID                  int64           `gorm:"primaryKey"`
	Name                string          `gorm:"column:name"`
	DisplayName         string          `gorm:"column:display_name"`
	Description         sql.NullString  `gorm:"column:description"`
	AgentType           string          `gorm:"column:agent_type"`
	PromptTemplateID    sql.NullInt64   `gorm:"column:prompt_template_id"`
	Instruction         sql.NullString  `gorm:"column:instruction"`
	MaxIterations       int             `gorm:"column:max_iterations"`
	TemperatureOverride sql.NullFloat64 `gorm:"column:temperature_override"`
	IsDefault           bool            `gorm:"column:is_default"`
	IsEnabled           bool            `gorm:"column:is_enabled"`
	CreatedAt           time.Time       `gorm:"column:created_at"`
	UpdatedAt           time.Time       `gorm:"column:updated_at"`
}

func (agentConfigRecord) TableName() string { return "agent_configs" }

type agentToolBindingRecord struct {
	ID        int64     `gorm:"primaryKey"`
	AgentID   int64     `gorm:"column:agent_id"`
	ToolName  string    `gorm:"column:tool_name"`
	IsEnabled bool      `gorm:"column:is_enabled"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (agentToolBindingRecord) TableName() string { return "agent_tool_bindings" }

type agentCapabilityBindingRecord struct {
	ID               int64          `gorm:"primaryKey"`
	AgentID          int64          `gorm:"column:agent_id"`
	CapabilitySource string         `gorm:"column:capability_source"`
	CapabilityKey    string         `gorm:"column:capability_key"`
	IsEnabled        bool           `gorm:"column:is_enabled"`
	Priority         int            `gorm:"column:priority"`
	PolicyJSON       sql.NullString `gorm:"column:policy_json"`
	CreatedAt        time.Time      `gorm:"column:created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at"`
}

func (agentCapabilityBindingRecord) TableName() string { return "agent_capability_bindings" }

type embeddingModelRecord struct {
	ID              int64          `gorm:"primaryKey"`
	ProviderID      int64          `gorm:"column:provider_id"`
	ModelName       string         `gorm:"column:model_name"`
	DisplayName     string         `gorm:"column:display_name"`
	EmbeddingDim    int            `gorm:"column:embedding_dim"`
	InputTokenLimit int            `gorm:"column:input_token_limit"`
	BatchSize       int            `gorm:"column:batch_size"`
	TimeoutSeconds  int            `gorm:"column:timeout_seconds"`
	MaxRetries      int            `gorm:"column:max_retries"`
	IsEnabled       bool           `gorm:"column:is_enabled"`
	IsDefault       bool           `gorm:"column:is_default"`
	LastTestStatus  string         `gorm:"column:last_test_status"`
	LastTestError   sql.NullString `gorm:"column:last_test_error"`
	LastTestAt      *time.Time     `gorm:"column:last_test_at"`
	CreatedAt       time.Time      `gorm:"column:created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at"`
}

func (embeddingModelRecord) TableName() string { return "embedding_models" }

type aiEmbeddingRecord struct {
	ID             int64          `gorm:"primaryKey"`
	ObjectType     string         `gorm:"column:object_type"`
	ObjectID       int64          `gorm:"column:object_id"`
	ScopeType      string         `gorm:"column:scope_type"`
	ScopeID        int64          `gorm:"column:scope_id"`
	TextHash       string         `gorm:"column:text_hash"`
	EmbeddingModel string         `gorm:"column:embedding_model"`
	EmbeddingDim   int            `gorm:"column:embedding_dim"`
	VectorJSON     sql.NullString `gorm:"column:vector_json"`
	MetadataJSON   sql.NullString `gorm:"column:metadata_json"`
	Status         string         `gorm:"column:status"`
	LastError      sql.NullString `gorm:"column:last_error"`
	CreatedAt      time.Time      `gorm:"column:created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at"`
}

func (aiEmbeddingRecord) TableName() string { return "ai_embeddings" }

func (s *NativeStore) CreateLlmProvider(ctx context.Context, req *pb.CreateProviderRequest) (*pb.ProviderResponse, error) {
	providerType := strings.TrimSpace(req.GetProviderType())
	if providerType == "" {
		providerType = "openai_compatible"
	}
	if strings.TrimSpace(req.GetName()) == "" || strings.TrimSpace(req.GetBaseUrl()) == "" {
		return &pb.ProviderResponse{Code: configBadRequest, Msg: "provider name and base_url are required"}, nil
	}
	encrypted, err := s.encryptAPIKey(req.GetApiKey())
	if err != nil {
		return &pb.ProviderResponse{Code: configBadRequest, Msg: err.Error()}, nil
	}
	row := llmProviderRecord{Name: strings.TrimSpace(req.GetName()), BaseURL: strings.TrimSpace(req.GetBaseUrl()), APIKeyEncrypted: encrypted, ProviderType: providerType, ExtraHeaders: nullableJSONText(req.GetExtraHeadersJson()), IsEnabled: true}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &pb.ProviderResponse{Code: configOK, Msg: "success", Provider: s.llmProviderToPB(row)}, nil
}

func (s *NativeStore) UpdateLlmProvider(ctx context.Context, req *pb.UpdateProviderRequest) (*pb.ProviderResponse, error) {
	updates := map[string]any{}
	putString(updates, "name", req.GetName())
	putString(updates, "base_url", req.GetBaseUrl())
	if strings.TrimSpace(req.GetApiKey()) != "" {
		encrypted, err := s.encryptAPIKey(req.GetApiKey())
		if err != nil {
			return &pb.ProviderResponse{Code: configBadRequest, Msg: err.Error()}, nil
		}
		updates["api_key_encrypted"] = encrypted
	}
	putString(updates, "provider_type", req.GetProviderType())
	if req.GetExtraHeadersSet() {
		updates["extra_headers"] = nullableJSONText(req.GetExtraHeadersJson())
	}
	if req.GetIsEnabledSet() {
		updates["is_enabled"] = req.GetIsEnabled()
	}
	if len(updates) == 0 {
		return s.getLlmProviderResponse(ctx, req.GetId())
	}
	result := s.db.WithContext(ctx).Model(&llmProviderRecord{}).Where("id = ?", req.GetId()).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	return s.getLlmProviderResponse(ctx, req.GetId())
}

func (s *NativeStore) DeleteLlmProvider(ctx context.Context, req *pb.DeleteProviderRequest) (*pb.CommonResponse, error) {
	return rowsCommon(s.db.WithContext(ctx).Delete(&llmProviderRecord{}, req.GetId()), "success", "provider not found")
}

func (s *NativeStore) TestLlmProviderConnection(ctx context.Context, req *pb.TestProviderConnectionRequest) (*pb.TestProviderConnectionResponse, error) {
	var row llmProviderRecord
	err := s.db.WithContext(ctx).First(&row, req.GetProviderId()).Error
	if err == gorm.ErrRecordNotFound {
		return &pb.TestProviderConnectionResponse{Code: configNotFound, Msg: "provider not found", Success: false}, nil
	}
	if err != nil {
		return nil, err
	}
	if !row.IsEnabled {
		return &pb.TestProviderConnectionResponse{Code: configUnavailable, Msg: "provider disabled", Success: false, Detail: "provider is disabled"}, nil
	}
	if strings.TrimSpace(row.BaseURL) == "" || strings.TrimSpace(row.APIKeyEncrypted) == "" {
		return &pb.TestProviderConnectionResponse{Code: configUnavailable, Msg: "provider configuration incomplete", Success: false, Detail: "base_url and api_key are required"}, nil
	}
	return s.validateLlmRuntimeConnection(ctx, 0, row.ID)
}

func (s *NativeStore) TestLlmModelConnection(ctx context.Context, req *pb.TestModelConnectionRequest) (*pb.TestProviderConnectionResponse, error) {
	if req.GetModelId() <= 0 {
		return &pb.TestProviderConnectionResponse{Code: configBadRequest, Msg: "model_id is required", Success: false, Detail: "model_id is required"}, nil
	}
	var model llmModelRecord
	err := s.db.WithContext(ctx).First(&model, req.GetModelId()).Error
	if err == gorm.ErrRecordNotFound {
		return &pb.TestProviderConnectionResponse{Code: configNotFound, Msg: "model not found", Success: false, Detail: "model not found"}, nil
	}
	if err != nil {
		return nil, err
	}
	if !model.IsEnabled {
		return &pb.TestProviderConnectionResponse{Code: configUnavailable, Msg: "model disabled", Success: false, Detail: "model is disabled"}, nil
	}
	var provider llmProviderRecord
	if err := s.db.WithContext(ctx).First(&provider, model.ProviderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.TestProviderConnectionResponse{Code: configNotFound, Msg: "provider not found", Success: false, Detail: "provider not found"}, nil
		}
		return nil, err
	}
	if !provider.IsEnabled {
		return &pb.TestProviderConnectionResponse{Code: configUnavailable, Msg: "provider disabled", Success: false, Detail: "provider is disabled"}, nil
	}
	if strings.TrimSpace(provider.BaseURL) == "" || strings.TrimSpace(provider.APIKeyEncrypted) == "" {
		return &pb.TestProviderConnectionResponse{Code: configUnavailable, Msg: "provider configuration incomplete", Success: false, Detail: "base_url and api_key are required"}, nil
	}
	return s.validateLlmRuntimeConnection(ctx, model.ID, 0)
}

func (s *NativeStore) CreateLlmModel(ctx context.Context, req *pb.CreateModelRequest) (*pb.ModelResponse, error) {
	if req.GetProviderId() <= 0 || strings.TrimSpace(req.GetModelName()) == "" {
		return &pb.ModelResponse{Code: configBadRequest, Msg: "provider_id and model_name are required"}, nil
	}
	row := llmModelRecord{ProviderID: req.GetProviderId(), ModelName: strings.TrimSpace(req.GetModelName()), DisplayName: strings.TrimSpace(req.GetDisplayName()), Temperature: defaultFloat(req.GetTemperature(), 0.7), TopP: defaultFloat(req.GetTopP(), 1), MaxTokens: defaultInt32(req.GetMaxTokens(), 4096), ContextWindowTokens: int(req.GetContextWindowTokens()), MaxConcurrency: defaultInt32(req.GetMaxConcurrency(), 10), TimeoutSeconds: defaultInt32(req.GetTimeoutSeconds(), 90), IsEnabled: true, IsDefault: req.GetIsDefault()}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if row.IsDefault {
			if err := tx.Model(&llmModelRecord{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(&row).Error
	})
	if err != nil {
		return nil, err
	}
	return s.getLlmModelResponse(ctx, row.ID)
}

func (s *NativeStore) UpdateLlmModel(ctx context.Context, req *pb.UpdateModelRequest) (*pb.ModelResponse, error) {
	updates := map[string]any{}
	putString(updates, "model_name", req.GetModelName())
	putString(updates, "display_name", req.GetDisplayName())
	if req.GetTemperatureSet() {
		updates["temperature"] = req.GetTemperature()
	}
	if req.GetTopPSet() {
		updates["top_p"] = req.GetTopP()
	}
	if req.GetMaxTokensSet() {
		updates["max_tokens"] = req.GetMaxTokens()
	}
	if req.GetContextWindowTokensSet() {
		updates["context_window_tokens"] = req.GetContextWindowTokens()
	}
	if req.GetMaxConcurrencySet() {
		updates["max_concurrency"] = req.GetMaxConcurrency()
	}
	if req.GetTimeoutSecondsSet() {
		updates["timeout_seconds"] = req.GetTimeoutSeconds()
	}
	if req.GetIsEnabledSet() {
		updates["is_enabled"] = req.GetIsEnabled()
	}
	if req.GetIsDefaultSet() {
		updates["is_default"] = req.GetIsDefault()
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if req.GetIsDefaultSet() && req.GetIsDefault() {
			if err := tx.Model(&llmModelRecord{}).Where("id <> ? AND is_default = ?", req.GetId(), true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		if len(updates) == 0 {
			return nil
		}
		result := tx.Model(&llmModelRecord{}).Where("id = ?", req.GetId()).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if err == gorm.ErrRecordNotFound {
		return &pb.ModelResponse{Code: configNotFound, Msg: "model not found"}, nil
	}
	if err != nil {
		return nil, err
	}
	return s.getLlmModelResponse(ctx, req.GetId())
}

func (s *NativeStore) DeleteLlmModel(ctx context.Context, req *pb.DeleteModelRequest) (*pb.CommonResponse, error) {
	return rowsCommon(s.db.WithContext(ctx).Delete(&llmModelRecord{}, req.GetId()), "success", "model not found")
}

func (s *NativeStore) CreatePromptTemplate(ctx context.Context, req *pb.CreatePromptTemplateRequest) (*pb.PromptTemplateResponse, error) {
	if strings.TrimSpace(req.GetName()) == "" || strings.TrimSpace(req.GetContent()) == "" || strings.TrimSpace(req.GetAgentType()) == "" {
		return &pb.PromptTemplateResponse{Code: configBadRequest, Msg: "name, content, and agent_type are required"}, nil
	}
	role := defaultString(strings.TrimSpace(req.GetPromptRole()), "system")
	row := promptTemplateRecord{Name: strings.TrimSpace(req.GetName()), Content: req.GetContent(), Variables: nullStringFrom(req.GetVariablesJson(), true), Version: 1, IsActive: true, AgentType: strings.TrimSpace(req.GetAgentType()), PromptRole: role, CreatedBy: nullInt64From(req.GetCreatedBy()), UpdatedBy: nullInt64From(req.GetCreatedBy())}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.Create(&promptVersionRecord{TemplateID: row.ID, Version: row.Version, Content: row.Content, ChangedBy: row.CreatedBy, ChangeNote: sql.NullString{String: "initial version", Valid: true}}).Error
	})
	if err != nil {
		return nil, err
	}
	return &pb.PromptTemplateResponse{Code: configOK, Msg: "success", Template: promptTemplateToPB(row)}, nil
}

func (s *NativeStore) UpdatePromptTemplate(ctx context.Context, req *pb.UpdatePromptTemplateRequest) (*pb.PromptTemplateResponse, error) {
	var row promptTemplateRecord
	if err := s.db.WithContext(ctx).First(&row, req.GetId()).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.PromptTemplateResponse{Code: configNotFound, Msg: "prompt template not found"}, nil
		}
		return nil, err
	}
	contentChanged := strings.TrimSpace(req.GetContent()) != "" && req.GetContent() != row.Content
	updates := map[string]any{"updated_by": nullInt64From(req.GetUpdatedBy())}
	putString(updates, "name", req.GetName())
	if strings.TrimSpace(req.GetContent()) != "" {
		updates["content"] = req.GetContent()
	}
	if strings.TrimSpace(req.GetVariablesJson()) != "" {
		updates["variables"] = nullStringFrom(req.GetVariablesJson(), true)
	}
	if req.GetIsActiveSet() {
		// Refuse to disable the last active prompt in a type/role scope so runtime always has a fallback.
		if row.IsActive && !req.GetIsActive() {
			hasOther, checkErr := s.hasOtherActivePromptTemplate(ctx, row.ID, row.AgentType, row.PromptRole)
			if checkErr != nil {
				return nil, checkErr
			}
			if !hasOther {
				return &pb.PromptTemplateResponse{
					Code: configBadRequest,
					Msg:  "当前绑定类型下没有其他启用中的提示词，不能禁用最后一条",
				}, nil
			}
		}
		updates["is_active"] = req.GetIsActive()
	}
	if contentChanged {
		updates["version"] = row.Version + 1
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&promptTemplateRecord{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
			return err
		}
		if contentChanged {
			return tx.Create(&promptVersionRecord{TemplateID: row.ID, Version: row.Version + 1, Content: req.GetContent(), ChangedBy: nullInt64From(req.GetUpdatedBy()), ChangeNote: nullStringFrom(req.GetChangeNote(), true)}).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.getPromptTemplateResponse(ctx, req.GetId())
}

// hasOtherActivePromptTemplate reports whether another enabled template can serve the same
// agent_type + prompt_role scope (hr_agent / hr_recruiting_agent share one conversation scope).
func (s *NativeStore) hasOtherActivePromptTemplate(ctx context.Context, excludeID int64, agentType, promptRole string) (bool, error) {
	query := s.db.WithContext(ctx).Model(&promptTemplateRecord{}).
		Where("id <> ? AND is_active = ?", excludeID, true)
	types := promptAgentTypeScope(agentType)
	if len(types) == 1 {
		query = query.Where("agent_type = ?", types[0])
	} else {
		query = query.Where("agent_type IN ?", types)
	}
	if role := strings.TrimSpace(promptRole); role != "" {
		query = query.Where("prompt_role = ?", role)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func promptAgentTypeScope(agentType string) []string {
	agentType = strings.TrimSpace(agentType)
	if strings.EqualFold(agentType, "hr_recruiting_agent") || strings.EqualFold(agentType, "hr_agent") {
		return []string{"hr_recruiting_agent", "hr_agent"}
	}
	if agentType == "" {
		return []string{""}
	}
	return []string{agentType}
}

func (s *NativeStore) DeletePromptTemplate(ctx context.Context, req *pb.DeletePromptTemplateRequest) (*pb.CommonResponse, error) {
	return rowsCommon(s.db.WithContext(ctx).Delete(&promptTemplateRecord{}, req.GetId()), "success", "prompt template not found")
}

func (s *NativeStore) GetPromptVersionHistory(ctx context.Context, req *pb.GetPromptVersionHistoryRequest) (*pb.GetPromptVersionHistoryResponse, error) {
	var total int64
	query := s.db.WithContext(ctx).Model(&promptVersionRecord{}).Where("template_id = ?", req.GetTemplateId())
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []promptVersionRecord
	page, pageSize := pageDefaults(req.GetPage(), req.GetPageSize())
	if err := query.Order("version DESC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]*pb.PromptVersionInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, &pb.PromptVersionInfo{Id: row.ID, TemplateId: row.TemplateID, Version: int32(row.Version), Content: row.Content, ChangedBy: nullInt64(row.ChangedBy), ChangeNote: nullString(row.ChangeNote), CreatedAt: formatTime(row.CreatedAt)})
	}
	return &pb.GetPromptVersionHistoryResponse{Code: configOK, Msg: "success", Total: total, List: items}, nil
}

func (s *NativeStore) RollbackPromptVersion(ctx context.Context, req *pb.RollbackPromptVersionRequest) (*pb.PromptTemplateResponse, error) {
	var version promptVersionRecord
	err := s.db.WithContext(ctx).Where("template_id = ? AND version = ?", req.GetTemplateId(), req.GetVersion()).First(&version).Error
	if err == gorm.ErrRecordNotFound {
		return &pb.PromptTemplateResponse{Code: configNotFound, Msg: "prompt version not found"}, nil
	}
	if err != nil {
		return nil, err
	}
	updateReq := &pb.UpdatePromptTemplateRequest{Id: req.GetTemplateId(), Content: version.Content, UpdatedBy: req.GetUpdatedBy(), ChangeNote: defaultString(req.GetChangeNote(), fmt.Sprintf("rollback to version %d", req.GetVersion()))}
	return s.UpdatePromptTemplate(ctx, updateReq)
}

func (s *NativeStore) RenderPrompt(ctx context.Context, req *pb.RenderPromptRequest) (*pb.RenderPromptResponse, error) {
	var row promptTemplateRecord
	if err := s.db.WithContext(ctx).First(&row, req.GetTemplateId()).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.RenderPromptResponse{Code: configNotFound, Msg: "prompt template not found"}, nil
		}
		return nil, err
	}
	rendered := row.Content
	for key, value := range req.GetVariables() {
		rendered = strings.ReplaceAll(rendered, "{{"+key+"}}", value)
	}
	return &pb.RenderPromptResponse{Code: configOK, Msg: "success", RenderedContent: rendered}, nil
}

func (s *NativeStore) GetActivePromptByAgentType(ctx context.Context, req *pb.GetActivePromptByAgentTypeRequest) (*pb.GetActivePromptByAgentTypeResponse, error) {
	var row promptTemplateRecord
	query := s.db.WithContext(ctx).Where("agent_type = ? AND is_active = ?", req.GetAgentType(), true)
	if strings.TrimSpace(req.GetPromptRole()) != "" {
		query = query.Where("prompt_role = ?", strings.TrimSpace(req.GetPromptRole()))
	}
	if err := query.Order("updated_at DESC, id DESC").First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.GetActivePromptByAgentTypeResponse{Code: configNotFound, Msg: "active prompt not found"}, nil
		}
		return nil, err
	}
	return &pb.GetActivePromptByAgentTypeResponse{Code: configOK, Msg: "success", Template: promptTemplateToPB(row)}, nil
}

func (s *NativeStore) CreateAgent(ctx context.Context, req *pb.CreateAgentRequest) (*pb.AgentConfigResponse, error) {
	if strings.TrimSpace(req.GetName()) == "" || strings.TrimSpace(req.GetDisplayName()) == "" || strings.TrimSpace(req.GetAgentType()) == "" {
		return &pb.AgentConfigResponse{Code: configBadRequest, Msg: "name, display_name, and agent_type are required"}, nil
	}
	if req.GetPromptTemplateId() > 0 {
		message, err := validateAgentPromptBinding(s.db.WithContext(ctx), req.GetPromptTemplateId(), req.GetAgentType())
		if err != nil {
			return nil, err
		}
		if message != "" {
			return &pb.AgentConfigResponse{Code: configBadRequest, Msg: message}, nil
		}
	}
	row := agentConfigRecord{Name: strings.TrimSpace(req.GetName()), DisplayName: strings.TrimSpace(req.GetDisplayName()), Description: nullStringFrom(req.GetDescription(), true), AgentType: strings.TrimSpace(req.GetAgentType()), PromptTemplateID: nullInt64From(req.GetPromptTemplateId()), Instruction: nullStringFrom(req.GetInstruction(), true), MaxIterations: defaultInt32(req.GetMaxIterations(), 5), IsDefault: req.GetIsDefault(), IsEnabled: true}
	if req.GetTemperatureOverrideSet() {
		row.TemperatureOverride = sql.NullFloat64{Float64: req.GetTemperatureOverride(), Valid: true}
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if row.IsDefault {
			if err := tx.Model(&agentConfigRecord{}).Where("agent_type = ? AND is_default = ?", row.AgentType, true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return replaceAgentBindings(tx, row.ID, req.GetToolNames(), req.GetCapabilityBindings(), true, true)
	})
	if err != nil {
		return nil, err
	}
	return s.getAgentResponse(ctx, row.ID)
}

func (s *NativeStore) UpdateAgent(ctx context.Context, req *pb.UpdateAgentRequest) (*pb.AgentConfigResponse, error) {
	updates := map[string]any{}
	putString(updates, "name", req.GetName())
	putString(updates, "display_name", req.GetDisplayName())
	putString(updates, "description", req.GetDescription())
	if req.GetPromptTemplateIdSet() {
		updates["prompt_template_id"] = nullInt64From(req.GetPromptTemplateId())
	}
	putString(updates, "instruction", req.GetInstruction())
	if req.GetMaxIterationsSet() {
		updates["max_iterations"] = req.GetMaxIterations()
	}
	if req.GetTemperatureOverrideSet() {
		updates["temperature_override"] = req.GetTemperatureOverride()
	}
	if req.GetIsDefaultSet() {
		updates["is_default"] = req.GetIsDefault()
	}
	if req.GetIsEnabledSet() {
		updates["is_enabled"] = req.GetIsEnabled()
	}
	var existing agentConfigRecord
	if err := s.db.WithContext(ctx).First(&existing, req.GetId()).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.AgentConfigResponse{Code: configNotFound, Msg: "agent not found"}, nil
		}
		return nil, err
	}
	if req.GetPromptTemplateIdSet() && req.GetPromptTemplateId() > 0 {
		message, err := validateAgentPromptBinding(s.db.WithContext(ctx), req.GetPromptTemplateId(), existing.AgentType)
		if err != nil {
			return nil, err
		}
		if message != "" {
			return &pb.AgentConfigResponse{Code: configBadRequest, Msg: message}, nil
		}
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if req.GetIsDefaultSet() && req.GetIsDefault() {
			if err := tx.Model(&agentConfigRecord{}).Where("id <> ? AND agent_type = ? AND is_default = ?", req.GetId(), existing.AgentType, true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		if len(updates) > 0 {
			if err := tx.Model(&agentConfigRecord{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
				return err
			}
		}
		return replaceAgentBindings(tx, req.GetId(), req.GetToolNames(), req.GetCapabilityBindings(), req.GetToolNamesSet(), req.GetCapabilityBindingsSet())
	})
	if err != nil {
		return nil, err
	}
	return s.getAgentResponse(ctx, req.GetId())
}

func validateAgentPromptBinding(db *gorm.DB, promptID int64, agentType string) (string, error) {
	var prompt promptTemplateRecord
	if err := db.Where("id = ?", promptID).First(&prompt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "prompt template not found", nil
		}
		return "", err
	}
	if !prompt.IsActive {
		return "prompt template must be active", nil
	}
	if !strings.EqualFold(strings.TrimSpace(prompt.PromptRole), "system") {
		return "prompt template role must be system", nil
	}
	if !agentPromptTypesCompatible(agentType, prompt.AgentType) {
		return "prompt template agent type is incompatible", nil
	}
	return "", nil
}

func agentPromptTypesCompatible(agentType, promptAgentType string) bool {
	agentType = strings.TrimSpace(agentType)
	promptAgentType = strings.TrimSpace(promptAgentType)
	if strings.EqualFold(agentType, promptAgentType) {
		return true
	}
	return strings.EqualFold(agentType, "hr_recruiting_agent") && strings.EqualFold(promptAgentType, "hr_agent")
}

func (s *NativeStore) DeleteAgent(ctx context.Context, req *pb.DeleteAgentRequest) (*pb.CommonResponse, error) {
	return rowsCommon(s.db.WithContext(ctx).Delete(&agentConfigRecord{}, req.GetId()), "success", "agent not found")
}

func (s *NativeStore) GetAgentConfig(ctx context.Context, req *pb.GetAgentConfigRequest) (*pb.GetAgentConfigResponse, error) {
	var row agentConfigListRow
	query := s.agentConfigSelect(ctx).Where("a.agent_type = ? AND a.is_enabled = ?", req.GetAgentType(), true).Order("a.is_default DESC, a.id ASC")
	if err := query.First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.GetAgentConfigResponse{Code: configNotFound, Msg: "agent config not found"}, nil
		}
		return nil, err
	}
	agent, err := s.agentConfigInfo(ctx, row)
	if err != nil {
		return nil, err
	}
	return &pb.GetAgentConfigResponse{Code: configOK, Msg: "success", Agent: agent}, nil
}

func (s *NativeStore) CreateEmbeddingProvider(ctx context.Context, req *pb.CreateEmbeddingProviderRequest) (*pb.EmbeddingProviderResponse, error) {
	if strings.TrimSpace(req.GetName()) == "" || strings.TrimSpace(req.GetProviderType()) == "" || strings.TrimSpace(req.GetEndpoint()) == "" {
		return &pb.EmbeddingProviderResponse{Code: configBadRequest, Msg: "name, provider_type, and endpoint are required"}, nil
	}
	encrypted, err := s.encryptAPIKey(req.GetApiKey())
	if err != nil {
		return &pb.EmbeddingProviderResponse{Code: configBadRequest, Msg: err.Error()}, nil
	}
	row := embeddingProviderRecord{Name: strings.TrimSpace(req.GetName()), ProviderType: strings.TrimSpace(req.GetProviderType()), Endpoint: strings.TrimSpace(req.GetEndpoint()), APIKeyEncrypted: encrypted, ExtraHeaders: nullableJSONText(req.GetExtraHeadersJson()), IsEnabled: true}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &pb.EmbeddingProviderResponse{Code: configOK, Msg: "success", Provider: s.embeddingProviderToPB(row)}, nil
}

func (s *NativeStore) UpdateEmbeddingProvider(ctx context.Context, req *pb.UpdateEmbeddingProviderRequest) (*pb.EmbeddingProviderResponse, error) {
	updates := map[string]any{}
	putString(updates, "name", req.GetName())
	putString(updates, "provider_type", req.GetProviderType())
	putString(updates, "endpoint", req.GetEndpoint())
	if strings.TrimSpace(req.GetApiKey()) != "" {
		encrypted, err := s.encryptAPIKey(req.GetApiKey())
		if err != nil {
			return &pb.EmbeddingProviderResponse{Code: configBadRequest, Msg: err.Error()}, nil
		}
		updates["api_key_encrypted"] = encrypted
	}
	if req.GetExtraHeadersSet() {
		updates["extra_headers"] = nullableJSONText(req.GetExtraHeadersJson())
	}
	if req.GetIsEnabledSet() {
		updates["is_enabled"] = req.GetIsEnabled()
	}
	if len(updates) == 0 {
		return s.getEmbeddingProviderResponse(ctx, req.GetId())
	}
	result := s.db.WithContext(ctx).Model(&embeddingProviderRecord{}).Where("id = ?", req.GetId()).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return &pb.EmbeddingProviderResponse{Code: configNotFound, Msg: "embedding provider not found"}, nil
	}
	return s.getEmbeddingProviderResponse(ctx, req.GetId())
}

func (s *NativeStore) DeleteEmbeddingProvider(ctx context.Context, req *pb.DeleteEmbeddingProviderRequest) (*pb.CommonResponse, error) {
	return rowsCommon(s.db.WithContext(ctx).Delete(&embeddingProviderRecord{}, req.GetId()), "success", "embedding provider not found")
}

func (s *NativeStore) CreateEmbeddingModel(ctx context.Context, req *pb.CreateEmbeddingModelRequest) (*pb.EmbeddingModelResponse, error) {
	if req.GetProviderId() <= 0 || strings.TrimSpace(req.GetModelName()) == "" {
		return &pb.EmbeddingModelResponse{Code: configBadRequest, Msg: "provider_id and model_name are required"}, nil
	}
	row := embeddingModelRecord{ProviderID: req.GetProviderId(), ModelName: strings.TrimSpace(req.GetModelName()), DisplayName: strings.TrimSpace(req.GetDisplayName()), EmbeddingDim: int(req.GetEmbeddingDim()), InputTokenLimit: int(req.GetInputTokenLimit()), BatchSize: defaultInt32(req.GetBatchSize(), 1), TimeoutSeconds: defaultInt32(req.GetTimeoutSeconds(), 30), MaxRetries: defaultInt32(req.GetMaxRetries(), 2), IsEnabled: true, IsDefault: req.GetIsDefault(), LastTestStatus: "untested"}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if row.IsDefault {
			if err := tx.Model(&embeddingModelRecord{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(&row).Error
	})
	if err != nil {
		return nil, err
	}
	return s.getEmbeddingModelResponse(ctx, row.ID)
}

func (s *NativeStore) UpdateEmbeddingModel(ctx context.Context, req *pb.UpdateEmbeddingModelRequest) (*pb.EmbeddingModelResponse, error) {
	updates := map[string]any{}
	putString(updates, "model_name", req.GetModelName())
	putString(updates, "display_name", req.GetDisplayName())
	if req.GetEmbeddingDimSet() {
		updates["embedding_dim"] = req.GetEmbeddingDim()
	}
	if req.GetInputTokenLimitSet() {
		updates["input_token_limit"] = req.GetInputTokenLimit()
	}
	if req.GetBatchSizeSet() {
		updates["batch_size"] = req.GetBatchSize()
	}
	if req.GetTimeoutSecondsSet() {
		updates["timeout_seconds"] = req.GetTimeoutSeconds()
	}
	if req.GetMaxRetriesSet() {
		updates["max_retries"] = req.GetMaxRetries()
	}
	if req.GetIsEnabledSet() {
		updates["is_enabled"] = req.GetIsEnabled()
	}
	if req.GetIsDefaultSet() {
		updates["is_default"] = req.GetIsDefault()
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if req.GetIsDefaultSet() && req.GetIsDefault() {
			if err := tx.Model(&embeddingModelRecord{}).Where("id <> ? AND is_default = ?", req.GetId(), true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		if len(updates) == 0 {
			return nil
		}
		result := tx.Model(&embeddingModelRecord{}).Where("id = ?", req.GetId()).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if err == gorm.ErrRecordNotFound {
		return &pb.EmbeddingModelResponse{Code: configNotFound, Msg: "embedding model not found"}, nil
	}
	if err != nil {
		return nil, err
	}
	return s.getEmbeddingModelResponse(ctx, req.GetId())
}

func (s *NativeStore) SetDefaultEmbeddingModel(ctx context.Context, req *pb.SetDefaultEmbeddingModelRequest) (*pb.CommonResponse, error) {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&embeddingModelRecord{}).Where("id = ?", req.GetId()).Updates(map[string]any{"is_default": true, "is_enabled": true})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Model(&embeddingModelRecord{}).Where("id <> ? AND is_default = ?", req.GetId(), true).Update("is_default", false).Error
	})
	if err == gorm.ErrRecordNotFound {
		return &pb.CommonResponse{Code: configNotFound, Msg: "embedding model not found"}, nil
	}
	if err != nil {
		return nil, err
	}
	return &pb.CommonResponse{Code: configOK, Msg: "success"}, nil
}

func (s *NativeStore) TestEmbeddingModel(ctx context.Context, req *pb.TestEmbeddingModelRequest) (*pb.TestEmbeddingModelResponse, error) {
	resp := &pb.TestEmbeddingModelResponse{Code: configUnsupported, Msg: "embedding runtime validation is not configured", Success: false, Detail: "native configuration store has no embedding client bound for live model tests"}
	if req.GetModelId() > 0 {
		now := time.Now()
		_ = s.db.WithContext(ctx).Model(&embeddingModelRecord{}).Where("id = ?", req.GetModelId()).Updates(map[string]any{"last_test_status": "unsupported", "last_test_error": resp.Detail, "last_test_at": now}).Error
	}
	return resp, nil
}

func (s *NativeStore) BackfillEmbeddings(context.Context, *pb.BackfillEmbeddingsRequest) (*pb.BackfillEmbeddingsResponse, error) {
	return &pb.BackfillEmbeddingsResponse{Code: configUnsupported, Msg: "embedding backfill worker is not configured in native configuration service"}, nil
}

func (s *NativeStore) ResolveEmbeddingConfig(ctx context.Context, providerID, modelID int64) (embeddinginfra.EmbeddingConfig, bool, error) {
	var model embeddingModelRecord
	query := s.db.WithContext(ctx).Model(&embeddingModelRecord{}).Where("is_enabled = ?", true)
	if modelID > 0 {
		query = query.Where("id = ?", modelID)
	} else {
		query = query.Where("is_default = ?", true)
		if providerID > 0 {
			query = query.Where("provider_id = ?", providerID)
		}
	}
	if err := query.Order("is_default DESC, id ASC").First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return embeddinginfra.EmbeddingConfig{}, false, nil
		}
		return embeddinginfra.EmbeddingConfig{}, false, err
	}
	var provider embeddingProviderRecord
	providerQuery := s.db.WithContext(ctx).Where("id = ? AND is_enabled = ?", model.ProviderID, true)
	if providerID > 0 {
		providerQuery = providerQuery.Where("id = ?", providerID)
	}
	if err := providerQuery.First(&provider).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return embeddinginfra.EmbeddingConfig{}, false, nil
		}
		return embeddinginfra.EmbeddingConfig{}, false, err
	}
	apiKey, err := s.decryptAPIKey(provider.APIKeyEncrypted)
	if err != nil {
		return embeddinginfra.EmbeddingConfig{}, false, fmt.Errorf("decrypt embedding provider %d api key: %w", provider.ID, err)
	}
	return embeddinginfra.EmbeddingConfig{
		ProviderID:     provider.ID,
		ProviderName:   provider.Name,
		ProviderType:   provider.ProviderType,
		Endpoint:       provider.Endpoint,
		APIKey:         apiKey,
		ExtraHeaders:   nullString(provider.ExtraHeaders),
		ModelID:        model.ID,
		ModelName:      model.ModelName,
		Dimension:      model.EmbeddingDim,
		BatchSize:      model.BatchSize,
		TimeoutSeconds: model.TimeoutSeconds,
		MaxRetries:     model.MaxRetries,
	}, true, nil
}

func (s *NativeStore) UpdateEmbeddingTestStatus(ctx context.Context, modelID int64, status, lastError string, testedAt time.Time) error {
	if modelID <= 0 {
		return nil
	}
	updates := map[string]any{"last_test_status": strings.TrimSpace(status), "last_test_at": testedAt}
	if strings.TrimSpace(lastError) == "" {
		updates["last_test_error"] = sql.NullString{}
	} else {
		updates["last_test_error"] = sql.NullString{String: strings.TrimSpace(lastError), Valid: true}
	}
	return s.db.WithContext(ctx).Model(&embeddingModelRecord{}).Where("id = ?", modelID).Updates(updates).Error
}

func (s *NativeStore) ListAgentSkillEmbeddingDocuments(ctx context.Context, objectID int64, limit int) ([]embeddinginfra.AgentSkillEmbeddingDocument, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	type row struct {
		agentSkillRecord
		BodyMarkdown sql.NullString `gorm:"column:body_markdown"`
		SkillMD      sql.NullString `gorm:"column:skill_md"`
	}
	query := s.db.WithContext(ctx).Table("agent_skills s").
		Select("s.*, v.body_markdown, v.skill_md").
		Joins("LEFT JOIN agent_skill_versions v ON v.id = s.current_version_id").
		Where("s.is_enabled = ? AND s.current_version_id IS NOT NULL", true)
	if objectID > 0 {
		query = query.Where("s.id = ?", objectID)
	}
	var rows []row
	if err := query.Order("s.priority DESC, s.id ASC").Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	docs := make([]embeddinginfra.AgentSkillEmbeddingDocument, 0, len(rows))
	for _, item := range rows {
		body := nullString(item.BodyMarkdown)
		if strings.TrimSpace(body) == "" {
			body = nullString(item.SkillMD)
		}
		docs = append(docs, embeddinginfra.AgentSkillEmbeddingDocument{
			ID:                   item.ID,
			Name:                 item.Name,
			DisplayName:          item.DisplayName,
			Description:          nullString(item.Description),
			AgentType:            item.AgentType,
			Category:             item.Category,
			Scenario:             item.Scenario,
			Priority:             item.Priority,
			RiskLevel:            item.RiskLevel,
			TriggerKeywords:      jsonStringList(item.TriggerKeywords),
			RequiredCapabilities: jsonStringList(item.RequiredCapabilities),
			EvaluationCriteria:   jsonStringList(item.EvaluationCriteria),
			SemanticTags:         jsonStringList(item.SemanticTags),
			OutputSchema:         nullString(item.OutputSchema),
			BodyMarkdown:         body,
			Enabled:              item.IsEnabled,
		})
	}
	return docs, nil
}

func (s *NativeStore) UpsertAIEmbedding(ctx context.Context, row embeddinginfra.AIEmbeddingRecord) error {
	vector, err := json.Marshal(row.Vector)
	if err != nil {
		return err
	}
	metadata, err := json.Marshal(row.Metadata)
	if err != nil {
		return err
	}
	record := aiEmbeddingRecord{
		ObjectType:     strings.TrimSpace(row.ObjectType),
		ObjectID:       row.ObjectID,
		ScopeType:      strings.TrimSpace(row.ScopeType),
		ScopeID:        row.ScopeID,
		TextHash:       strings.TrimSpace(row.TextHash),
		EmbeddingModel: strings.TrimSpace(row.EmbeddingModel),
		EmbeddingDim:   row.EmbeddingDim,
		VectorJSON:     sql.NullString{String: string(vector), Valid: len(row.Vector) > 0},
		MetadataJSON:   sql.NullString{String: string(metadata), Valid: row.Metadata != nil},
		Status:         defaultString(strings.TrimSpace(row.Status), "ready"),
		LastError:      nullStringFrom(row.LastError, false),
	}
	if record.ObjectType == "" || record.ObjectID <= 0 || record.TextHash == "" || record.EmbeddingModel == "" {
		return fmt.Errorf("invalid embedding record")
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "object_type"}, {Name: "object_id"}, {Name: "embedding_model"}, {Name: "text_hash"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"scope_type", "scope_id", "embedding_dim", "vector_json", "metadata_json", "status", "last_error", "updated_at",
		}),
	}).Create(&record).Error
}

func (s *NativeStore) InvalidateAIEmbedding(ctx context.Context, objectType string, objectID int64) error {
	if strings.TrimSpace(objectType) == "" || objectID <= 0 {
		return nil
	}
	return s.db.WithContext(ctx).Model(&aiEmbeddingRecord{}).
		Where("object_type = ? AND object_id = ?", strings.TrimSpace(objectType), objectID).
		Updates(map[string]any{"status": "invalidated", "last_error": sql.NullString{String: "object disabled or changed", Valid: true}}).Error
}

func (s *NativeStore) ListAIEmbeddings(ctx context.Context, objectType, modelName string, limit int) ([]embeddinginfra.AIEmbeddingRecord, error) {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	var rows []aiEmbeddingRecord
	query := s.db.WithContext(ctx).Where("object_type = ? AND embedding_model = ? AND status = ?", strings.TrimSpace(objectType), strings.TrimSpace(modelName), "ready")
	if err := query.Order("updated_at DESC, id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]embeddinginfra.AIEmbeddingRecord, 0, len(rows))
	for _, row := range rows {
		var vector []float64
		if row.VectorJSON.Valid {
			_ = json.Unmarshal([]byte(row.VectorJSON.String), &vector)
		}
		var metadata map[string]any
		if row.MetadataJSON.Valid {
			_ = json.Unmarshal([]byte(row.MetadataJSON.String), &metadata)
		}
		out = append(out, embeddinginfra.AIEmbeddingRecord{
			ObjectType:     row.ObjectType,
			ObjectID:       row.ObjectID,
			ScopeType:      row.ScopeType,
			ScopeID:        row.ScopeID,
			TextHash:       row.TextHash,
			EmbeddingModel: row.EmbeddingModel,
			EmbeddingDim:   row.EmbeddingDim,
			Vector:         vector,
			Metadata:       metadata,
			Status:         row.Status,
			LastError:      nullString(row.LastError),
		})
	}
	return out, nil
}

func (s *NativeStore) getLlmProviderResponse(ctx context.Context, id int64) (*pb.ProviderResponse, error) {
	var row llmProviderRecord
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.ProviderResponse{Code: configNotFound, Msg: "provider not found"}, nil
		}
		return nil, err
	}
	return &pb.ProviderResponse{Code: configOK, Msg: "success", Provider: s.llmProviderToPB(row)}, nil
}

func (s *NativeStore) getLlmModelResponse(ctx context.Context, id int64) (*pb.ModelResponse, error) {
	var row llmModelListRow
	if err := s.llmModelSelect(ctx).Where("m.id = ?", id).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.ModelResponse{Code: configNotFound, Msg: "model not found"}, nil
		}
		return nil, err
	}
	return &pb.ModelResponse{Code: configOK, Msg: "success", Model: llmModelToPB(row)}, nil
}

func (s *NativeStore) getPromptTemplateResponse(ctx context.Context, id int64) (*pb.PromptTemplateResponse, error) {
	var row promptTemplateRecord
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.PromptTemplateResponse{Code: configNotFound, Msg: "prompt template not found"}, nil
		}
		return nil, err
	}
	return &pb.PromptTemplateResponse{Code: configOK, Msg: "success", Template: promptTemplateToPB(row)}, nil
}

func (s *NativeStore) getAgentResponse(ctx context.Context, id int64) (*pb.AgentConfigResponse, error) {
	var row agentConfigListRow
	if err := s.agentConfigSelect(ctx).Where("a.id = ?", id).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.AgentConfigResponse{Code: configNotFound, Msg: "agent not found"}, nil
		}
		return nil, err
	}
	agent, err := s.agentConfigInfo(ctx, row)
	if err != nil {
		return nil, err
	}
	return &pb.AgentConfigResponse{Code: configOK, Msg: "success", Agent: agent}, nil
}

func (s *NativeStore) getEmbeddingProviderResponse(ctx context.Context, id int64) (*pb.EmbeddingProviderResponse, error) {
	var row embeddingProviderRecord
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.EmbeddingProviderResponse{Code: configNotFound, Msg: "embedding provider not found"}, nil
		}
		return nil, err
	}
	return &pb.EmbeddingProviderResponse{Code: configOK, Msg: "success", Provider: s.embeddingProviderToPB(row)}, nil
}

func (s *NativeStore) getEmbeddingModelResponse(ctx context.Context, id int64) (*pb.EmbeddingModelResponse, error) {
	var row embeddingModelListRow
	if err := s.embeddingModelSelect(ctx).Where("m.id = ?", id).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.EmbeddingModelResponse{Code: configNotFound, Msg: "embedding model not found"}, nil
		}
		return nil, err
	}
	return &pb.EmbeddingModelResponse{Code: configOK, Msg: "success", Model: embeddingModelToPB(row)}, nil
}

func (s *NativeStore) llmModelSelect(ctx context.Context) *gorm.DB {
	return s.db.WithContext(ctx).Table("llm_models m").Select("m.*, p.name AS provider_name").Joins("LEFT JOIN llm_providers p ON p.id = m.provider_id")
}

func (s *NativeStore) agentConfigSelect(ctx context.Context) *gorm.DB {
	return s.db.WithContext(ctx).Table("agent_configs a").Select("a.*, p.name AS prompt_template_name").Joins("LEFT JOIN prompt_templates p ON p.id = a.prompt_template_id")
}

func (s *NativeStore) embeddingModelSelect(ctx context.Context) *gorm.DB {
	return s.db.WithContext(ctx).Table("embedding_models m").Select("m.*, p.name AS provider_name").Joins("LEFT JOIN embedding_providers p ON p.id = m.provider_id")
}

func (s *NativeStore) agentConfigInfo(ctx context.Context, row agentConfigListRow) (*pb.AgentConfigInfo, error) {
	info := &pb.AgentConfigInfo{Id: row.ID, Name: row.Name, DisplayName: row.DisplayName, Description: nullString(row.Description), AgentType: row.AgentType, PromptTemplateId: nullInt64(row.PromptTemplateID), Instruction: nullString(row.Instruction), MaxIterations: int32(row.MaxIterations), TemperatureOverride: nullFloat64(row.TemperatureOverride), IsDefault: row.IsDefault, IsEnabled: row.IsEnabled, CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt), PromptTemplateName: row.PromptTemplateName}
	var tools []agentToolBindingRecord
	if err := s.db.WithContext(ctx).Where("agent_id = ?", row.ID).Order("id ASC").Find(&tools).Error; err != nil {
		return nil, err
	}
	for _, tool := range tools {
		info.ToolBindings = append(info.ToolBindings, &pb.AgentToolBindingInfo{Id: tool.ID, AgentId: tool.AgentID, ToolName: tool.ToolName, IsEnabled: tool.IsEnabled, CreatedAt: formatTime(tool.CreatedAt)})
	}
	var caps []agentCapabilityBindingRecord
	if err := s.db.WithContext(ctx).Where("agent_id = ?", row.ID).Order("priority DESC, id ASC").Find(&caps).Error; err != nil {
		return nil, err
	}
	for _, cap := range caps {
		info.CapabilityBindings = append(info.CapabilityBindings, &pb.AgentCapabilityBindingInfo{Id: cap.ID, AgentId: cap.AgentID, CapabilitySource: cap.CapabilitySource, CapabilityKey: cap.CapabilityKey, IsEnabled: cap.IsEnabled, Priority: int32(cap.Priority), PolicyJson: nullString(cap.PolicyJSON), CreatedAt: formatTime(cap.CreatedAt), UpdatedAt: formatTime(cap.UpdatedAt)})
	}
	return info, nil
}

func replaceAgentBindings(tx *gorm.DB, agentID int64, toolNames []string, caps []*pb.AgentCapabilityBindingInfo, replaceTools, replaceCaps bool) error {
	if replaceTools {
		if err := tx.Where("agent_id = ?", agentID).Delete(&agentToolBindingRecord{}).Error; err != nil {
			return err
		}
		for _, name := range toolNames {
			if strings.TrimSpace(name) == "" {
				continue
			}
			if err := tx.Create(&agentToolBindingRecord{AgentID: agentID, ToolName: strings.TrimSpace(name), IsEnabled: true}).Error; err != nil {
				return err
			}
		}
	}
	if replaceCaps {
		if err := tx.Where("agent_id = ?", agentID).Delete(&agentCapabilityBindingRecord{}).Error; err != nil {
			return err
		}
		for _, cap := range caps {
			if cap == nil || strings.TrimSpace(cap.GetCapabilitySource()) == "" || strings.TrimSpace(cap.GetCapabilityKey()) == "" {
				continue
			}
			if err := tx.Create(&agentCapabilityBindingRecord{AgentID: agentID, CapabilitySource: strings.TrimSpace(cap.GetCapabilitySource()), CapabilityKey: strings.TrimSpace(cap.GetCapabilityKey()), IsEnabled: cap.GetIsEnabled(), Priority: int(cap.GetPriority()), PolicyJSON: nullableJSONText(cap.GetPolicyJson())}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *NativeStore) llmProviderToPB(row llmProviderRecord) *pb.LlmProviderInfo {
	return &pb.LlmProviderInfo{Id: row.ID, Name: row.Name, BaseUrl: row.BaseURL, ApiKeyMasked: s.maskStoredAPIKey(row.APIKeyEncrypted), ProviderType: row.ProviderType, ExtraHeadersJson: redactHeaders(nullString(row.ExtraHeaders)), IsEnabled: row.IsEnabled, CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt)}
}

func llmModelToPB(row llmModelListRow) *pb.LlmModelInfo {
	return &pb.LlmModelInfo{Id: row.ID, ProviderId: row.ProviderID, ModelName: row.ModelName, DisplayName: row.DisplayName, Temperature: row.Temperature, TopP: row.TopP, MaxTokens: int32(row.MaxTokens), ContextWindowTokens: int32(row.ContextWindowTokens), MaxConcurrency: int32(row.MaxConcurrency), TimeoutSeconds: int32(row.TimeoutSeconds), IsEnabled: row.IsEnabled, IsDefault: row.IsDefault, CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt), ProviderName: row.ProviderName}
}

func promptTemplateToPB(row promptTemplateRecord) *pb.PromptTemplateInfo {
	return &pb.PromptTemplateInfo{Id: row.ID, Name: row.Name, Content: row.Content, VariablesJson: nullString(row.Variables), Version: int32(row.Version), IsActive: row.IsActive, AgentType: row.AgentType, PromptRole: row.PromptRole, CreatedBy: nullInt64(row.CreatedBy), UpdatedBy: nullInt64(row.UpdatedBy), CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt)}
}

func (s *NativeStore) embeddingProviderToPB(row embeddingProviderRecord) *pb.EmbeddingProviderInfo {
	return &pb.EmbeddingProviderInfo{Id: row.ID, Name: row.Name, ProviderType: row.ProviderType, Endpoint: row.Endpoint, ApiKeyMasked: s.maskStoredAPIKey(row.APIKeyEncrypted), ExtraHeadersJson: redactHeaders(nullString(row.ExtraHeaders)), IsEnabled: row.IsEnabled, CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt)}
}

func embeddingModelToPB(row embeddingModelListRow) *pb.EmbeddingModelInfo {
	return &pb.EmbeddingModelInfo{Id: row.ID, ProviderId: row.ProviderID, ModelName: row.ModelName, DisplayName: row.DisplayName, EmbeddingDim: int32(row.EmbeddingDim), InputTokenLimit: int32(row.InputTokenLimit), BatchSize: int32(row.BatchSize), TimeoutSeconds: int32(row.TimeoutSeconds), MaxRetries: int32(row.MaxRetries), IsEnabled: row.IsEnabled, IsDefault: row.IsDefault, LastTestStatus: row.LastTestStatus, LastTestError: nullString(row.LastTestError), LastTestAt: formatTimePtr(row.LastTestAt), CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt), ProviderName: row.ProviderName}
}

func rowsCommon(result *gorm.DB, okMsg, missingMsg string) (*pb.CommonResponse, error) {
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return &pb.CommonResponse{Code: configNotFound, Msg: missingMsg}, nil
	}
	return &pb.CommonResponse{Code: configOK, Msg: okMsg}, nil
}

func putString(updates map[string]any, key, value string) {
	if strings.TrimSpace(value) != "" {
		updates[key] = strings.TrimSpace(value)
	}
}

func nullStringFrom(value string, allowEmpty bool) sql.NullString {
	if strings.TrimSpace(value) == "" && !allowEmpty {
		return sql.NullString{}
	}
	return sql.NullString{String: strings.TrimSpace(value), Valid: strings.TrimSpace(value) != "" || allowEmpty}
}

func nullableJSONText(value string) sql.NullString {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: trimmed, Valid: true}
}

func nullInt64From(value int64) sql.NullInt64 {
	return sql.NullInt64{Int64: value, Valid: value > 0}
}

func defaultFloat(value, fallback float64) float64 {
	if value == 0 {
		return fallback
	}
	return value
}

func defaultInt32(value int32, fallback int) int {
	if value == 0 {
		return fallback
	}
	return int(value)
}

func pageDefaults(page, pageSize int32) (int32, int32) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func redactHeaders(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return `{"redacted":true}`
}
