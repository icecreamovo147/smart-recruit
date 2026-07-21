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

	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	aiagentgrpc "smart-recruit-ai-agent-service/internal/interfaces/grpc"
	"smart-recruit-commons/pkg/crypto"
	"smart-recruit-platform-go/businessclock"
	platformlogger "smart-recruit-platform-go/logger"
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
	if strings.TrimSpace(value) == "" {
		return ""
	}
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
	return s.EnsureChatSessionWithOptions(ctx, ownerRole, ownerID, title, applicationID, aiagentgrpc.ChatSessionCreateOptions{})
}

func (s *NativeStore) EnsureChatSessionWithOptions(ctx context.Context, ownerRole int32, ownerID int64, title string, applicationID int64, opts aiagentgrpc.ChatSessionCreateOptions) (aiagentgrpc.ChatSessionRow, error) {
	now := time.Now()
	session := aiChatSessionRecord{
		HRID:          chatCompatibilityHRID(ownerRole, ownerID),
		OwnerRole:     ownerRole,
		OwnerID:       ownerID,
		Title:         title,
		ApplicationID: applicationID,
		SessionType:   normalizeChatSessionType(opts.SessionType),
		SourceType:    strings.TrimSpace(opts.SourceType),
		SourceID:      opts.SourceID,
		SourceTitle:   strings.TrimSpace(opts.SourceTitle),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return aiagentgrpc.ChatSessionRow{}, err
	}
	return mapSessionRecord(ctx, session), nil
}

func (s *NativeStore) GetChatSession(ctx context.Context, ownerRole int32, ownerID, sessionID int64) (aiagentgrpc.ChatSessionRow, bool, error) {
	var row aiChatSessionRecord
	query := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", sessionID)
	query = applyChatOwnerScope(query, ownerRole, ownerID)
	if err := query.First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return aiagentgrpc.ChatSessionRow{}, false, nil
		}
		return aiagentgrpc.ChatSessionRow{}, false, err
	}
	return mapSessionRecord(ctx, row), true, nil
}

func (s *NativeStore) ListChatSessions(ctx context.Context, ownerRole int32, ownerID int64, page, pageSize int32) ([]aiagentgrpc.ChatSessionRow, int64, error) {
	return s.ListChatSessionsWithFilter(ctx, ownerRole, ownerID, page, pageSize, aiagentgrpc.ChatSessionListFilter{})
}

func (s *NativeStore) ListChatSessionsWithFilter(ctx context.Context, ownerRole int32, ownerID int64, page, pageSize int32, filter aiagentgrpc.ChatSessionListFilter) ([]aiagentgrpc.ChatSessionRow, int64, error) {
	var total int64
	query := s.db.WithContext(ctx).Model(&aiChatSessionRecord{}).
		Where("deleted_at IS NULL")
	query = applyChatOwnerScope(query, ownerRole, ownerID)
	if sessionType := strings.TrimSpace(filter.SessionType); sessionType != "" {
		query = query.Where("session_type = ?", sessionType)
	}
	if sourceType := strings.TrimSpace(filter.SourceType); sourceType != "" {
		query = query.Where("source_type = ?", sourceType)
	}
	if filter.SourceID > 0 {
		query = query.Where("source_id = ?", filter.SourceID)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("(title LIKE ? OR source_title LIKE ? OR summary LIKE ? OR last_message_preview LIKE ?)", like, like, like, like)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []aiChatSessionRecord
	if err := query.Order("updated_at DESC, id DESC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]aiagentgrpc.ChatSessionRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapSessionRecord(ctx, row))
	}
	return result, total, nil
}

func (s *NativeStore) UpdateChatSessionTitle(ctx context.Context, ownerRole int32, ownerID, sessionID int64, title string) error {
	query := s.db.WithContext(ctx).Model(&aiChatSessionRecord{}).Where("id = ? AND deleted_at IS NULL", sessionID)
	query = applyChatOwnerScope(query, ownerRole, ownerID)
	return query.Update("title", title).Error
}

func (s *NativeStore) UpdateChatSessionContextModel(ctx context.Context, ownerRole int32, ownerID, sessionID, selectedModelID int64, usage *pb.ContextUsageInfo) error {
	updates := map[string]any{"selected_model_id": selectedModelID}
	if usage != nil {
		contextUsageJSON, err := marshalContextUsage(usage)
		if err != nil {
			return fmt.Errorf("marshal chat session context preview: %w", err)
		}
		updates["latest_context_usage_json"] = contextUsageJSON
	}
	query := s.db.WithContext(ctx).Model(&aiChatSessionRecord{}).Where("id = ? AND deleted_at IS NULL", sessionID)
	query = applyChatOwnerScope(query, ownerRole, ownerID)
	return query.Updates(updates).Error
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
	contextUsageJSON, err := marshalContextUsage(message.ContextUsage)
	if err != nil {
		return aiagentgrpc.ChatMessageRow{}, fmt.Errorf("marshal chat message context usage: %w", err)
	}
	row := aiChatHistoryRecord{
		HRID:             chatCompatibilityHRID(message.OwnerRole, message.OwnerID),
		OwnerRole:        message.OwnerRole,
		OwnerID:          message.OwnerID,
		SessionID:        message.SessionID,
		Role:             message.Role,
		Content:          message.Content,
		ProcessContent:   message.ProcessContent,
		ModelID:          message.ModelID,
		ModelName:        message.ModelName,
		ContextUsageJSON: contextUsageJSON,
		AgentSkillIDs:    nullableJSON(marshalInt64Slice(message.AgentSkillIDs)),
		AgentSkillNames:  nullableJSON(marshalStringSlice(message.AgentSkillNames)),
		CreatedAt:        now,
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"updated_at":           now,
			"last_message_preview": chatMessagePreview(message.Content),
			"message_count":        gorm.Expr("message_count + 1"),
		}
		if contextUsageJSON != nil {
			updates["latest_context_usage_json"] = contextUsageJSON
		}
		return tx.Model(&aiChatSessionRecord{}).Where("id = ?", message.SessionID).Updates(updates).Error
	}); err != nil {
		return aiagentgrpc.ChatMessageRow{}, err
	}
	return mapMessageRecord(ctx, row), nil
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
		result = append(result, mapMessageRecord(ctx, row))
	}
	return result, nil
}

// ListRecentChatMessages returns the newest messages for a session in chronological order.
func (s *NativeStore) ListRecentChatMessages(ctx context.Context, ownerRole int32, ownerID, sessionID int64, limit int32) ([]aiagentgrpc.ChatMessageRow, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	query := s.db.WithContext(ctx).Model(&aiChatHistoryRecord{})
	query = applyChatOwnerScope(query, ownerRole, ownerID)
	if sessionID > 0 {
		query = query.Where("session_id = ?", sessionID)
	}
	var rows []aiChatHistoryRecord
	if err := query.Order("created_at DESC, id DESC").Limit(int(limit)).Find(&rows).Error; err != nil {
		return nil, err
	}
	// Reverse to chronological ascending for agent message assembly.
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	result := make([]aiagentgrpc.ChatMessageRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapMessageRecord(ctx, row))
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
		result = append(result, mapToolTraceRecord(row))
	}
	return result, nil
}

func (s *NativeStore) AppendToolTrace(ctx context.Context, ownerID int64, trace aiagentgrpc.ToolTraceRow) (aiagentgrpc.ToolTraceRow, error) {
	now := time.Now()
	if !trace.CreatedAt.IsZero() {
		now = trace.CreatedAt
	}
	status := strings.TrimSpace(trace.Status)
	if status == "" {
		if strings.TrimSpace(trace.ErrorMsg) != "" {
			status = "error"
		} else {
			status = "success"
		}
	}
	row := aiToolTraceRecord{
		HRID:          ownerID,
		SessionID:     trace.SessionID,
		ToolCallID:    strings.TrimSpace(trace.ToolCallID),
		ToolName:      strings.TrimSpace(trace.ToolName),
		ArgsJSON:      strings.TrimSpace(trace.ArgsJSON),
		ResultContent: strings.TrimSpace(trace.ResultContent),
		Status:        status,
		DurationMs:    trace.DurationMs,
		ErrorMsg:      strings.TrimSpace(trace.ErrorMsg),
		CreatedAt:     now,
	}
	if trace.AgentRunID > 0 {
		id := trace.AgentRunID
		row.AgentRunID = &id
	}
	if trace.AgentRunStepID > 0 {
		id := trace.AgentRunStepID
		row.AgentRunStepID = &id
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return aiagentgrpc.ToolTraceRow{}, err
	}
	return mapToolTraceRecord(row), nil
}

func mapToolTraceRecord(row aiToolTraceRecord) aiagentgrpc.ToolTraceRow {
	out := aiagentgrpc.ToolTraceRow{
		ID:            row.ID,
		SessionID:     row.SessionID,
		ToolCallID:    row.ToolCallID,
		ToolName:      row.ToolName,
		ArgsJSON:      row.ArgsJSON,
		ResultContent: row.ResultContent,
		Status:        row.Status,
		DurationMs:    row.DurationMs,
		ErrorMsg:      row.ErrorMsg,
		CreatedAt:     row.CreatedAt,
	}
	if row.AgentRunID != nil {
		out.AgentRunID = *row.AgentRunID
	}
	if row.AgentRunStepID != nil {
		out.AgentRunStepID = *row.AgentRunStepID
	}
	return out
}

func (s *NativeStore) AppendAgentRunStep(ctx context.Context, step aiagentgrpc.AgentRunStepRow) (aiagentgrpc.AgentRunStepRow, error) {
	if step.RunID <= 0 {
		return aiagentgrpc.AgentRunStepRow{}, fmt.Errorf("run_id is required")
	}
	now := time.Now()
	if step.CreatedAt.IsZero() {
		step.CreatedAt = now
	}
	if step.UpdatedAt.IsZero() {
		step.UpdatedAt = now
	}
	if step.StartedAt.IsZero() {
		step.StartedAt = step.CreatedAt
	}
	if step.StepIndex <= 0 {
		var maxIndex sql.NullInt64
		if err := s.db.WithContext(ctx).Model(&agentRunStepRecord{}).
			Select("MAX(step_index)").
			Where("run_id = ?", step.RunID).
			Scan(&maxIndex).Error; err != nil {
			return aiagentgrpc.AgentRunStepRow{}, err
		}
		if maxIndex.Valid {
			step.StepIndex = int32(maxIndex.Int64) + 1
		} else {
			step.StepIndex = 1
		}
	}
	status := strings.TrimSpace(step.Status)
	if status == "" {
		status = "running"
	}
	row := agentRunStepRecord{
		RunID:            step.RunID,
		StepIndex:        step.StepIndex,
		StepType:         defaultString(strings.TrimSpace(step.StepType), "tool"),
		CapabilitySource: strings.TrimSpace(step.CapabilitySource),
		CapabilityKey:    strings.TrimSpace(step.CapabilityKey),
		ToolName:         strings.TrimSpace(step.ToolName),
		InputJSON:        nullableJSON(step.InputJSON),
		OutputJSON:       nullableJSON(step.OutputJSON),
		Status:           status,
		DurationMs:       step.DurationMs,
		ErrorMsg:         strings.TrimSpace(step.ErrorMsg),
		StartedAt:        step.StartedAt,
		CompletedAt:      step.CompletedAt,
		CreatedAt:        step.CreatedAt,
		UpdatedAt:        step.UpdatedAt,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return aiagentgrpc.AgentRunStepRow{}, err
	}
	return mapAgentRunStepRecord(row), nil
}

func (s *NativeStore) ListAgentRunSteps(ctx context.Context, runID int64) ([]aiagentgrpc.AgentRunStepRow, error) {
	if runID <= 0 {
		return nil, nil
	}
	var rows []agentRunStepRecord
	if err := s.db.WithContext(ctx).Where("run_id = ?", runID).Order("step_index ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]aiagentgrpc.AgentRunStepRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapAgentRunStepRecord(row))
	}
	return result, nil
}

func mapAgentRunStepRecord(row agentRunStepRecord) aiagentgrpc.AgentRunStepRow {
	return aiagentgrpc.AgentRunStepRow{
		ID:               row.ID,
		RunID:            row.RunID,
		StepIndex:        row.StepIndex,
		StepType:         row.StepType,
		CapabilitySource: row.CapabilitySource,
		CapabilityKey:    row.CapabilityKey,
		ToolName:         row.ToolName,
		InputJSON:        stringFromPtr(row.InputJSON),
		OutputJSON:       stringFromPtr(row.OutputJSON),
		Status:           row.Status,
		DurationMs:       row.DurationMs,
		ErrorMsg:         row.ErrorMsg,
		StartedAt:        row.StartedAt,
		CompletedAt:      row.CompletedAt,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func stringFromPtr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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
		TenantID:        positiveInt64Pointer(run.TenantID),
		SessionID:       run.SessionID,
		HRID:            run.OwnerID,
		ClientRequestID: run.ClientRequestID,
		Status:          "queued",
		PlanJSON:        nullableJSON(run.PlanJSON),
		OptionContext:   nullableJSON(run.OptionContextJSON),
		ModelID:         run.ModelID,
		ModelName:       run.ModelName,
		AgentType:       defaultString(run.AgentType, "hr"),
		AgentID:         run.AgentID,
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
	if status == "canceled" {
		updates["canceled_at"] = now
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

func (s *NativeStore) UpdateAgentRunPlan(ctx context.Context, ownerID, runID int64, planJSON, optionContextJSON string) (aiagentgrpc.AgentRunRow, bool, error) {
	updates := map[string]any{
		"plan_json":  nullableJSON(planJSON),
		"updated_at": time.Now(),
	}
	if strings.TrimSpace(optionContextJSON) != "" {
		updates["option_context_json"] = nullableJSON(optionContextJSON)
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
	if status == "canceled" {
		updates["canceled_at"] = now
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

func (s *NativeStore) UpdateAgentRunRuntimeGovernance(ctx context.Context, ownerID, runID int64, model aiagentgrpc.RuntimeModelInfo) error {
	return s.db.WithContext(ctx).Model(&agentRunRecord{}).Where("id = ? AND hr_id = ?", runID, ownerID).Updates(map[string]any{
		"requested_model_id":       model.RequestedModelID,
		"effective_model_id":       model.ID,
		"model_fallback_reason":    nullableSQLString(model.FallbackReason),
		"capability_version_id":    model.CapabilityVersionID,
		"capability_snapshot_hash": nullableSQLString(model.CapabilitySnapshotHash),
		"updated_at":               time.Now(),
	}).Error
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
		items = append(items, s.llmProviderToPB(row))
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
		items = append(items, llmModelToPB(row))
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

func (s *NativeStore) GetRuntimePromptTemplateByID(ctx context.Context, id int64) (*pb.PromptTemplateInfo, bool, error) {
	var row promptTemplateRecord
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &pb.PromptTemplateInfo{
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
	}, true, nil
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
		info, err := s.agentConfigInfo(ctx, row)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, info)
	}
	return items, total, nil
}

func (s *NativeStore) GetRuntimeAgentConfigByID(ctx context.Context, id int64) (*pb.AgentConfigInfo, bool, error) {
	var row agentConfigListRow
	query := s.agentConfigSelect(ctx).Where("a.id = ?", id)
	if err := query.First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	agent, err := s.agentConfigInfo(ctx, row)
	if err != nil {
		return nil, false, err
	}
	return agent, true, nil
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

func (s *NativeStore) GetRecruitingApplicationByID(ctx context.Context, applicationID int64) (aiagentgrpc.RecruitingApplicationContext, bool, error) {
	var row recruitingApplicationReadRow
	err := recruitingApplicationReadQuery(s.db.WithContext(ctx)).
		Where("a.id = ?", applicationID).
		Limit(1).
		Scan(&row).Error
	if err != nil {
		return aiagentgrpc.RecruitingApplicationContext{}, false, err
	}
	if row.ApplicationID == 0 {
		return aiagentgrpc.RecruitingApplicationContext{}, false, nil
	}
	return mapRecruitingApplicationReadRow(row), true, nil
}

func (s *NativeStore) GetLatestRecruitingApplicationByResumeID(ctx context.Context, resumeID int64) (aiagentgrpc.RecruitingApplicationContext, bool, error) {
	var row recruitingApplicationReadRow
	err := recruitingApplicationReadQuery(s.db.WithContext(ctx)).
		Where("a.resume_id = ?", resumeID).
		Order("a.applied_at DESC, a.id DESC").
		Limit(1).
		Scan(&row).Error
	if err != nil {
		return aiagentgrpc.RecruitingApplicationContext{}, false, err
	}
	if row.ApplicationID == 0 {
		return aiagentgrpc.RecruitingApplicationContext{}, false, nil
	}
	return mapRecruitingApplicationReadRow(row), true, nil
}

func (s *NativeStore) ListCurrentRecruitingApplicationsByJobID(ctx context.Context, jobID int64) ([]aiagentgrpc.RecruitingApplicationContext, error) {
	var rows []recruitingApplicationReadRow
	if err := recruitingApplicationReadQuery(s.db.WithContext(ctx)).
		Where("a.job_id = ? AND a.is_current = ?", jobID, 1).
		Order("a.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]aiagentgrpc.RecruitingApplicationContext, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapRecruitingApplicationReadRow(row))
	}
	return items, nil
}

func (s *NativeStore) LoadCandidateRuntimeContext(ctx context.Context, userID int64, limit int32) (aiagentgrpc.CandidateRuntimeContext, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	applications, err := s.listCandidateApplications(ctx, userID, limit)
	if err != nil {
		return aiagentgrpc.CandidateRuntimeContext{}, err
	}
	resume, err := s.getCandidateResumeContext(ctx, userID)
	if err != nil {
		return aiagentgrpc.CandidateRuntimeContext{}, err
	}
	jobs, err := s.listCandidateJobs(ctx, userID, limit)
	if err != nil {
		return aiagentgrpc.CandidateRuntimeContext{}, err
	}
	interviews, err := s.listCandidateInterviews(ctx, userID, limit)
	if err != nil {
		return aiagentgrpc.CandidateRuntimeContext{}, err
	}
	offers, err := s.listCandidateOffers(ctx, userID, limit)
	if err != nil {
		return aiagentgrpc.CandidateRuntimeContext{}, err
	}
	return aiagentgrpc.CandidateRuntimeContext{
		Applications: applications,
		Resume:       resume,
		Jobs:         jobs,
		Interviews:   interviews,
		Offers:       offers,
	}, nil
}

func (s *NativeStore) RecordCandidateUsageAudit(ctx context.Context, row aiagentgrpc.CandidateUsageAuditRow) (int64, error) {
	return s.RecordUsageAudit(ctx, candidateUsageRowFromLegacy(row))
}

func (s *NativeStore) RecordUsageAudit(ctx context.Context, row aiagentgrpc.UsageAuditRow) (int64, error) {
	row = normalizeUsageAuditRow(row)
	tokenCount := row.TokenUsageTotal
	if tokenCount <= 0 {
		tokenCount = row.EstimatedTokens
	}
	if tokenCount <= 0 {
		tokenCount = estimateUsageTokens(row.RequestChars, row.ResponseChars)
	}
	usage := thirdPartyUsageLogRecord{
		UserID:          row.UserID,
		Role:            row.Role,
		ServiceType:     row.ServiceType,
		Endpoint:        row.Endpoint,
		Provider:        row.Provider,
		Model:           row.Model,
		RequestChars:    row.RequestChars,
		ResponseChars:   row.ResponseChars,
		EstimatedTokens: tokenCount,
		Status:          row.Status,
		ErrorCode:       row.ErrorCode,
		CostMs:          row.CostMs,
		RequestID:       row.RequestID,
		IP:              row.IP,
		CreatedAt:       time.Now(),
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&usage).Error; err != nil {
			return err
		}
		authCtx := aiUsageAuthContextRecord{
			UsageLogID:    usage.ID,
			ActorUserID:   row.UserID,
			AccountType:   row.AccountType,
			RoleKeys:      strings.Join(row.RoleKeys, ","),
			PermissionKey: row.PermissionKey,
			ScopeKeys:     strings.Join(row.ScopeKeys, ","),
			ResourceType:  row.ResourceType,
			ResourceID:    row.ResourceID,
			Decision:      "allowed",
			RequestID:     row.RequestID,
			CreatedAt:     time.Now(),
		}
		return tx.Create(&authCtx).Error
	})
	if err != nil {
		return 0, err
	}
	return usage.ID, nil
}

func candidateUsageRowFromLegacy(row aiagentgrpc.CandidateUsageAuditRow) aiagentgrpc.UsageAuditRow {
	return aiagentgrpc.UsageAuditRow{
		UserID:          row.UserID,
		Role:            1,
		AccountType:     "candidate",
		ServiceType:     row.ServiceType,
		Endpoint:        row.Endpoint,
		Provider:        row.Provider,
		Model:           row.Model,
		RequestChars:    row.RequestChars,
		ResponseChars:   row.ResponseChars,
		EstimatedTokens: row.EstimatedTokens,
		Status:          row.Status,
		ErrorCode:       row.ErrorCode,
		CostMs:          row.CostMs,
		RequestID:       row.RequestID,
		IP:              row.IP,
		RoleKeys:        append([]string(nil), row.RoleKeys...),
		PermissionKey:   row.PermissionKey,
		ScopeKeys:       append([]string(nil), row.ScopeKeys...),
		ResourceType:    "ai",
		ResourceID:      0,
	}
}

func normalizeUsageAuditRow(row aiagentgrpc.UsageAuditRow) aiagentgrpc.UsageAuditRow {
	if row.ServiceType == "" {
		row.ServiceType = "ai_chat"
	}
	if row.Status == "" {
		row.Status = "ok"
	}
	if row.AccountType == "" {
		if row.Role == 2 {
			row.AccountType = "staff"
		} else {
			row.AccountType = "candidate"
			if row.Role == 0 {
				row.Role = 1
			}
		}
	}
	if row.Role == 0 {
		if row.AccountType == "staff" {
			row.Role = 2
		} else {
			row.Role = 1
		}
	}
	if row.Endpoint == "" {
		if row.AccountType == "staff" {
			row.Endpoint = "/hr/ai/chat"
		} else {
			row.Endpoint = "/candidate/ai/chat/stream"
		}
	}
	if row.Provider == "" {
		row.Provider = "openai_compatible"
	}
	if row.PermissionKey == "" {
		if row.AccountType == "staff" {
			row.PermissionKey = "ai.hr.use"
		} else {
			row.PermissionKey = "ai.candidate.use"
		}
	}
	if len(row.RoleKeys) == 0 {
		if row.AccountType == "staff" {
			row.RoleKeys = []string{"staff"}
		} else {
			row.RoleKeys = []string{"candidate"}
		}
	}
	if len(row.ScopeKeys) == 0 && row.AccountType != "staff" {
		row.ScopeKeys = []string{"self"}
	}
	if row.ResourceType == "" {
		row.ResourceType = "ai"
	}
	return row
}

func estimateUsageTokens(requestChars, responseChars int) int {
	total := requestChars + responseChars
	if total <= 0 {
		return 0
	}
	return (total + 3) / 4
}

func (s *NativeStore) GetRecruitingResumeProfileByID(ctx context.Context, profileID uint64) (aiagentgrpc.RecruitingResumeProfileRow, bool, error) {
	var row recruitingResumeProfileRecord
	if err := s.db.WithContext(ctx).Where("id = ?", profileID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return aiagentgrpc.RecruitingResumeProfileRow{}, false, nil
		}
		return aiagentgrpc.RecruitingResumeProfileRow{}, false, err
	}
	return mapRecruitingResumeProfileRecord(row), true, nil
}

func (s *NativeStore) GetCurrentRecruitingResumeProfileByResumeID(ctx context.Context, resumeID int64) (aiagentgrpc.RecruitingResumeProfileRow, bool, error) {
	var row recruitingResumeProfileRecord
	if err := s.db.WithContext(ctx).
		Where("resume_id = ? AND is_current = ?", resumeID, 1).
		Order("version DESC, id DESC").
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return aiagentgrpc.RecruitingResumeProfileRow{}, false, nil
		}
		return aiagentgrpc.RecruitingResumeProfileRow{}, false, err
	}
	return mapRecruitingResumeProfileRecord(row), true, nil
}

func (s *NativeStore) GetRecruitingResumeProfileSnapshot(ctx context.Context, profileID uint64) (aiagentgrpc.RecruitingResumeProfileSnapshot, bool, error) {
	var profile recruitingResumeProfileRecord
	if err := s.db.WithContext(ctx).Where("id = ?", profileID).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return aiagentgrpc.RecruitingResumeProfileSnapshot{}, false, nil
		}
		return aiagentgrpc.RecruitingResumeProfileSnapshot{}, false, err
	}
	var parseRun recruitingResumeParseRunRecord
	if err := s.db.WithContext(ctx).Where("id = ?", profile.ParseRunID).First(&parseRun).Error; err != nil {
		return aiagentgrpc.RecruitingResumeProfileSnapshot{}, false, err
	}
	var educations []recruitingResumeEducationRecord
	if err := s.db.WithContext(ctx).Where("resume_profile_id = ?", profile.ID).Order("sort_order ASC, id ASC").Find(&educations).Error; err != nil {
		return aiagentgrpc.RecruitingResumeProfileSnapshot{}, false, err
	}
	var experiences []recruitingResumeExperienceRecord
	if err := s.db.WithContext(ctx).Where("resume_profile_id = ?", profile.ID).Order("sort_order ASC, id ASC").Find(&experiences).Error; err != nil {
		return aiagentgrpc.RecruitingResumeProfileSnapshot{}, false, err
	}
	var projects []recruitingResumeProjectRecord
	if err := s.db.WithContext(ctx).Where("resume_profile_id = ?", profile.ID).Order("sort_order ASC, id ASC").Find(&projects).Error; err != nil {
		return aiagentgrpc.RecruitingResumeProfileSnapshot{}, false, err
	}
	var skills []recruitingResumeSkillRecord
	if err := s.db.WithContext(ctx).Where("resume_profile_id = ?", profile.ID).Order("sort_order ASC, id ASC").Find(&skills).Error; err != nil {
		return aiagentgrpc.RecruitingResumeProfileSnapshot{}, false, err
	}
	snapshot := aiagentgrpc.RecruitingResumeProfileSnapshot{
		ParseRun:    mapRecruitingResumeParseRunRecord(parseRun),
		Profile:     mapRecruitingResumeProfileRecord(profile),
		Educations:  make([]aiagentgrpc.RecruitingResumeEducationRow, 0, len(educations)),
		Experiences: make([]aiagentgrpc.RecruitingResumeExperienceRow, 0, len(experiences)),
		Projects:    make([]aiagentgrpc.RecruitingResumeProjectRow, 0, len(projects)),
		Skills:      make([]aiagentgrpc.RecruitingResumeSkillRow, 0, len(skills)),
	}
	for _, row := range educations {
		snapshot.Educations = append(snapshot.Educations, mapRecruitingResumeEducationRecord(row))
	}
	for _, row := range experiences {
		snapshot.Experiences = append(snapshot.Experiences, mapRecruitingResumeExperienceRecord(row))
	}
	for _, row := range projects {
		snapshot.Projects = append(snapshot.Projects, mapRecruitingResumeProjectRecord(row))
	}
	for _, row := range skills {
		snapshot.Skills = append(snapshot.Skills, mapRecruitingResumeSkillRecord(row))
	}
	return snapshot, true, nil
}

func (s *NativeStore) GetRecruitingResumeSource(ctx context.Context, resumeID int64) (aiagentgrpc.RecruitingResumeSource, bool, error) {
	var row recruitingResumeSourceRow
	if err := s.db.WithContext(ctx).Table("resumes").
		Select("id AS resume_id, user_id, file_name, parsed_text").
		Where("id = ?", resumeID).
		Limit(1).
		Scan(&row).Error; err != nil {
		return aiagentgrpc.RecruitingResumeSource{}, false, err
	}
	if row.ResumeID == 0 {
		return aiagentgrpc.RecruitingResumeSource{}, false, nil
	}
	return aiagentgrpc.RecruitingResumeSource{
		ResumeID:   row.ResumeID,
		UserID:     row.UserID,
		FileName:   row.FileName,
		ParsedText: nullString(row.ParsedText),
	}, true, nil
}

func (s *NativeStore) SaveRecruitingResumeProfileDraft(ctx context.Context, draft aiagentgrpc.RecruitingResumeProfileDraft) (aiagentgrpc.RecruitingResumeProfileSnapshot, error) {
	now := time.Now()
	completedAt := now
	var profileID uint64
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maxVersion sql.NullInt32
		if err := tx.Model(&recruitingResumeProfileRecord{}).
			Where("resume_id = ?", draft.ResumeID).
			Select("MAX(version)").
			Scan(&maxVersion).Error; err != nil {
			return err
		}
		version := int32(1)
		if maxVersion.Valid {
			version = maxVersion.Int32 + 1
		}
		if err := tx.Model(&recruitingResumeProfileRecord{}).
			Where("resume_id = ? AND is_current = ?", draft.ResumeID, 1).
			Update("is_current", 0).Error; err != nil {
			return err
		}
		parseRun := recruitingResumeParseRunRecord{
			ResumeID:               draft.ResumeID,
			UserID:                 draft.UserID,
			RequestedModelID:       nullInt64From(draft.RequestedModelID),
			EffectiveModelID:       nullInt64From(draft.EffectiveModelID),
			ModelFallbackReason:    nullableSQLString(draft.ModelFallbackReason),
			CapabilityVersionID:    nullInt64From(draft.CapabilityVersionID),
			CapabilitySnapshotHash: nullableSQLString(draft.CapabilitySnapshotHash),
			Status:                 "succeeded",
			ParserVersion:          nullableSQLString(draft.ParserVersion),
			InputHash:              nullableSQLString(draft.InputHash),
			StartedAt:              now,
			CompletedAt:            &completedAt,
			CreatedAt:              now,
			UpdatedAt:              now,
		}
		if err := tx.Create(&parseRun).Error; err != nil {
			return err
		}
		profile := recruitingResumeProfileRecord{
			ResumeID:             draft.ResumeID,
			UserID:               draft.UserID,
			ParseRunID:           parseRun.ID,
			Version:              version,
			IsCurrent:            1,
			FullName:             nullableSQLString(draft.FullName),
			Email:                nullableSQLString(draft.Email),
			Phone:                nullableSQLString(draft.Phone),
			Location:             nullableSQLString(draft.Location),
			Headline:             nullableSQLString(draft.Headline),
			Summary:              nullableSQLString(draft.Summary),
			TotalExperienceYears: sql.NullFloat64{Float64: draft.TotalExperienceYears, Valid: true},
			HighestDegree:        nullableSQLString(draft.HighestDegree),
			RawJSON:              nullableSQLString(draft.RawJSON),
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		if err := tx.Create(&profile).Error; err != nil {
			return err
		}
		profileID = profile.ID
		for _, row := range draft.Educations {
			record := recruitingResumeEducationRecord{
				ResumeProfileID: profile.ID,
				School:          strings.TrimSpace(row.School),
				Degree:          nullableSQLString(row.Degree),
				Major:           nullableSQLString(row.Major),
				StartDate:       row.StartDate,
				EndDate:         row.EndDate,
				Description:     nullableSQLString(row.Description),
				SortOrder:       row.SortOrder,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			if record.School == "" {
				continue
			}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
		}
		for _, row := range draft.Experiences {
			record := recruitingResumeExperienceRecord{
				ResumeProfileID:  profile.ID,
				Company:          strings.TrimSpace(row.Company),
				Title:            nullableSQLString(row.Title),
				Location:         nullableSQLString(row.Location),
				StartDate:        row.StartDate,
				EndDate:          row.EndDate,
				IsCurrent:        row.IsCurrent,
				Description:      nullableSQLString(row.Description),
				AchievementsJSON: nullableSQLString(row.AchievementsJSON),
				SortOrder:        row.SortOrder,
				CreatedAt:        now,
				UpdatedAt:        now,
			}
			if record.Company == "" {
				continue
			}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
		}
		for _, row := range draft.Projects {
			record := recruitingResumeProjectRecord{
				ResumeProfileID:  profile.ID,
				Name:             strings.TrimSpace(row.Name),
				Role:             nullableSQLString(row.Role),
				StartDate:        row.StartDate,
				EndDate:          row.EndDate,
				Description:      nullableSQLString(row.Description),
				TechnologiesJSON: nullableSQLString(row.TechnologiesJSON),
				HighlightsJSON:   nullableSQLString(row.HighlightsJSON),
				SortOrder:        row.SortOrder,
				CreatedAt:        now,
				UpdatedAt:        now,
			}
			if record.Name == "" {
				continue
			}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
		}
		for _, row := range draft.Skills {
			record := recruitingResumeSkillRecord{
				ResumeProfileID: profile.ID,
				Name:            strings.TrimSpace(row.Name),
				Category:        nullableSQLString(row.Category),
				Level:           nullableSQLString(row.Level),
				Years:           sql.NullFloat64{Float64: row.Years, Valid: true},
				Evidence:        nullableSQLString(row.Evidence),
				SortOrder:       row.SortOrder,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			if record.Name == "" {
				continue
			}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return aiagentgrpc.RecruitingResumeProfileSnapshot{}, err
	}
	snapshot, found, err := s.GetRecruitingResumeProfileSnapshot(ctx, profileID)
	if err != nil {
		return aiagentgrpc.RecruitingResumeProfileSnapshot{}, err
	}
	if !found {
		return aiagentgrpc.RecruitingResumeProfileSnapshot{}, fmt.Errorf("saved resume profile not found")
	}
	return snapshot, nil
}

func (s *NativeStore) GetRecruitingCandidateMatchEvaluationSnapshot(ctx context.Context, evaluationID uint64) (aiagentgrpc.RecruitingCandidateMatchSnapshot, bool, error) {
	var evaluation recruitingCandidateMatchEvaluationRecord
	if err := s.db.WithContext(ctx).Where("id = ?", evaluationID).First(&evaluation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return aiagentgrpc.RecruitingCandidateMatchSnapshot{}, false, nil
		}
		return aiagentgrpc.RecruitingCandidateMatchSnapshot{}, false, err
	}
	return s.recruitingCandidateMatchSnapshot(ctx, evaluation)
}

func (s *NativeStore) GetRecruitingCandidateMatchEvaluationSnapshotByApplicationVersion(ctx context.Context, applicationID int64, version int32) (aiagentgrpc.RecruitingCandidateMatchSnapshot, bool, error) {
	var evaluation recruitingCandidateMatchEvaluationRecord
	if err := s.db.WithContext(ctx).Where("application_id = ? AND evaluation_version = ?", applicationID, version).First(&evaluation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return aiagentgrpc.RecruitingCandidateMatchSnapshot{}, false, nil
		}
		return aiagentgrpc.RecruitingCandidateMatchSnapshot{}, false, err
	}
	return s.recruitingCandidateMatchSnapshot(ctx, evaluation)
}

func (s *NativeStore) GetRecruitingCandidateMatchEvaluationSnapshotByApplicationAgentRunID(ctx context.Context, applicationID int64, agentRunID uint64) (aiagentgrpc.RecruitingCandidateMatchSnapshot, bool, error) {
	var evaluation recruitingCandidateMatchEvaluationRecord
	if err := s.db.WithContext(ctx).
		Where("application_id = ? AND agent_run_id = ?", applicationID, agentRunID).
		Order("evaluation_version DESC, id DESC").
		First(&evaluation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return aiagentgrpc.RecruitingCandidateMatchSnapshot{}, false, nil
		}
		return aiagentgrpc.RecruitingCandidateMatchSnapshot{}, false, err
	}
	return s.recruitingCandidateMatchSnapshot(ctx, evaluation)
}

func (s *NativeStore) GetLatestRecruitingCandidateMatchEvaluationSnapshotByApplicationID(ctx context.Context, applicationID int64) (aiagentgrpc.RecruitingCandidateMatchSnapshot, bool, error) {
	var evaluation recruitingCandidateMatchEvaluationRecord
	if err := s.db.WithContext(ctx).
		Where("application_id = ? AND is_latest = ?", applicationID, 1).
		Order("evaluation_version DESC, id DESC").
		First(&evaluation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return aiagentgrpc.RecruitingCandidateMatchSnapshot{}, false, nil
		}
		return aiagentgrpc.RecruitingCandidateMatchSnapshot{}, false, err
	}
	return s.recruitingCandidateMatchSnapshot(ctx, evaluation)
}

func (s *NativeStore) ListLatestRecruitingCandidateMatchEvaluationsByApplicationIDs(ctx context.Context, applicationIDs []int64) ([]aiagentgrpc.RecruitingCandidateMatchEvaluationRow, error) {
	if len(applicationIDs) == 0 {
		return nil, nil
	}
	var rows []recruitingCandidateMatchEvaluationRecord
	if err := s.db.WithContext(ctx).
		Where("application_id IN ? AND is_latest = ?", applicationIDs, 1).
		Order("application_id ASC, evaluation_version DESC, id DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]aiagentgrpc.RecruitingCandidateMatchEvaluationRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapRecruitingCandidateMatchEvaluationRecord(row))
	}
	return items, nil
}

func (s *NativeStore) recruitingCandidateMatchSnapshot(ctx context.Context, evaluation recruitingCandidateMatchEvaluationRecord) (aiagentgrpc.RecruitingCandidateMatchSnapshot, bool, error) {
	var evidence []recruitingCandidateMatchEvidenceRecord
	if err := s.db.WithContext(ctx).Where("evaluation_id = ?", evaluation.ID).Order("id ASC").Find(&evidence).Error; err != nil {
		return aiagentgrpc.RecruitingCandidateMatchSnapshot{}, false, err
	}
	snapshot := aiagentgrpc.RecruitingCandidateMatchSnapshot{
		Evaluation: mapRecruitingCandidateMatchEvaluationRecord(evaluation),
		Evidence:   make([]aiagentgrpc.RecruitingCandidateMatchEvidenceRow, 0, len(evidence)),
	}
	for _, row := range evidence {
		snapshot.Evidence = append(snapshot.Evidence, mapRecruitingCandidateMatchEvidenceRecord(row))
	}
	return snapshot, true, nil
}

func (s *NativeStore) GetRecruitingMatchSource(ctx context.Context, applicationID int64) (aiagentgrpc.RecruitingMatchSource, bool, error) {
	application, found, err := s.GetRecruitingApplicationByID(ctx, applicationID)
	if err != nil || !found {
		return aiagentgrpc.RecruitingMatchSource{}, found, err
	}
	profile, found, err := s.GetCurrentRecruitingResumeProfileByResumeID(ctx, application.ResumeID)
	if err != nil || !found {
		return aiagentgrpc.RecruitingMatchSource{}, found, err
	}
	snapshot, found, err := s.GetRecruitingResumeProfileSnapshot(ctx, profile.ID)
	if err != nil || !found {
		return aiagentgrpc.RecruitingMatchSource{}, found, err
	}
	var job recruitingJobContextRow
	if err := s.db.WithContext(ctx).Table("jobs").
		Select("id AS job_id, title, department, location, description, requirements").
		Where("id = ?", application.JobID).
		Limit(1).
		Scan(&job).Error; err != nil {
		return aiagentgrpc.RecruitingMatchSource{}, false, err
	}
	if job.JobID == 0 {
		return aiagentgrpc.RecruitingMatchSource{}, false, nil
	}
	var candidateProfile recruitingCandidateProfileReadRow
	if err := s.db.WithContext(ctx).Table("candidate_profiles").
		Select("id, real_name, phone, education, school, work_experience, skills, is_complete").
		Where("user_id = ?", application.CandidateUserID).
		Limit(1).
		Scan(&candidateProfile).Error; err != nil {
		return aiagentgrpc.RecruitingMatchSource{}, false, err
	}
	var resume struct {
		ID         int64          `gorm:"column:id"`
		ParsedText sql.NullString `gorm:"column:parsed_text"`
	}
	if err := s.db.WithContext(ctx).Table("resumes").
		Select("id, parsed_text").
		Where("id = ?", application.ResumeID).
		Limit(1).
		Scan(&resume).Error; err != nil {
		return aiagentgrpc.RecruitingMatchSource{}, false, err
	}
	if resume.ID == 0 {
		return aiagentgrpc.RecruitingMatchSource{}, false, nil
	}
	var mappedCandidateProfile *aiagentgrpc.RecruitingCandidateProfileRow
	if candidateProfile.ID > 0 {
		mappedCandidateProfile = &aiagentgrpc.RecruitingCandidateProfileRow{
			ID: candidateProfile.ID, RealName: nullString(candidateProfile.RealName), Phone: nullString(candidateProfile.Phone),
			Education: nullString(candidateProfile.Education), School: nullString(candidateProfile.School),
			WorkExperience: nullString(candidateProfile.WorkExperience), Skills: nullString(candidateProfile.Skills), IsComplete: candidateProfile.IsComplete,
		}
	}
	return aiagentgrpc.RecruitingMatchSource{
		Application: application, CandidateProfile: mappedCandidateProfile,
		ApplicationEducation: nullString(candidateProfile.Education), ResumeParsedText: nullString(resume.ParsedText),
		Job: aiagentgrpc.RecruitingJobContext{
			JobID:        job.JobID,
			Title:        job.Title,
			Department:   nullString(job.Department),
			Location:     nullString(job.Location),
			Description:  nullString(job.Description),
			Requirements: nullString(job.Requirements),
		},
		Profile: snapshot,
	}, true, nil
}

func (s *NativeStore) SaveRecruitingCandidateMatchDraft(ctx context.Context, draft aiagentgrpc.RecruitingCandidateMatchDraft) (aiagentgrpc.RecruitingCandidateMatchSnapshot, error) {
	now := time.Now()
	var evaluationID uint64
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maxVersion sql.NullInt32
		if err := tx.Model(&recruitingCandidateMatchEvaluationRecord{}).
			Where("application_id = ?", draft.ApplicationID).
			Select("MAX(evaluation_version)").
			Scan(&maxVersion).Error; err != nil {
			return err
		}
		version := int32(1)
		if maxVersion.Valid {
			version = maxVersion.Int32 + 1
		}
		if err := tx.Model(&recruitingCandidateMatchEvaluationRecord{}).
			Where("application_id = ? AND is_latest = ?", draft.ApplicationID, 1).
			Update("is_latest", 0).Error; err != nil {
			return err
		}
		evaluation := recruitingCandidateMatchEvaluationRecord{
			ApplicationID:          draft.ApplicationID,
			JobID:                  draft.JobID,
			CandidateUserID:        draft.CandidateUserID,
			ResumeProfileID:        draft.ResumeProfileID,
			AgentRunID:             draft.AgentRunID,
			RequestedModelID:       nullInt64From(draft.RequestedModelID),
			EffectiveModelID:       nullInt64From(draft.EffectiveModelID),
			ModelFallbackReason:    nullableSQLString(draft.ModelFallbackReason),
			CapabilityVersionID:    nullInt64From(draft.CapabilityVersionID),
			CapabilitySnapshotHash: nullableSQLString(draft.CapabilitySnapshotHash),
			EvaluationVersion:      version,
			IsLatest:               1,
			OverallScore:           sql.NullFloat64{Float64: draft.OverallScore, Valid: true},
			Recommendation:         nullableSQLString(draft.Recommendation),
			Summary:                nullableSQLString(draft.Summary),
			StrengthsJSON:          nullableSQLString(draft.StrengthsJSON),
			RisksJSON:              nullableSQLString(draft.RisksJSON),
			ScoreBreakdownJSON:     nullableSQLString(draft.ScoreBreakdownJSON),
			ModelName:              nullableSQLString(draft.ModelName),
			EvaluatedAt:            now,
			CreatedAt:              now,
			UpdatedAt:              now,
		}
		if err := tx.Create(&evaluation).Error; err != nil {
			return err
		}
		evaluationID = evaluation.ID
		for _, row := range draft.Evidence {
			record := recruitingCandidateMatchEvidenceRecord{
				EvaluationID: evaluation.ID,
				EvidenceType: strings.TrimSpace(row.EvidenceType),
				Dimension:    nullableSQLString(row.Dimension),
				SourceTable:  nullableSQLString(row.SourceTable),
				SourceID:     row.SourceID,
				Snippet:      nullableSQLString(row.Snippet),
				Weight:       sql.NullFloat64{Float64: row.Weight, Valid: true},
				ScoreImpact:  sql.NullFloat64{Float64: row.ScoreImpact, Valid: true},
				MetadataJSON: nullableSQLString(row.MetadataJSON),
				CreatedAt:    now,
			}
			if record.EvidenceType == "" {
				record.EvidenceType = "profile"
			}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return aiagentgrpc.RecruitingCandidateMatchSnapshot{}, err
	}
	snapshot, found, err := s.GetRecruitingCandidateMatchEvaluationSnapshot(ctx, evaluationID)
	if err != nil {
		return aiagentgrpc.RecruitingCandidateMatchSnapshot{}, err
	}
	if !found {
		return aiagentgrpc.RecruitingCandidateMatchSnapshot{}, fmt.Errorf("saved candidate match evaluation not found")
	}
	return snapshot, nil
}

type recruitingApplicationReadRow struct {
	ApplicationID   int64  `gorm:"column:application_id"`
	JobID           int64  `gorm:"column:job_id"`
	CandidateUserID int64  `gorm:"column:candidate_user_id"`
	CandidateName   string `gorm:"column:candidate_name"`
	ResumeID        int64  `gorm:"column:resume_id"`
	IsCurrent       int32  `gorm:"column:is_current"`
}

type recruitingCandidateProfileReadRow struct {
	ID             uint64         `gorm:"column:id"`
	RealName       sql.NullString `gorm:"column:real_name"`
	Phone          sql.NullString `gorm:"column:phone"`
	Education      sql.NullString `gorm:"column:education"`
	School         sql.NullString `gorm:"column:school"`
	WorkExperience sql.NullString `gorm:"column:work_experience"`
	Skills         sql.NullString `gorm:"column:skills"`
	IsComplete     int32          `gorm:"column:is_complete"`
}

type candidateApplicationReadRow struct {
	ApplicationID int64     `gorm:"column:application_id"`
	JobID         int64     `gorm:"column:job_id"`
	JobTitle      string    `gorm:"column:job_title"`
	Status        int32     `gorm:"column:status"`
	StatusKey     string    `gorm:"column:status_key"`
	RoundNo       int32     `gorm:"column:round_no"`
	AppliedAt     time.Time `gorm:"column:applied_at"`
}

type candidateResumeReadRow struct {
	ResumeID   int64          `gorm:"column:resume_id"`
	FileName   string         `gorm:"column:file_name"`
	ParsedText sql.NullString `gorm:"column:parsed_text"`
}

type recruitingResumeSourceRow struct {
	ResumeID   int64          `gorm:"column:resume_id"`
	UserID     int64          `gorm:"column:user_id"`
	FileName   string         `gorm:"column:file_name"`
	ParsedText sql.NullString `gorm:"column:parsed_text"`
}

type recruitingJobContextRow struct {
	JobID        int64          `gorm:"column:job_id"`
	Title        string         `gorm:"column:title"`
	Department   sql.NullString `gorm:"column:department"`
	Location     sql.NullString `gorm:"column:location"`
	Description  sql.NullString `gorm:"column:description"`
	Requirements sql.NullString `gorm:"column:requirements"`
}

type candidateJobReadRow struct {
	JobID       int64          `gorm:"column:job_id"`
	Title       string         `gorm:"column:title"`
	Department  sql.NullString `gorm:"column:department"`
	Location    sql.NullString `gorm:"column:location"`
	SalaryRange sql.NullString `gorm:"column:salary_range"`
	Status      int32          `gorm:"column:status"`
	HasApplied  bool           `gorm:"column:has_applied"`
}

type candidateInterviewReadRow struct {
	InterviewID   int64          `gorm:"column:interview_id"`
	ApplicationID int64          `gorm:"column:application_id"`
	JobTitle      string         `gorm:"column:job_title"`
	RoundNo       int32          `gorm:"column:round_no"`
	Title         sql.NullString `gorm:"column:title"`
	Mode          sql.NullString `gorm:"column:mode"`
	ScheduledAt   *time.Time     `gorm:"column:scheduled_at"`
	Status        string         `gorm:"column:status"`
	CandidateNote sql.NullString `gorm:"column:candidate_note"`
}

type candidateOfferReadRow struct {
	OfferID       int64          `gorm:"column:offer_id"`
	ApplicationID int64          `gorm:"column:application_id"`
	JobID         int64          `gorm:"column:job_id"`
	Title         string         `gorm:"column:title"`
	Status        string         `gorm:"column:status"`
	SalaryRange   sql.NullString `gorm:"column:salary_range"`
	WorkLocation  sql.NullString `gorm:"column:work_location"`
	StartDate     sql.NullString `gorm:"column:start_date"`
	ExpiresAt     *time.Time     `gorm:"column:expires_at"`
}

type recruitingResumeParseRunRecord struct {
	ID                     uint64         `gorm:"primaryKey"`
	ResumeID               int64          `gorm:"column:resume_id"`
	UserID                 int64          `gorm:"column:user_id"`
	AgentRunID             *uint64        `gorm:"column:agent_run_id"`
	RequestedModelID       sql.NullInt64  `gorm:"column:requested_model_id"`
	EffectiveModelID       sql.NullInt64  `gorm:"column:effective_model_id"`
	ModelFallbackReason    sql.NullString `gorm:"column:model_fallback_reason"`
	CapabilityVersionID    sql.NullInt64  `gorm:"column:capability_version_id"`
	CapabilitySnapshotHash sql.NullString `gorm:"column:capability_snapshot_hash"`
	Status                 string         `gorm:"column:status"`
	ParserVersion          sql.NullString `gorm:"column:parser_version"`
	InputHash              sql.NullString `gorm:"column:input_hash"`
	ErrorMessage           sql.NullString `gorm:"column:error_message"`
	StartedAt              time.Time      `gorm:"column:started_at"`
	CompletedAt            *time.Time     `gorm:"column:completed_at"`
	CreatedAt              time.Time      `gorm:"column:created_at"`
	UpdatedAt              time.Time      `gorm:"column:updated_at"`
}

func (recruitingResumeParseRunRecord) TableName() string { return "resume_parse_runs" }

type recruitingResumeProfileRecord struct {
	ID                   uint64          `gorm:"primaryKey"`
	ResumeID             int64           `gorm:"column:resume_id"`
	UserID               int64           `gorm:"column:user_id"`
	ParseRunID           uint64          `gorm:"column:parse_run_id"`
	Version              int32           `gorm:"column:version"`
	IsCurrent            int32           `gorm:"column:is_current"`
	FullName             sql.NullString  `gorm:"column:full_name"`
	Email                sql.NullString  `gorm:"column:email"`
	Phone                sql.NullString  `gorm:"column:phone"`
	Location             sql.NullString  `gorm:"column:location"`
	Headline             sql.NullString  `gorm:"column:headline"`
	Summary              sql.NullString  `gorm:"column:summary"`
	TotalExperienceYears sql.NullFloat64 `gorm:"column:total_experience_years"`
	HighestDegree        sql.NullString  `gorm:"column:highest_degree"`
	RawJSON              sql.NullString  `gorm:"column:raw_json"`
	CreatedAt            time.Time       `gorm:"column:created_at"`
	UpdatedAt            time.Time       `gorm:"column:updated_at"`
}

func (recruitingResumeProfileRecord) TableName() string { return "resume_profiles" }

type recruitingResumeEducationRecord struct {
	ID              uint64         `gorm:"primaryKey"`
	ResumeProfileID uint64         `gorm:"column:resume_profile_id"`
	School          string         `gorm:"column:school"`
	Degree          sql.NullString `gorm:"column:degree"`
	Major           sql.NullString `gorm:"column:major"`
	StartDate       *time.Time     `gorm:"column:start_date"`
	EndDate         *time.Time     `gorm:"column:end_date"`
	Description     sql.NullString `gorm:"column:description"`
	SortOrder       int32          `gorm:"column:sort_order"`
	CreatedAt       time.Time      `gorm:"column:created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at"`
}

func (recruitingResumeEducationRecord) TableName() string { return "resume_educations" }

type recruitingResumeExperienceRecord struct {
	ID               uint64         `gorm:"primaryKey"`
	ResumeProfileID  uint64         `gorm:"column:resume_profile_id"`
	Company          string         `gorm:"column:company"`
	Title            sql.NullString `gorm:"column:title"`
	Location         sql.NullString `gorm:"column:location"`
	StartDate        *time.Time     `gorm:"column:start_date"`
	EndDate          *time.Time     `gorm:"column:end_date"`
	IsCurrent        int32          `gorm:"column:is_current"`
	Description      sql.NullString `gorm:"column:description"`
	AchievementsJSON sql.NullString `gorm:"column:achievements_json"`
	SortOrder        int32          `gorm:"column:sort_order"`
	CreatedAt        time.Time      `gorm:"column:created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at"`
}

func (recruitingResumeExperienceRecord) TableName() string { return "resume_experiences" }

type recruitingResumeProjectRecord struct {
	ID               uint64         `gorm:"primaryKey"`
	ResumeProfileID  uint64         `gorm:"column:resume_profile_id"`
	Name             string         `gorm:"column:name"`
	Role             sql.NullString `gorm:"column:role"`
	StartDate        *time.Time     `gorm:"column:start_date"`
	EndDate          *time.Time     `gorm:"column:end_date"`
	Description      sql.NullString `gorm:"column:description"`
	TechnologiesJSON sql.NullString `gorm:"column:technologies_json"`
	HighlightsJSON   sql.NullString `gorm:"column:highlights_json"`
	SortOrder        int32          `gorm:"column:sort_order"`
	CreatedAt        time.Time      `gorm:"column:created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at"`
}

func (recruitingResumeProjectRecord) TableName() string { return "resume_projects" }

type recruitingResumeSkillRecord struct {
	ID              uint64          `gorm:"primaryKey"`
	ResumeProfileID uint64          `gorm:"column:resume_profile_id"`
	Name            string          `gorm:"column:name"`
	Category        sql.NullString  `gorm:"column:category"`
	Level           sql.NullString  `gorm:"column:level"`
	Years           sql.NullFloat64 `gorm:"column:years"`
	Evidence        sql.NullString  `gorm:"column:evidence"`
	SortOrder       int32           `gorm:"column:sort_order"`
	CreatedAt       time.Time       `gorm:"column:created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at"`
}

func (recruitingResumeSkillRecord) TableName() string { return "resume_skills" }

type recruitingCandidateMatchEvaluationRecord struct {
	ID                     uint64          `gorm:"primaryKey"`
	TenantID               int64           `gorm:"column:tenant_id"`
	ApplicationID          int64           `gorm:"column:application_id"`
	JobID                  int64           `gorm:"column:job_id"`
	CandidateUserID        int64           `gorm:"column:candidate_user_id"`
	ResumeProfileID        uint64          `gorm:"column:resume_profile_id"`
	AgentRunID             *uint64         `gorm:"column:agent_run_id"`
	RequestedModelID       sql.NullInt64   `gorm:"column:requested_model_id"`
	EffectiveModelID       sql.NullInt64   `gorm:"column:effective_model_id"`
	ModelFallbackReason    sql.NullString  `gorm:"column:model_fallback_reason"`
	CapabilityVersionID    sql.NullInt64   `gorm:"column:capability_version_id"`
	CapabilitySnapshotHash sql.NullString  `gorm:"column:capability_snapshot_hash"`
	EvaluationVersion      int32           `gorm:"column:evaluation_version"`
	IsLatest               int32           `gorm:"column:is_latest"`
	OverallScore           sql.NullFloat64 `gorm:"column:overall_score"`
	Recommendation         sql.NullString  `gorm:"column:recommendation"`
	Summary                sql.NullString  `gorm:"column:summary"`
	StrengthsJSON          sql.NullString  `gorm:"column:strengths_json"`
	RisksJSON              sql.NullString  `gorm:"column:risks_json"`
	ScoreBreakdownJSON     sql.NullString  `gorm:"column:score_breakdown_json"`
	ModelName              sql.NullString  `gorm:"column:model_name"`
	EvaluatedAt            time.Time       `gorm:"column:evaluated_at"`
	CreatedAt              time.Time       `gorm:"column:created_at"`
	UpdatedAt              time.Time       `gorm:"column:updated_at"`
}

func (recruitingCandidateMatchEvaluationRecord) TableName() string {
	return "candidate_match_evaluations"
}

type recruitingCandidateMatchEvidenceRecord struct {
	ID           uint64          `gorm:"primaryKey"`
	TenantID     int64           `gorm:"column:tenant_id"`
	EvaluationID uint64          `gorm:"column:evaluation_id"`
	EvidenceType string          `gorm:"column:evidence_type"`
	Dimension    sql.NullString  `gorm:"column:dimension"`
	SourceTable  sql.NullString  `gorm:"column:source_table"`
	SourceID     *uint64         `gorm:"column:source_id"`
	Snippet      sql.NullString  `gorm:"column:snippet"`
	Weight       sql.NullFloat64 `gorm:"column:weight"`
	ScoreImpact  sql.NullFloat64 `gorm:"column:score_impact"`
	MetadataJSON sql.NullString  `gorm:"column:metadata_json"`
	CreatedAt    time.Time       `gorm:"column:created_at"`
}

func (recruitingCandidateMatchEvidenceRecord) TableName() string {
	return "candidate_match_evidence"
}

type aiChatSessionRecord struct {
	ID                     int64      `gorm:"primaryKey"`
	TenantID               *int64     `gorm:"column:tenant_id"`
	HRID                   int64      `gorm:"column:hr_id"`
	OwnerRole              int32      `gorm:"column:owner_role"`
	OwnerID                int64      `gorm:"column:owner_id"`
	Title                  string     `gorm:"column:title"`
	ApplicationID          int64      `gorm:"column:application_id"`
	SessionType            string     `gorm:"column:session_type"`
	SourceType             string     `gorm:"column:source_type"`
	SourceID               int64      `gorm:"column:source_id"`
	SourceTitle            string     `gorm:"column:source_title"`
	Summary                string     `gorm:"column:summary"`
	LastMessagePreview     string     `gorm:"column:last_message_preview"`
	MessageCount           int32      `gorm:"column:message_count"`
	LatestContextUsageJSON *string    `gorm:"column:latest_context_usage_json"`
	SelectedModelID        int64      `gorm:"column:selected_model_id"`
	ActiveRunID            *int64     `gorm:"column:active_run_id"`
	DeletedAt              *time.Time `gorm:"column:deleted_at"`
	CreatedAt              time.Time  `gorm:"column:created_at"`
	UpdatedAt              time.Time  `gorm:"column:updated_at"`
}

func (aiChatSessionRecord) TableName() string { return "ai_chat_sessions" }

type aiChatHistoryRecord struct {
	ID               int64     `gorm:"primaryKey"`
	TenantID         *int64    `gorm:"column:tenant_id"`
	HRID             int64     `gorm:"column:hr_id"`
	OwnerRole        int32     `gorm:"column:owner_role"`
	OwnerID          int64     `gorm:"column:owner_id"`
	SessionID        int64     `gorm:"column:session_id"`
	Role             string    `gorm:"column:role"`
	Content          string    `gorm:"column:content"`
	ProcessContent   string    `gorm:"column:process_content"`
	ModelID          int64     `gorm:"column:model_id"`
	ModelName        string    `gorm:"column:model_name"`
	ContextUsageJSON *string   `gorm:"column:context_usage_json"`
	AgentSkillIDs    *string   `gorm:"column:agent_skill_ids_json"`
	AgentSkillNames  *string   `gorm:"column:agent_skill_names_json"`
	AgentRunID       *int64    `gorm:"column:agent_run_id"`
	CreatedAt        time.Time `gorm:"column:created_at"`
}

func (aiChatHistoryRecord) TableName() string { return "ai_chat_history" }

type aiToolTraceRecord struct {
	ID             int64     `gorm:"primaryKey"`
	TenantID       *int64    `gorm:"column:tenant_id"`
	HRID           int64     `gorm:"column:hr_id"`
	SessionID      int64     `gorm:"column:session_id"`
	AgentRunID     *int64    `gorm:"column:agent_run_id"`
	AgentRunStepID *int64    `gorm:"column:agent_run_step_id"`
	ToolCallID     string    `gorm:"column:tool_call_id"`
	ToolName       string    `gorm:"column:tool_name"`
	ArgsJSON       string    `gorm:"column:arguments_json"`
	ResultContent  string    `gorm:"column:result_summary"`
	Status         string    `gorm:"column:status"`
	DurationMs     int64     `gorm:"column:duration_ms"`
	ErrorMsg       string    `gorm:"column:error_message"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (aiToolTraceRecord) TableName() string { return "ai_tool_traces" }

type agentRunStepRecord struct {
	ID               int64      `gorm:"primaryKey"`
	TenantID         *int64     `gorm:"column:tenant_id"`
	RunID            int64      `gorm:"column:run_id"`
	StepIndex        int32      `gorm:"column:step_index"`
	StepType         string     `gorm:"column:step_type"`
	CapabilitySource string     `gorm:"column:capability_source"`
	CapabilityKey    string     `gorm:"column:capability_key"`
	ToolName         string     `gorm:"column:tool_name"`
	InputJSON        *string    `gorm:"column:input_json"`
	OutputJSON       *string    `gorm:"column:output_json"`
	Status           string     `gorm:"column:status"`
	DurationMs       int64      `gorm:"column:duration_ms"`
	ErrorMsg         string     `gorm:"column:error_message"`
	StartedAt        time.Time  `gorm:"column:started_at"`
	CompletedAt      *time.Time `gorm:"column:completed_at"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}

func (agentRunStepRecord) TableName() string { return "agent_run_steps" }

type thirdPartyUsageLogRecord struct {
	ID              int64     `gorm:"primaryKey"`
	TenantID        *int64    `gorm:"column:tenant_id"`
	UserID          int64     `gorm:"column:user_id"`
	Role            int32     `gorm:"column:role"`
	ServiceType     string    `gorm:"column:service_type"`
	Endpoint        string    `gorm:"column:endpoint"`
	Provider        string    `gorm:"column:provider"`
	Model           string    `gorm:"column:model"`
	RequestChars    int       `gorm:"column:request_chars"`
	ResponseChars   int       `gorm:"column:response_chars"`
	EstimatedTokens int       `gorm:"column:estimated_tokens"`
	Status          string    `gorm:"column:status"`
	ErrorCode       string    `gorm:"column:error_code"`
	CostMs          int       `gorm:"column:cost_ms"`
	RequestID       string    `gorm:"column:request_id"`
	IP              string    `gorm:"column:ip"`
	CreatedAt       time.Time `gorm:"column:created_at"`
}

func (thirdPartyUsageLogRecord) TableName() string { return "third_party_usage_logs" }

type aiUsageAuthContextRecord struct {
	ID            int64     `gorm:"primaryKey"`
	TenantID      *int64    `gorm:"column:tenant_id"`
	UsageLogID    int64     `gorm:"column:usage_log_id"`
	ActorUserID   int64     `gorm:"column:actor_user_id"`
	AccountType   string    `gorm:"column:account_type"`
	RoleKeys      string    `gorm:"column:role_keys"`
	PermissionKey string    `gorm:"column:permission_key"`
	ScopeKeys     string    `gorm:"column:scope_keys"`
	ResourceType  string    `gorm:"column:resource_type"`
	ResourceID    int64     `gorm:"column:resource_id"`
	Decision      string    `gorm:"column:decision"`
	RequestID     string    `gorm:"column:request_id"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (aiUsageAuthContextRecord) TableName() string { return "ai_usage_auth_contexts" }

type agentRunRecord struct {
	ID                int64      `gorm:"primaryKey"`
	TenantID          *int64     `gorm:"column:tenant_id"`
	SessionID         int64      `gorm:"column:session_id"`
	MessageID         int64      `gorm:"column:message_id"`
	HistoryID         int64      `gorm:"column:history_id"`
	HRID              int64      `gorm:"column:hr_id"`
	ClientRequestID   string     `gorm:"column:client_request_id"`
	Status            string     `gorm:"column:status"`
	AssistantText     string     `gorm:"column:assistant_text"`
	ProcessText       string     `gorm:"column:process_text"`
	PlanJSON          *string    `gorm:"column:plan_json"`
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
	TenantID    *int64    `gorm:"column:tenant_id"`
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
	ProtocolType    string         `gorm:"column:protocol_type"`
	AuthType        string         `gorm:"column:auth_type"`
	APIVersion      sql.NullString `gorm:"column:api_version"`
	DiscoveryURL    sql.NullString `gorm:"column:discovery_url"`
	ExtraHeaders    sql.NullString `gorm:"column:extra_headers"`
	IsEnabled       bool           `gorm:"column:is_enabled"`
	CreatedAt       time.Time      `gorm:"column:created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at"`
}

func (llmProviderRecord) TableName() string { return "llm_providers" }

type llmModelListRow struct {
	ID                      int64          `gorm:"column:id"`
	ProviderID              int64          `gorm:"column:provider_id"`
	ModelName               string         `gorm:"column:model_name"`
	CatalogModelName        sql.NullString `gorm:"column:catalog_model_name"`
	DisplayName             string         `gorm:"column:display_name"`
	Temperature             float64        `gorm:"column:temperature"`
	TopP                    float64        `gorm:"column:top_p"`
	MaxTokens               int            `gorm:"column:max_tokens"`
	ContextWindowTokens     int            `gorm:"column:context_window_tokens"`
	ProviderMaxInputTokens  sql.NullInt64  `gorm:"column:provider_max_input_tokens"`
	ProviderMaxOutputTokens sql.NullInt64  `gorm:"column:provider_max_output_tokens"`
	Capabilities            sql.NullString `gorm:"column:capabilities"`
	MetadataSource          string         `gorm:"column:metadata_source"`
	MetadataSources         sql.NullString `gorm:"column:metadata_sources"`
	MetadataSyncedAt        *time.Time     `gorm:"column:metadata_synced_at"`
	TemperatureEnabled      bool           `gorm:"column:temperature_enabled"`
	TopPEnabled             bool           `gorm:"column:top_p_enabled"`
	MaxConcurrency          int            `gorm:"column:max_concurrency"`
	TimeoutSeconds          int            `gorm:"column:timeout_seconds"`
	IsEnabled               bool           `gorm:"column:is_enabled"`
	IsDefault               bool           `gorm:"column:is_default"`
	CreatedAt               time.Time      `gorm:"column:created_at"`
	UpdatedAt               time.Time      `gorm:"column:updated_at"`
	ProviderName            string         `gorm:"column:provider_name"`
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

func recruitingApplicationReadQuery(db *gorm.DB) *gorm.DB {
	return db.Table("applications a").
		Select("a.id AS application_id, a.job_id, a.user_id AS candidate_user_id, COALESCE(cp.real_name, '') AS candidate_name, a.resume_id, a.is_current").
		Joins("LEFT JOIN candidate_profiles cp ON cp.user_id = a.user_id")
}

func (s *NativeStore) listCandidateApplications(ctx context.Context, userID int64, limit int32) ([]aiagentgrpc.CandidateApplicationContext, error) {
	var rows []candidateApplicationReadRow
	if err := s.db.WithContext(ctx).Table("applications a").
		Select("a.id AS application_id, a.job_id, COALESCE(j.title, '') AS job_title, a.status, a.status_key, a.round_no, a.applied_at").
		Joins("LEFT JOIN jobs j ON j.id = a.job_id").
		Where("a.user_id = ?", userID).
		Order("a.applied_at DESC, a.id DESC").
		Limit(int(limit)).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]aiagentgrpc.CandidateApplicationContext, 0, len(rows))
	for _, row := range rows {
		items = append(items, aiagentgrpc.CandidateApplicationContext{
			ApplicationID: row.ApplicationID,
			JobID:         row.JobID,
			JobTitle:      row.JobTitle,
			Status:        row.Status,
			StatusKey:     row.StatusKey,
			StatusText:    candidateApplicationStatusText(row.StatusKey, row.Status),
			RoundNo:       row.RoundNo,
			AppliedAt:     formatTime(row.AppliedAt),
		})
	}
	return items, nil
}

func (s *NativeStore) getCandidateResumeContext(ctx context.Context, userID int64) (aiagentgrpc.CandidateResumeContext, error) {
	var row candidateResumeReadRow
	err := s.db.WithContext(ctx).Table("resumes").
		Select("id AS resume_id, file_name, parsed_text").
		Where("user_id = ? AND is_valid = ?", userID, 1).
		Order("uploaded_at DESC, id DESC").
		Limit(1).
		Scan(&row).Error
	if err != nil {
		return aiagentgrpc.CandidateResumeContext{}, err
	}
	if row.ResumeID == 0 {
		return aiagentgrpc.CandidateResumeContext{Available: false, Message: "candidate has no valid resume"}, nil
	}
	text := strings.TrimSpace(nullString(row.ParsedText))
	if text == "" {
		return aiagentgrpc.CandidateResumeContext{Available: false, ResumeID: row.ResumeID, FileName: row.FileName, Message: "resume parsed text is unavailable"}, nil
	}
	return aiagentgrpc.CandidateResumeContext{
		Available:  true,
		ResumeID:   row.ResumeID,
		FileName:   row.FileName,
		TextLength: len([]rune(text)),
		Summary:    truncateRunes(text, 1200),
	}, nil
}

func (s *NativeStore) listCandidateJobs(ctx context.Context, userID int64, limit int32) ([]aiagentgrpc.CandidateJobContext, error) {
	var rows []candidateJobReadRow
	if err := s.db.WithContext(ctx).Table("jobs j").
		Select("j.id AS job_id, j.title, j.department, j.location, j.salary_range, j.status, CASE WHEN a.id IS NULL THEN false ELSE true END AS has_applied").
		Joins("LEFT JOIN applications a ON a.job_id = j.id AND a.user_id = ? AND a.is_current = 1", userID).
		Where("j.status = ?", 1).
		Order("j.created_at DESC, j.id DESC").
		Limit(int(limit)).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]aiagentgrpc.CandidateJobContext, 0, len(rows))
	for _, row := range rows {
		items = append(items, aiagentgrpc.CandidateJobContext{
			JobID:       row.JobID,
			Title:       row.Title,
			Department:  nullString(row.Department),
			Location:    nullString(row.Location),
			SalaryRange: nullString(row.SalaryRange),
			Status:      row.Status,
			StatusText:  candidateJobStatusText(row.Status),
			HasApplied:  row.HasApplied,
		})
	}
	return items, nil
}

func (s *NativeStore) listCandidateInterviews(ctx context.Context, userID int64, limit int32) ([]aiagentgrpc.CandidateInterviewContext, error) {
	var rows []candidateInterviewReadRow
	if err := s.db.WithContext(ctx).Table("interview_schedules i").
		Select("i.id AS interview_id, i.application_id, COALESCE(j.title, '') AS job_title, i.round_no, i.title, i.mode, i.scheduled_at, i.status, i.candidate_note").
		Joins("JOIN applications a ON a.id = i.application_id AND a.user_id = ?", userID).
		Joins("LEFT JOIN jobs j ON j.id = a.job_id").
		Where("i.deleted_at IS NULL").
		Order("i.scheduled_at DESC, i.id DESC").
		Limit(int(limit)).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]aiagentgrpc.CandidateInterviewContext, 0, len(rows))
	for _, row := range rows {
		items = append(items, aiagentgrpc.CandidateInterviewContext{
			InterviewID:   row.InterviewID,
			ApplicationID: row.ApplicationID,
			JobTitle:      row.JobTitle,
			RoundNo:       row.RoundNo,
			Title:         nullString(row.Title),
			Mode:          nullString(row.Mode),
			ScheduledAt:   formatTimePtr(row.ScheduledAt),
			Status:        row.Status,
			CandidateNote: nullString(row.CandidateNote),
		})
	}
	return items, nil
}

func (s *NativeStore) listCandidateOffers(ctx context.Context, userID int64, limit int32) ([]aiagentgrpc.CandidateOfferContext, error) {
	var rows []candidateOfferReadRow
	if err := s.db.WithContext(ctx).Table("offers").
		Select("id AS offer_id, application_id, job_id, title, status, salary_range, work_location, start_date, expires_at").
		Where("candidate_user_id = ?", userID).
		Order("created_at DESC, id DESC").
		Limit(int(limit)).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]aiagentgrpc.CandidateOfferContext, 0, len(rows))
	for _, row := range rows {
		items = append(items, aiagentgrpc.CandidateOfferContext{
			OfferID:       row.OfferID,
			ApplicationID: row.ApplicationID,
			JobID:         row.JobID,
			Title:         row.Title,
			Status:        row.Status,
			SalaryRange:   nullString(row.SalaryRange),
			WorkLocation:  nullString(row.WorkLocation),
			StartDate:     nullString(row.StartDate),
			ExpiresAt:     formatTimePtr(row.ExpiresAt),
		})
	}
	return items, nil
}

func mapRecruitingApplicationReadRow(row recruitingApplicationReadRow) aiagentgrpc.RecruitingApplicationContext {
	return aiagentgrpc.RecruitingApplicationContext{
		ApplicationID:   row.ApplicationID,
		JobID:           row.JobID,
		CandidateUserID: row.CandidateUserID,
		CandidateName:   row.CandidateName,
		ResumeID:        row.ResumeID,
		IsCurrent:       row.IsCurrent,
	}
}

func mapRecruitingResumeParseRunRecord(row recruitingResumeParseRunRecord) aiagentgrpc.RecruitingResumeParseRunRow {
	return aiagentgrpc.RecruitingResumeParseRunRow{
		ID:                     row.ID,
		ResumeID:               row.ResumeID,
		UserID:                 row.UserID,
		AgentRunID:             row.AgentRunID,
		RequestedModelID:       nullInt64(row.RequestedModelID),
		EffectiveModelID:       nullInt64(row.EffectiveModelID),
		ModelFallbackReason:    nullString(row.ModelFallbackReason),
		CapabilityVersionID:    nullInt64(row.CapabilityVersionID),
		CapabilitySnapshotHash: nullString(row.CapabilitySnapshotHash),
		Status:                 row.Status,
		ParserVersion:          nullString(row.ParserVersion),
		InputHash:              nullString(row.InputHash),
		ErrorMessage:           nullString(row.ErrorMessage),
		StartedAt:              row.StartedAt,
		CompletedAt:            row.CompletedAt,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
}

func mapRecruitingResumeProfileRecord(row recruitingResumeProfileRecord) aiagentgrpc.RecruitingResumeProfileRow {
	return aiagentgrpc.RecruitingResumeProfileRow{
		ID:                   row.ID,
		ResumeID:             row.ResumeID,
		UserID:               row.UserID,
		ParseRunID:           row.ParseRunID,
		Version:              row.Version,
		IsCurrent:            row.IsCurrent,
		FullName:             nullString(row.FullName),
		Email:                nullString(row.Email),
		Phone:                nullString(row.Phone),
		Location:             nullString(row.Location),
		Headline:             nullString(row.Headline),
		Summary:              nullString(row.Summary),
		TotalExperienceYears: nullFloat64(row.TotalExperienceYears),
		HighestDegree:        nullString(row.HighestDegree),
		RawJSON:              nullString(row.RawJSON),
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func mapRecruitingResumeEducationRecord(row recruitingResumeEducationRecord) aiagentgrpc.RecruitingResumeEducationRow {
	return aiagentgrpc.RecruitingResumeEducationRow{
		ID:          row.ID,
		School:      row.School,
		Degree:      nullString(row.Degree),
		Major:       nullString(row.Major),
		StartDate:   row.StartDate,
		EndDate:     row.EndDate,
		Description: nullString(row.Description),
		SortOrder:   row.SortOrder,
	}
}

func mapRecruitingResumeExperienceRecord(row recruitingResumeExperienceRecord) aiagentgrpc.RecruitingResumeExperienceRow {
	return aiagentgrpc.RecruitingResumeExperienceRow{
		ID:               row.ID,
		Company:          row.Company,
		Title:            nullString(row.Title),
		Location:         nullString(row.Location),
		StartDate:        row.StartDate,
		EndDate:          row.EndDate,
		IsCurrent:        row.IsCurrent,
		Description:      nullString(row.Description),
		AchievementsJSON: nullString(row.AchievementsJSON),
		SortOrder:        row.SortOrder,
	}
}

func mapRecruitingResumeProjectRecord(row recruitingResumeProjectRecord) aiagentgrpc.RecruitingResumeProjectRow {
	return aiagentgrpc.RecruitingResumeProjectRow{
		ID:               row.ID,
		Name:             row.Name,
		Role:             nullString(row.Role),
		StartDate:        row.StartDate,
		EndDate:          row.EndDate,
		Description:      nullString(row.Description),
		TechnologiesJSON: nullString(row.TechnologiesJSON),
		HighlightsJSON:   nullString(row.HighlightsJSON),
		SortOrder:        row.SortOrder,
	}
}

func mapRecruitingResumeSkillRecord(row recruitingResumeSkillRecord) aiagentgrpc.RecruitingResumeSkillRow {
	return aiagentgrpc.RecruitingResumeSkillRow{
		ID:        row.ID,
		Name:      row.Name,
		Category:  nullString(row.Category),
		Level:     nullString(row.Level),
		Years:     nullFloat64(row.Years),
		Evidence:  nullString(row.Evidence),
		SortOrder: row.SortOrder,
	}
}

func mapRecruitingCandidateMatchEvaluationRecord(row recruitingCandidateMatchEvaluationRecord) aiagentgrpc.RecruitingCandidateMatchEvaluationRow {
	return aiagentgrpc.RecruitingCandidateMatchEvaluationRow{
		ID:                     row.ID,
		ApplicationID:          row.ApplicationID,
		JobID:                  row.JobID,
		CandidateUserID:        row.CandidateUserID,
		ResumeProfileID:        row.ResumeProfileID,
		AgentRunID:             row.AgentRunID,
		RequestedModelID:       nullInt64(row.RequestedModelID),
		EffectiveModelID:       nullInt64(row.EffectiveModelID),
		ModelFallbackReason:    nullString(row.ModelFallbackReason),
		CapabilityVersionID:    nullInt64(row.CapabilityVersionID),
		CapabilitySnapshotHash: nullString(row.CapabilitySnapshotHash),
		EvaluationVersion:      row.EvaluationVersion,
		IsLatest:               row.IsLatest,
		OverallScore:           nullFloat64(row.OverallScore),
		Recommendation:         nullString(row.Recommendation),
		Summary:                nullString(row.Summary),
		StrengthsJSON:          nullString(row.StrengthsJSON),
		RisksJSON:              nullString(row.RisksJSON),
		ScoreBreakdownJSON:     nullString(row.ScoreBreakdownJSON),
		ModelName:              nullString(row.ModelName),
		EvaluatedAt:            row.EvaluatedAt,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
}

func mapRecruitingCandidateMatchEvidenceRecord(row recruitingCandidateMatchEvidenceRecord) aiagentgrpc.RecruitingCandidateMatchEvidenceRow {
	return aiagentgrpc.RecruitingCandidateMatchEvidenceRow{
		ID:           row.ID,
		EvidenceType: row.EvidenceType,
		Dimension:    nullString(row.Dimension),
		SourceTable:  nullString(row.SourceTable),
		SourceID:     row.SourceID,
		Snippet:      nullString(row.Snippet),
		Weight:       nullFloat64(row.Weight),
		ScoreImpact:  nullFloat64(row.ScoreImpact),
		MetadataJSON: nullString(row.MetadataJSON),
		CreatedAt:    row.CreatedAt,
	}
}

func mapSessionRecord(ctx context.Context, row aiChatSessionRecord) aiagentgrpc.ChatSessionRow {
	return aiagentgrpc.ChatSessionRow{
		ID:                 row.ID,
		Title:              row.Title,
		ApplicationID:      row.ApplicationID,
		SessionType:        row.SessionType,
		SourceType:         row.SourceType,
		SourceID:           row.SourceID,
		SourceTitle:        row.SourceTitle,
		Summary:            row.Summary,
		LastMessagePreview: row.LastMessagePreview,
		MessageCount:       row.MessageCount,
		LatestContextUsage: parseContextUsage(ctx, "chat_session", row.ID, row.ID, row.LatestContextUsageJSON),
		SelectedModelID:    row.SelectedModelID,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func normalizeChatSessionType(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "general"
	}
	return trimmed
}

func chatMessagePreview(content string) string {
	preview := strings.Join(strings.Fields(strings.TrimSpace(content)), " ")
	if preview == "" {
		return ""
	}
	runes := []rune(preview)
	if len(runes) > 500 {
		return string(runes[:500])
	}
	return preview
}

func mapMessageRecord(ctx context.Context, row aiChatHistoryRecord) aiagentgrpc.ChatMessageRow {
	return aiagentgrpc.ChatMessageRow{ID: row.ID, OwnerRole: row.OwnerRole, OwnerID: row.OwnerID, SessionID: row.SessionID, Role: row.Role, Content: row.Content, ProcessContent: row.ProcessContent, ModelID: row.ModelID, ModelName: row.ModelName, ContextUsage: parseContextUsage(ctx, "chat_message", row.ID, row.SessionID, row.ContextUsageJSON), AgentSkillIDs: parseInt64JSONSlice(stringValue(row.AgentSkillIDs)), AgentSkillNames: parseStringJSONSlice(stringValue(row.AgentSkillNames)), CreatedAt: row.CreatedAt}
}

var contextUsageMarshalOptions = protojson.MarshalOptions{UseProtoNames: true}

func marshalContextUsage(usage *pb.ContextUsageInfo) (*string, error) {
	if usage == nil {
		return nil, nil
	}
	payload, err := contextUsageMarshalOptions.Marshal(usage)
	if err != nil {
		return nil, err
	}
	encoded := string(payload)
	return &encoded, nil
}

func parseContextUsage(ctx context.Context, recordType string, recordID, sessionID int64, payload *string) *pb.ContextUsageInfo {
	if payload == nil || strings.TrimSpace(*payload) == "" {
		return nil
	}
	usage := &pb.ContextUsageInfo{}
	if err := protojson.Unmarshal([]byte(*payload), usage); err != nil {
		platformlogger.GetRequestLogger(ctx).Warn("invalid persisted context usage snapshot",
			zap.String("record_type", recordType),
			zap.Int64("record_id", recordID),
			zap.Int64("session_id", sessionID),
			zap.String("error_code", "context_usage_parse_failed"),
			zap.String("error_category", "invalid_json"),
		)
		return nil
	}
	return usage
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
	return aiagentgrpc.AgentRunRow{ID: row.ID, TenantID: int64FromPointer(row.TenantID), SessionID: row.SessionID, MessageID: row.MessageID, HistoryID: row.HistoryID, OwnerID: row.HRID, ClientRequestID: row.ClientRequestID, Status: row.Status, AssistantText: row.AssistantText, ProcessText: row.ProcessText, PlanJSON: stringValue(row.PlanJSON), OptionContextJSON: stringValue(row.OptionContext), LastEventSeq: row.LastEventSeq, ErrorType: row.ErrorType, ErrorMessage: row.ErrorMessage, ModelID: row.ModelID, ModelName: row.ModelName, AgentType: row.AgentType, AgentID: row.AgentID, AgentName: row.AgentName, StartedAt: row.StartedAt, CompletedAt: row.CompletedAt, CancelRequestedAt: row.CancelRequestedAt, CanceledAt: row.CanceledAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func positiveInt64Pointer(value int64) *int64 {
	if value <= 0 {
		return nil
	}
	return &value
}

func int64FromPointer(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
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

func nullableSQLString(value string) sql.NullString {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: trimmed, Valid: true}
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

func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func candidateApplicationStatusText(statusKey string, status int32) string {
	switch strings.TrimSpace(statusKey) {
	case "applied":
		return "待查看"
	case "screening":
		return "筛选中"
	case "interview":
		return "面试中"
	case "offer":
		return "Offer 阶段"
	case "hired":
		return "已录用"
	case "rejected":
		return "未通过"
	case "withdrawn":
		return "已撤回"
	case "offer_rejected":
		return "Offer 已拒绝"
	}
	switch status {
	case 0:
		return "待查看"
	case 1:
		return "已查看"
	case 2:
		return "已通过"
	case 3:
		return "未通过"
	default:
		return "未知"
	}
}

func candidateJobStatusText(status int32) string {
	if status == 1 {
		return "招募中"
	}
	return "已下架"
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return businessclock.FormatRFC3339(value)
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

func marshalInt64Slice(values []int64) string {
	if len(values) == 0 {
		return ""
	}
	data, err := json.Marshal(values)
	if err != nil {
		return ""
	}
	return string(data)
}

func marshalStringSlice(values []string) string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			cleaned = append(cleaned, strings.TrimSpace(value))
		}
	}
	if len(cleaned) == 0 {
		return ""
	}
	data, err := json.Marshal(cleaned)
	if err != nil {
		return ""
	}
	return string(data)
}

func parseInt64JSONSlice(value string) []int64 {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	var items []int64
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil
	}
	return items
}

func parseStringJSONSlice(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil
	}
	return items
}
