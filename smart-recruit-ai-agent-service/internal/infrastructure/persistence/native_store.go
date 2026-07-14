package persistence

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	aiagentgrpc "smart-recruit-ai-agent-service/internal/interfaces/grpc"
	"smart-recruit-commons/pkg/crypto"
	"smart-recruit-proto/recruitment/pb"
)

type NativeStore struct {
	db               *gorm.DB
	runtimeLLM       RuntimeLLMConfig
	encryptionKey    crypto.EncryptionKey
	hasEncryptionKey bool
}

const (
	chatOwnerRoleLegacyHR  int32 = 0
	chatOwnerRoleCandidate int32 = 1
	chatOwnerRoleHR        int32 = 2
)

func NewNativeStore(db *gorm.DB) *NativeStore {
	return &NativeStore{db: db}
}

func (s *NativeStore) SetRuntimeLLMConfig(cfg RuntimeLLMConfig) {
	s.runtimeLLM = cfg
}

func (s *NativeStore) SetEncryptionKey(key crypto.EncryptionKey) {
	s.encryptionKey = key
	s.hasEncryptionKey = true
}

func (s *NativeStore) encryptAPIKey(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("api_key is required")
	}
	if !s.hasEncryptionKey {
		return "", fmt.Errorf("api key encryption key is not configured")
	}
	return crypto.Encrypt(s.encryptionKey, []byte(trimmed))
}

func (s *NativeStore) decryptAPIKey(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("api_key is empty")
	}
	if !s.hasEncryptionKey {
		if looksEncryptedAPIKey(trimmed) {
			return "", fmt.Errorf("api key encryption key is not configured")
		}
		return trimmed, nil
	}
	plaintext, err := crypto.Decrypt(s.encryptionKey, trimmed)
	if err == nil {
		return strings.TrimSpace(string(plaintext)), nil
	}
	if looksEncryptedAPIKey(trimmed) {
		return "", fmt.Errorf("decrypt api key: %w", err)
	}
	return trimmed, nil
}

func (s *NativeStore) maskStoredAPIKey(value string) string {
	plaintext, err := s.decryptAPIKey(value)
	if err != nil {
		return "********"
	}
	return crypto.MaskAPIKey(plaintext)
}

func looksEncryptedAPIKey(value string) bool {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < 96 || len(trimmed)%2 != 0 {
		return false
	}
	_, err := hex.DecodeString(trimmed)
	return err == nil
}

func (s *NativeStore) EnsureChatSession(ctx context.Context, ownerRole int32, ownerID int64, title string, applicationID int64) (aiagentgrpc.ChatSessionRow, error) {
	now := time.Now()
	session := aiChatSessionRecord{
		HRID:          chatCompatibilityHRID(ownerRole, ownerID),
		OwnerRole:     ownerRole,
		OwnerID:       ownerID,
		Title:         title,
		ApplicationID: applicationID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return aiagentgrpc.ChatSessionRow{}, err
	}
	return mapSessionRecord(session), nil
}

func (s *NativeStore) ListChatSessions(ctx context.Context, ownerRole int32, ownerID int64, page, pageSize int32) ([]aiagentgrpc.ChatSessionRow, int64, error) {
	var total int64
	query := s.db.WithContext(ctx).Model(&aiChatSessionRecord{}).
		Where("deleted_at IS NULL")
	query = applyChatOwnerScope(query, ownerRole, ownerID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []aiChatSessionRecord
	if err := query.Order("updated_at DESC, id DESC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]aiagentgrpc.ChatSessionRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapSessionRecord(row))
	}
	return result, total, nil
}

func (s *NativeStore) UpdateChatSessionTitle(ctx context.Context, ownerRole int32, ownerID, sessionID int64, title string) error {
	query := s.db.WithContext(ctx).Model(&aiChatSessionRecord{}).Where("id = ? AND deleted_at IS NULL", sessionID)
	query = applyChatOwnerScope(query, ownerRole, ownerID)
	return query.Update("title", title).Error
}

func (s *NativeStore) DeleteChatSession(ctx context.Context, ownerRole int32, ownerID, sessionID int64) error {
	now := time.Now()
	query := s.db.WithContext(ctx).Model(&aiChatSessionRecord{}).Where("id = ? AND deleted_at IS NULL", sessionID)
	query = applyChatOwnerScope(query, ownerRole, ownerID)
	return query.Update("deleted_at", &now).Error
}

func (s *NativeStore) AppendChatMessage(ctx context.Context, message aiagentgrpc.ChatMessageRow) (aiagentgrpc.ChatMessageRow, error) {
	now := time.Now()
	if !message.CreatedAt.IsZero() {
		now = message.CreatedAt
	}
	row := aiChatHistoryRecord{
		HRID:           chatCompatibilityHRID(message.OwnerRole, message.OwnerID),
		OwnerRole:      message.OwnerRole,
		OwnerID:        message.OwnerID,
		SessionID:      message.SessionID,
		Role:           message.Role,
		Content:        message.Content,
		ProcessContent: message.ProcessContent,
		ModelID:        message.ModelID,
		ModelName:      message.ModelName,
		CreatedAt:      now,
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.Model(&aiChatSessionRecord{}).Where("id = ?", message.SessionID).Update("updated_at", now).Error
	}); err != nil {
		return aiagentgrpc.ChatMessageRow{}, err
	}
	return mapMessageRecord(row), nil
}

func (s *NativeStore) ListChatMessages(ctx context.Context, ownerRole int32, ownerID, sessionID int64, page, pageSize int32) ([]aiagentgrpc.ChatMessageRow, error) {
	query := s.db.WithContext(ctx).Model(&aiChatHistoryRecord{})
	query = applyChatOwnerScope(query, ownerRole, ownerID)
	if sessionID > 0 {
		query = query.Where("session_id = ?", sessionID)
	}
	var rows []aiChatHistoryRecord
	if err := query.Order("created_at ASC, id ASC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]aiagentgrpc.ChatMessageRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapMessageRecord(row))
	}
	return result, nil
}

func (s *NativeStore) ListToolTraces(ctx context.Context, ownerID, sessionID int64) ([]aiagentgrpc.ToolTraceRow, error) {
	var rows []aiToolTraceRecord
	query := s.db.WithContext(ctx).Where("hr_id = ?", ownerID)
	if sessionID > 0 {
		query = query.Where("session_id = ?", sessionID)
	}
	if err := query.Order("created_at DESC, id DESC").Limit(200).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]aiagentgrpc.ToolTraceRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, aiagentgrpc.ToolTraceRow{ID: row.ID, SessionID: row.SessionID, ToolName: row.ToolName, ArgsJSON: row.ArgsJSON, ResultContent: row.ResultContent, DurationMs: row.DurationMs, ErrorMsg: row.ErrorMsg, CreatedAt: row.CreatedAt})
	}
	return result, nil
}

func (s *NativeStore) CreateAgentRun(ctx context.Context, run aiagentgrpc.AgentRunRow) (aiagentgrpc.AgentRunRow, bool, error) {
	if run.ClientRequestID != "" {
		var existing agentRunRecord
		err := s.db.WithContext(ctx).Where("hr_id = ? AND session_id = ? AND client_request_id = ?", run.OwnerID, run.SessionID, run.ClientRequestID).First(&existing).Error
		if err == nil {
			return mapRunRecord(existing), true, nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return aiagentgrpc.AgentRunRow{}, false, err
		}
	}
	now := time.Now()
	row := agentRunRecord{
		SessionID:       run.SessionID,
		HRID:            run.OwnerID,
		ClientRequestID: run.ClientRequestID,
		Status:          "queued",
		OptionContext:   nullableJSON(run.OptionContextJSON),
		ModelID:         run.ModelID,
		ModelName:       run.ModelName,
		AgentType:       defaultString(run.AgentType, "hr"),
		AgentName:       defaultString(run.AgentName, "hr_recruiting_agent"),
		StartedAt:       now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.Model(&aiChatSessionRecord{}).Where("id = ?", row.SessionID).Update("active_run_id", row.ID).Error
	}); err != nil {
		return aiagentgrpc.AgentRunRow{}, false, err
	}
	return mapRunRecord(row), false, nil
}

func (s *NativeStore) ListAgentRuns(ctx context.Context, ownerID, sessionID int64) ([]aiagentgrpc.AgentRunRow, error) {
	var rows []agentRunRecord
	query := s.db.WithContext(ctx).Where("hr_id = ?", ownerID)
	if sessionID > 0 {
		query = query.Where("session_id = ?", sessionID)
	}
	if err := query.Order("created_at DESC, id DESC").Limit(100).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]aiagentgrpc.AgentRunRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapRunRecord(row))
	}
	return result, nil
}

func (s *NativeStore) GetAgentRun(ctx context.Context, ownerID, runID int64) (aiagentgrpc.AgentRunRow, bool, error) {
	var row agentRunRecord
	err := s.db.WithContext(ctx).Where("id = ? AND hr_id = ?", runID, ownerID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return aiagentgrpc.AgentRunRow{}, false, nil
	}
	if err != nil {
		return aiagentgrpc.AgentRunRow{}, false, err
	}
	return mapRunRecord(row), true, nil
}

func (s *NativeStore) GetActiveAgentRun(ctx context.Context, ownerID, sessionID int64) (aiagentgrpc.AgentRunRow, bool, error) {
	var row agentRunRecord
	err := s.db.WithContext(ctx).
		Where("hr_id = ? AND session_id = ? AND status IN ?", ownerID, sessionID, []string{"queued", "planning", "running", "waiting_confirmation", "cancel_requested"}).
		Order("updated_at DESC, id DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return aiagentgrpc.AgentRunRow{}, false, nil
	}
	if err != nil {
		return aiagentgrpc.AgentRunRow{}, false, err
	}
	return mapRunRecord(row), true, nil
}

func (s *NativeStore) UpdateAgentRunStatus(ctx context.Context, ownerID, runID int64, status string) (aiagentgrpc.AgentRunRow, bool, error) {
	now := time.Now()
	updates := map[string]any{"status": status, "updated_at": now}
	if status == "cancel_requested" {
		updates["cancel_requested_at"] = now
	}
	if status == "canceled" || status == "succeeded" || status == "failed" {
		updates["completed_at"] = now
	}
	result := s.db.WithContext(ctx).Model(&agentRunRecord{}).Where("id = ? AND hr_id = ?", runID, ownerID).Updates(updates)
	if result.Error != nil {
		return aiagentgrpc.AgentRunRow{}, false, result.Error
	}
	if result.RowsAffected == 0 {
		return aiagentgrpc.AgentRunRow{}, false, nil
	}
	return s.GetAgentRun(ctx, ownerID, runID)
}

func (s *NativeStore) CompleteAgentRun(ctx context.Context, ownerID, runID int64, assistantText, status, errorType, errorMessage string) (aiagentgrpc.AgentRunRow, bool, error) {
	now := time.Now()
	if status == "" {
		status = "succeeded"
	}
	updates := map[string]any{
		"status":         status,
		"assistant_text": assistantText,
		"updated_at":     now,
	}
	if status == "succeeded" || status == "failed" || status == "canceled" {
		updates["completed_at"] = now
	}
	if errorType != "" {
		updates["error_type"] = errorType
	}
	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}
	result := s.db.WithContext(ctx).Model(&agentRunRecord{}).Where("id = ? AND hr_id = ?", runID, ownerID).Updates(updates)
	if result.Error != nil {
		return aiagentgrpc.AgentRunRow{}, false, result.Error
	}
	if result.RowsAffected == 0 {
		return aiagentgrpc.AgentRunRow{}, false, nil
	}
	return s.GetAgentRun(ctx, ownerID, runID)
}

func (s *NativeStore) AppendAgentRunEvent(ctx context.Context, runID int64, eventType, payload string) (aiagentgrpc.AgentRunEventRow, error) {
	var event agentRunEventRecord
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var run agentRunRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&run, runID).Error; err != nil {
			return err
		}
		event = agentRunEventRecord{RunID: runID, Seq: run.LastEventSeq + 1, EventType: eventType, PayloadJSON: payload, CreatedAt: time.Now()}
		if event.PayloadJSON == "" {
			event.PayloadJSON = "{}"
		}
		if err := tx.Create(&event).Error; err != nil {
			return err
		}
		return tx.Model(&agentRunRecord{}).Where("id = ?", runID).Updates(map[string]any{"last_event_seq": event.Seq, "updated_at": time.Now()}).Error
	})
	if err != nil {
		return aiagentgrpc.AgentRunEventRow{}, err
	}
	return aiagentgrpc.AgentRunEventRow{RunID: event.RunID, Seq: event.Seq, EventType: event.EventType, PayloadJSON: event.PayloadJSON, CreatedAt: event.CreatedAt}, nil
}

func (s *NativeStore) ListAgentRunEvents(ctx context.Context, ownerID, runID, afterSeq int64) ([]aiagentgrpc.AgentRunEventRow, error) {
	var rows []agentRunEventRecord
	if err := s.db.WithContext(ctx).
		Table("agent_run_events").
		Joins("JOIN agent_runs ON agent_runs.id = agent_run_events.run_id").
		Where("agent_run_events.run_id = ? AND agent_runs.hr_id = ? AND agent_run_events.seq > ?", runID, ownerID, afterSeq).
		Order("agent_run_events.seq ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]aiagentgrpc.AgentRunEventRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, aiagentgrpc.AgentRunEventRow{RunID: row.RunID, Seq: row.Seq, EventType: row.EventType, PayloadJSON: row.PayloadJSON, CreatedAt: row.CreatedAt})
	}
	return result, nil
}

func (s *NativeStore) ListLlmProviders(ctx context.Context, page, pageSize int32) ([]*pb.LlmProviderInfo, int64, error) {
	var total int64
	query := s.db.WithContext(ctx).Model(&llmProviderRecord{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []llmProviderRecord
	if err := query.Order("id ASC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]*pb.LlmProviderInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, &pb.LlmProviderInfo{
			Id:               row.ID,
			Name:             row.Name,
			BaseUrl:          row.BaseURL,
			ApiKeyMasked:     maskSecret(row.APIKeyEncrypted),
			ProviderType:     row.ProviderType,
			ExtraHeadersJson: nullString(row.ExtraHeaders),
			IsEnabled:        row.IsEnabled,
			CreatedAt:        formatTime(row.CreatedAt),
			UpdatedAt:        formatTime(row.UpdatedAt),
		})
	}
	return items, total, nil
}

func (s *NativeStore) ListLlmModels(ctx context.Context, page, pageSize int32, providerID int64) ([]*pb.LlmModelInfo, int64, error) {
	var total int64
	query := s.db.WithContext(ctx).Table("llm_models")
	if providerID > 0 {
		query = query.Where("provider_id = ?", providerID)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []llmModelListRow
	selectQuery := s.db.WithContext(ctx).Table("llm_models m").
		Select("m.*, p.name AS provider_name").
		Joins("LEFT JOIN llm_providers p ON p.id = m.provider_id")
	if providerID > 0 {
		selectQuery = selectQuery.Where("m.provider_id = ?", providerID)
	}
	if err := selectQuery.Order("m.is_default DESC, m.id ASC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]*pb.LlmModelInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, &pb.LlmModelInfo{
			Id:                  row.ID,
			ProviderId:          row.ProviderID,
			ModelName:           row.ModelName,
			DisplayName:         row.DisplayName,
			Temperature:         row.Temperature,
			TopP:                row.TopP,
			MaxTokens:           int32(row.MaxTokens),
			ContextWindowTokens: int32(row.ContextWindowTokens),
			MaxConcurrency:      int32(row.MaxConcurrency),
			TimeoutSeconds:      int32(row.TimeoutSeconds),
			IsEnabled:           row.IsEnabled,
			IsDefault:           row.IsDefault,
			CreatedAt:           formatTime(row.CreatedAt),
			UpdatedAt:           formatTime(row.UpdatedAt),
			ProviderName:        row.ProviderName,
		})
	}
	return items, total, nil
}

func (s *NativeStore) ListPromptTemplates(ctx context.Context, page, pageSize int32, agentType string) ([]*pb.PromptTemplateInfo, int64, error) {
	var total int64
	query := s.db.WithContext(ctx).Model(&promptTemplateRecord{})
	if strings.TrimSpace(agentType) != "" {
		query = query.Where("agent_type = ?", strings.TrimSpace(agentType))
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []promptTemplateRecord
	if err := query.Order("updated_at DESC, id ASC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]*pb.PromptTemplateInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, &pb.PromptTemplateInfo{
			Id:            row.ID,
			Name:          row.Name,
			Content:       row.Content,
			VariablesJson: nullString(row.Variables),
			Version:       int32(row.Version),
			IsActive:      row.IsActive,
			AgentType:     row.AgentType,
			PromptRole:    row.PromptRole,
			CreatedBy:     nullInt64(row.CreatedBy),
			UpdatedBy:     nullInt64(row.UpdatedBy),
			CreatedAt:     formatTime(row.CreatedAt),
			UpdatedAt:     formatTime(row.UpdatedAt),
		})
	}
	return items, total, nil
}

func (s *NativeStore) ListAgentConfigs(ctx context.Context, page, pageSize int32, agentType string) ([]*pb.AgentConfigInfo, int64, error) {
	var total int64
	query := s.db.WithContext(ctx).Table("agent_configs")
	if strings.TrimSpace(agentType) != "" {
		query = query.Where("agent_type = ?", strings.TrimSpace(agentType))
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []agentConfigListRow
	selectQuery := s.db.WithContext(ctx).Table("agent_configs a").
		Select("a.*, p.name AS prompt_template_name").
		Joins("LEFT JOIN prompt_templates p ON p.id = a.prompt_template_id")
	if strings.TrimSpace(agentType) != "" {
		selectQuery = selectQuery.Where("a.agent_type = ?", strings.TrimSpace(agentType))
	}
	if err := selectQuery.Order("a.id ASC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]*pb.AgentConfigInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, &pb.AgentConfigInfo{
			Id:                  row.ID,
			Name:                row.Name,
			DisplayName:         row.DisplayName,
			Description:         nullString(row.Description),
			AgentType:           row.AgentType,
			PromptTemplateId:    nullInt64(row.PromptTemplateID),
			Instruction:         nullString(row.Instruction),
			MaxIterations:       int32(row.MaxIterations),
			TemperatureOverride: nullFloat64(row.TemperatureOverride),
			IsDefault:           row.IsDefault,
			IsEnabled:           row.IsEnabled,
			CreatedAt:           formatTime(row.CreatedAt),
			UpdatedAt:           formatTime(row.UpdatedAt),
			PromptTemplateName:  row.PromptTemplateName,
		})
	}
	return items, total, nil
}

func (s *NativeStore) ListMCPServers(ctx context.Context, page, pageSize int32) ([]*pb.MCPServerInfo, int64, error) {
	var total int64
	query := s.db.WithContext(ctx).Model(&mcpServerRecord{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []mcpServerRecord
	if err := query.Order("id ASC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]*pb.MCPServerInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, mcpServerToPB(row))
	}
	return items, total, nil
}

func (s *NativeStore) ListAgentSkills(ctx context.Context, page, pageSize int32, keyword string, enabledOnly bool) ([]*pb.AgentSkillInfo, int64, error) {
	var total int64
	query := s.db.WithContext(ctx).Model(&agentSkillRecord{})
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR display_name LIKE ? OR description LIKE ?", like, like, like)
	}
	if enabledOnly {
		query = query.Where("is_enabled = ?", true)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []agentSkillRecord
	if err := query.Order("priority DESC, id ASC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]*pb.AgentSkillInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, &pb.AgentSkillInfo{
			Id:                   row.ID,
			Name:                 row.Name,
			DisplayName:          row.DisplayName,
			Description:          nullString(row.Description),
			CurrentVersionId:     nullInt64(row.CurrentVersionID),
			IsEnabled:            row.IsEnabled,
			IsManualInvocable:    row.IsManualInvocable,
			TriggerKeywords:      jsonStringList(row.TriggerKeywords),
			CreatedAt:            formatTime(row.CreatedAt),
			UpdatedAt:            formatTime(row.UpdatedAt),
			AgentType:            row.AgentType,
			Category:             row.Category,
			Scenario:             row.Scenario,
			Priority:             int32(row.Priority),
			RiskLevel:            row.RiskLevel,
			RequiredCapabilities: jsonStringList(row.RequiredCapabilities),
			OutputSchema:         nullString(row.OutputSchema),
			EvaluationCriteria:   jsonStringList(row.EvaluationCriteria),
			SemanticTags:         jsonStringList(row.SemanticTags),
		})
	}
	return items, total, nil
}

func (s *NativeStore) ListEmbeddingProviders(ctx context.Context, page, pageSize int32) ([]*pb.EmbeddingProviderInfo, int64, error) {
	var total int64
	query := s.db.WithContext(ctx).Model(&embeddingProviderRecord{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []embeddingProviderRecord
	if err := query.Order("id ASC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]*pb.EmbeddingProviderInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, &pb.EmbeddingProviderInfo{
			Id:               row.ID,
			Name:             row.Name,
			ProviderType:     row.ProviderType,
			Endpoint:         row.Endpoint,
			ApiKeyMasked:     maskSecret(row.APIKeyEncrypted),
			ExtraHeadersJson: nullString(row.ExtraHeaders),
			IsEnabled:        row.IsEnabled,
			CreatedAt:        formatTime(row.CreatedAt),
			UpdatedAt:        formatTime(row.UpdatedAt),
		})
	}
	return items, total, nil
}

func (s *NativeStore) ListEmbeddingModels(ctx context.Context, page, pageSize int32, providerID int64) ([]*pb.EmbeddingModelInfo, int64, error) {
	var total int64
	query := s.db.WithContext(ctx).Table("embedding_models")
	if providerID > 0 {
		query = query.Where("provider_id = ?", providerID)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []embeddingModelListRow
	selectQuery := s.db.WithContext(ctx).Table("embedding_models m").
		Select("m.*, p.name AS provider_name").
		Joins("LEFT JOIN embedding_providers p ON p.id = m.provider_id")
	if providerID > 0 {
		selectQuery = selectQuery.Where("m.provider_id = ?", providerID)
	}
	if err := selectQuery.Order("m.is_default DESC, m.id ASC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]*pb.EmbeddingModelInfo, 0, len(rows))
	for _, row := range rows {
		items = append(items, &pb.EmbeddingModelInfo{
			Id:              row.ID,
			ProviderId:      row.ProviderID,
			ModelName:       row.ModelName,
			DisplayName:     row.DisplayName,
			EmbeddingDim:    int32(row.EmbeddingDim),
			InputTokenLimit: int32(row.InputTokenLimit),
			BatchSize:       int32(row.BatchSize),
			TimeoutSeconds:  int32(row.TimeoutSeconds),
			MaxRetries:      int32(row.MaxRetries),
			IsEnabled:       row.IsEnabled,
			IsDefault:       row.IsDefault,
			LastTestStatus:  row.LastTestStatus,
			LastTestError:   nullString(row.LastTestError),
			LastTestAt:      formatTimePtr(row.LastTestAt),
			CreatedAt:       formatTime(row.CreatedAt),
			UpdatedAt:       formatTime(row.UpdatedAt),
			ProviderName:    row.ProviderName,
		})
	}
	return items, total, nil
}

type aiChatSessionRecord struct {
	ID            int64      `gorm:"primaryKey"`
	HRID          int64      `gorm:"column:hr_id"`
	OwnerRole     int32      `gorm:"column:owner_role"`
	OwnerID       int64      `gorm:"column:owner_id"`
	Title         string     `gorm:"column:title"`
	ApplicationID int64      `gorm:"column:application_id"`
	ActiveRunID   *int64     `gorm:"column:active_run_id"`
	DeletedAt     *time.Time `gorm:"column:deleted_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

func (aiChatSessionRecord) TableName() string { return "ai_chat_sessions" }

type aiChatHistoryRecord struct {
	ID             int64     `gorm:"primaryKey"`
	HRID           int64     `gorm:"column:hr_id"`
	OwnerRole      int32     `gorm:"column:owner_role"`
	OwnerID        int64     `gorm:"column:owner_id"`
	SessionID      int64     `gorm:"column:session_id"`
	Role           string    `gorm:"column:role"`
	Content        string    `gorm:"column:content"`
	ProcessContent string    `gorm:"column:process_content"`
	ModelID        int64     `gorm:"column:model_id"`
	ModelName      string    `gorm:"column:model_name"`
	AgentRunID     *int64    `gorm:"column:agent_run_id"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (aiChatHistoryRecord) TableName() string { return "ai_chat_history" }

type aiToolTraceRecord struct {
	ID            int64     `gorm:"primaryKey"`
	HRID          int64     `gorm:"column:hr_id"`
	SessionID     int64     `gorm:"column:session_id"`
	ToolName      string    `gorm:"column:tool_name"`
	ArgsJSON      string    `gorm:"column:args_json"`
	ResultContent string    `gorm:"column:result_content"`
	DurationMs    int64     `gorm:"column:duration_ms"`
	ErrorMsg      string    `gorm:"column:error_msg"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (aiToolTraceRecord) TableName() string { return "ai_tool_traces" }

type agentRunRecord struct {
	ID                int64      `gorm:"primaryKey"`
	SessionID         int64      `gorm:"column:session_id"`
	MessageID         int64      `gorm:"column:message_id"`
	HistoryID         int64      `gorm:"column:history_id"`
	HRID              int64      `gorm:"column:hr_id"`
	ClientRequestID   string     `gorm:"column:client_request_id"`
	Status            string     `gorm:"column:status"`
	AssistantText     string     `gorm:"column:assistant_text"`
	ProcessText       string     `gorm:"column:process_text"`
	OptionContext     *string    `gorm:"column:option_context_json"`
	LastEventSeq      int64      `gorm:"column:last_event_seq"`
	ErrorType         string     `gorm:"column:error_type"`
	ErrorMessage      string     `gorm:"column:error_message"`
	ModelID           int64      `gorm:"column:model_id"`
	ModelName         string     `gorm:"column:model_name"`
	AgentType         string     `gorm:"column:agent_type"`
	AgentID           int64      `gorm:"column:agent_id"`
	AgentName         string     `gorm:"column:agent_name"`
	StartedAt         time.Time  `gorm:"column:started_at"`
	CompletedAt       *time.Time `gorm:"column:completed_at"`
	CancelRequestedAt *time.Time `gorm:"column:cancel_requested_at"`
	CanceledAt        *time.Time `gorm:"column:canceled_at"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
}

func (agentRunRecord) TableName() string { return "agent_runs" }

type agentRunEventRecord struct {
	ID          int64     `gorm:"primaryKey"`
	RunID       int64     `gorm:"column:run_id"`
	Seq         int64     `gorm:"column:seq"`
	EventType   string    `gorm:"column:event_type"`
	PayloadJSON string    `gorm:"column:payload_json"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

func (agentRunEventRecord) TableName() string { return "agent_run_events" }

type llmProviderRecord struct {
	ID              int64          `gorm:"primaryKey"`
	Name            string         `gorm:"column:name"`
	BaseURL         string         `gorm:"column:base_url"`
	APIKeyEncrypted string         `gorm:"column:api_key_encrypted"`
	ProviderType    string         `gorm:"column:provider_type"`
	ExtraHeaders    sql.NullString `gorm:"column:extra_headers"`
	IsEnabled       bool           `gorm:"column:is_enabled"`
	CreatedAt       time.Time      `gorm:"column:created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at"`
}

func (llmProviderRecord) TableName() string { return "llm_providers" }

type llmModelListRow struct {
	ID                  int64     `gorm:"column:id"`
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
	ProviderName        string    `gorm:"column:provider_name"`
}

type promptTemplateRecord struct {
	ID         int64          `gorm:"primaryKey"`
	Name       string         `gorm:"column:name"`
	Content    string         `gorm:"column:content"`
	Variables  sql.NullString `gorm:"column:variables"`
	Version    int            `gorm:"column:version"`
	IsActive   bool           `gorm:"column:is_active"`
	AgentType  string         `gorm:"column:agent_type"`
	PromptRole string         `gorm:"column:prompt_role"`
	CreatedBy  sql.NullInt64  `gorm:"column:created_by"`
	UpdatedBy  sql.NullInt64  `gorm:"column:updated_by"`
	CreatedAt  time.Time      `gorm:"column:created_at"`
	UpdatedAt  time.Time      `gorm:"column:updated_at"`
}

func (promptTemplateRecord) TableName() string { return "prompt_templates" }

type agentConfigListRow struct {
	ID                  int64           `gorm:"column:id"`
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
	PromptTemplateName  string          `gorm:"column:prompt_template_name"`
}

type mcpServerRecord struct {
	ID             int64          `gorm:"primaryKey"`
	Name           string         `gorm:"column:name"`
	Description    sql.NullString `gorm:"column:description"`
	Transport      string         `gorm:"column:transport"`
	CommandOrURL   string         `gorm:"column:command_or_url"`
	Args           sql.NullString `gorm:"column:args"`
	EnvVars        sql.NullString `gorm:"column:env_vars"`
	TimeoutSeconds int            `gorm:"column:timeout_seconds"`
	IsEnabled      bool           `gorm:"column:is_enabled"`
	Status         string         `gorm:"column:status"`
	ToolCount      int            `gorm:"column:tool_count"`
	LastError      sql.NullString `gorm:"column:last_error"`
	CreatedAt      time.Time      `gorm:"column:created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at"`
}

func (mcpServerRecord) TableName() string { return "mcp_servers" }

type agentSkillRecord struct {
	ID                   int64          `gorm:"primaryKey"`
	Name                 string         `gorm:"column:name"`
	DisplayName          string         `gorm:"column:display_name"`
	Description          sql.NullString `gorm:"column:description"`
	CurrentVersionID     sql.NullInt64  `gorm:"column:current_version_id"`
	IsEnabled            bool           `gorm:"column:is_enabled"`
	IsManualInvocable    bool           `gorm:"column:is_manual_invocable"`
	TriggerKeywords      sql.NullString `gorm:"column:trigger_keywords"`
	AgentType            string         `gorm:"column:agent_type"`
	Category             string         `gorm:"column:category"`
	Scenario             string         `gorm:"column:scenario"`
	Priority             int            `gorm:"column:priority"`
	RiskLevel            string         `gorm:"column:risk_level"`
	RequiredCapabilities sql.NullString `gorm:"column:required_capabilities"`
	OutputSchema         sql.NullString `gorm:"column:output_schema"`
	EvaluationCriteria   sql.NullString `gorm:"column:evaluation_criteria"`
	SemanticTags         sql.NullString `gorm:"column:semantic_tags"`
	CreatedAt            time.Time      `gorm:"column:created_at"`
	UpdatedAt            time.Time      `gorm:"column:updated_at"`
}

func (agentSkillRecord) TableName() string { return "agent_skills" }

type embeddingProviderRecord struct {
	ID              int64          `gorm:"primaryKey"`
	Name            string         `gorm:"column:name"`
	ProviderType    string         `gorm:"column:provider_type"`
	Endpoint        string         `gorm:"column:endpoint"`
	APIKeyEncrypted string         `gorm:"column:api_key_encrypted"`
	ExtraHeaders    sql.NullString `gorm:"column:extra_headers"`
	IsEnabled       bool           `gorm:"column:is_enabled"`
	CreatedAt       time.Time      `gorm:"column:created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at"`
}

func (embeddingProviderRecord) TableName() string { return "embedding_providers" }

type embeddingModelListRow struct {
	ID              int64          `gorm:"column:id"`
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
	ProviderName    string         `gorm:"column:provider_name"`
}

func mapSessionRecord(row aiChatSessionRecord) aiagentgrpc.ChatSessionRow {
	return aiagentgrpc.ChatSessionRow{ID: row.ID, Title: row.Title, ApplicationID: row.ApplicationID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func mapMessageRecord(row aiChatHistoryRecord) aiagentgrpc.ChatMessageRow {
	return aiagentgrpc.ChatMessageRow{ID: row.ID, OwnerRole: row.OwnerRole, OwnerID: row.OwnerID, SessionID: row.SessionID, Role: row.Role, Content: row.Content, ProcessContent: row.ProcessContent, ModelID: row.ModelID, ModelName: row.ModelName, CreatedAt: row.CreatedAt}
}

func applyChatOwnerScope(query *gorm.DB, ownerRole int32, ownerID int64) *gorm.DB {
	if ownerRole != chatOwnerRoleHR {
		return query.Where("owner_role = ? AND owner_id = ?", ownerRole, ownerID)
	}
	return query.Where(
		"((owner_role = ? AND owner_id = ?) OR (hr_id = ? AND owner_role IN ?))",
		chatOwnerRoleHR,
		ownerID,
		ownerID,
		[]int32{chatOwnerRoleLegacyHR, chatOwnerRoleHR},
	)
}

func chatCompatibilityHRID(ownerRole int32, ownerID int64) int64 {
	if ownerRole == chatOwnerRoleHR || ownerRole == chatOwnerRoleLegacyHR {
		return ownerID
	}
	return 0
}

func mapRunRecord(row agentRunRecord) aiagentgrpc.AgentRunRow {
	return aiagentgrpc.AgentRunRow{ID: row.ID, SessionID: row.SessionID, MessageID: row.MessageID, HistoryID: row.HistoryID, OwnerID: row.HRID, ClientRequestID: row.ClientRequestID, Status: row.Status, AssistantText: row.AssistantText, ProcessText: row.ProcessText, OptionContextJSON: stringValue(row.OptionContext), LastEventSeq: row.LastEventSeq, ErrorType: row.ErrorType, ErrorMessage: row.ErrorMessage, ModelID: row.ModelID, ModelName: row.ModelName, AgentType: row.AgentType, AgentID: row.AgentID, AgentName: row.AgentName, StartedAt: row.StartedAt, CompletedAt: row.CompletedAt, CancelRequestedAt: row.CancelRequestedAt, CanceledAt: row.CanceledAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func offset(page, pageSize int32) int {
	return int((page - 1) * pageSize)
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func nullableJSON(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nullString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullInt64(value sql.NullInt64) int64 {
	if !value.Valid {
		return 0
	}
	return value.Int64
}

func nullFloat64(value sql.NullFloat64) float64 {
	if !value.Valid {
		return 0
	}
	return value.Float64
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func formatTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatTime(*value)
}

func maskSecret(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return "********"
}

func jsonStringList(value sql.NullString) []string {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(value.String), &items); err != nil {
		return nil
	}
	return items
}
