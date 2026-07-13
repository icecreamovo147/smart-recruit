package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"smart-recruit-interview-service/internal/legacydomain/model"
)

type AgentRunRepo struct {
	db *gorm.DB
}

// AgentRunSnapshotPatch updates durable snapshot fields used for refresh restore.
// Nil pointer fields are left unchanged.
type AgentRunSnapshotPatch struct {
	AssistantText           *string
	ProcessText             *string
	ResultMetadataJSON      *string
	ConfirmationRequestJSON *string
	OptionContextJSON       *string
	ModelName               *string
	LastEventSeq            *int64
	FinalAnswer             *string
	ErrorType               *string
	ErrorMessage            *string
}

func NewAgentRunRepo(db *gorm.DB) *AgentRunRepo {
	return &AgentRunRepo{db: db}
}

// mysqlJSONValue maps empty strings to SQL NULL. MySQL rejects ” for JSON columns
// with Error 3140 ("The document is empty").
func mysqlJSONValue(raw string) any {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return raw
}

func (r *AgentRunRepo) CreateRun(ctx context.Context, run *model.AgentRun) error {
	if run == nil {
		return errors.New("agent run is nil")
	}
	// MySQL JSON columns reject empty string '' (Error 3140). Omit zero-value JSON
	// fields so the INSERT leaves them as NULL. BeforeCreate is a second line of defense.
	db := r.db.WithContext(ctx)
	var omits []string
	if strings.TrimSpace(run.PlanJSON) == "" {
		omits = append(omits, "PlanJSON")
	}
	if strings.TrimSpace(run.ResultMetadataJSON) == "" {
		omits = append(omits, "ResultMetadataJSON")
	}
	if strings.TrimSpace(run.ConfirmationRequestJSON) == "" {
		omits = append(omits, "ConfirmationRequestJSON")
	}
	if strings.TrimSpace(run.OptionContextJSON) == "" {
		omits = append(omits, "OptionContextJSON")
	}
	if len(omits) > 0 {
		db = db.Omit(omits...)
	}
	return db.Create(run).Error
}

func (r *AgentRunRepo) GetRunByID(ctx context.Context, runID uint64) (*model.AgentRun, error) {
	var run model.AgentRun
	err := r.db.WithContext(ctx).First(&run, runID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// GetRunByClientRequestID looks up an idempotent create-run key.
func (r *AgentRunRepo) GetRunByClientRequestID(ctx context.Context, hrID, sessionID uint64, clientRequestID string) (*model.AgentRun, error) {
	if clientRequestID == "" {
		return nil, nil
	}
	var run model.AgentRun
	err := r.db.WithContext(ctx).
		Where("hr_id = ? AND session_id = ? AND client_request_id = ?", hrID, sessionID, clientRequestID).
		First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// GetActiveRunBySession loads the session's active_run_id pointer and returns that run.
func (r *AgentRunRepo) GetActiveRunBySession(ctx context.Context, sessionID int64) (*model.AgentRun, error) {
	var session model.AIChatSession
	err := r.db.WithContext(ctx).Select("id", "active_run_id").First(&session, sessionID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if session.ActiveRunID == nil || *session.ActiveRunID <= 0 {
		return nil, nil
	}
	return r.GetRunByID(ctx, uint64(*session.ActiveRunID))
}

// SetSessionActiveRun updates ai_chat_sessions.active_run_id. Pass nil to clear.
func (r *AgentRunRepo) SetSessionActiveRun(ctx context.Context, sessionID int64, runID *int64) error {
	return r.db.WithContext(ctx).
		Model(&model.AIChatSession{}).
		Where("id = ?", sessionID).
		Update("active_run_id", runID).Error
}

func (r *AgentRunRepo) UpdateRunStatus(ctx context.Context, runID uint64, status, finalAnswer, errorType, errorMessage string, completedAt *time.Time) error {
	updates := map[string]any{
		"status":        status,
		"final_answer":  finalAnswer,
		"error_type":    errorType,
		"error_message": errorMessage,
	}
	if completedAt != nil {
		updates["completed_at"] = completedAt
	}
	return r.db.WithContext(ctx).Model(&model.AgentRun{}).Where("id = ?", runID).Updates(updates).Error
}

// UpdateRunStatusFields updates status and optional cancel/completion timestamps.
func (r *AgentRunRepo) UpdateRunStatusFields(ctx context.Context, runID uint64, status string, completedAt, cancelRequestedAt, canceledAt *time.Time) error {
	updates := map[string]any{"status": status}
	if completedAt != nil {
		updates["completed_at"] = completedAt
	}
	if cancelRequestedAt != nil {
		updates["cancel_requested_at"] = cancelRequestedAt
	}
	if canceledAt != nil {
		updates["canceled_at"] = canceledAt
	}
	return r.db.WithContext(ctx).Model(&model.AgentRun{}).Where("id = ?", runID).Updates(updates).Error
}

// UpdateRunSnapshot applies a partial durable snapshot patch for refresh restore.
func (r *AgentRunRepo) UpdateRunSnapshot(ctx context.Context, runID uint64, patch AgentRunSnapshotPatch) error {
	updates := map[string]any{}
	if patch.AssistantText != nil {
		updates["assistant_text"] = *patch.AssistantText
	}
	if patch.ProcessText != nil {
		updates["process_text"] = *patch.ProcessText
	}
	if patch.ResultMetadataJSON != nil {
		updates["result_metadata_json"] = mysqlJSONValue(*patch.ResultMetadataJSON)
	}
	if patch.ConfirmationRequestJSON != nil {
		updates["confirmation_request_json"] = mysqlJSONValue(*patch.ConfirmationRequestJSON)
	}
	if patch.OptionContextJSON != nil {
		updates["option_context_json"] = mysqlJSONValue(*patch.OptionContextJSON)
	}
	if patch.ModelName != nil {
		updates["model_name"] = *patch.ModelName
	}
	if patch.LastEventSeq != nil {
		updates["last_event_seq"] = *patch.LastEventSeq
	}
	if patch.FinalAnswer != nil {
		updates["final_answer"] = *patch.FinalAnswer
	}
	if patch.ErrorType != nil {
		updates["error_type"] = *patch.ErrorType
	}
	if patch.ErrorMessage != nil {
		updates["error_message"] = *patch.ErrorMessage
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&model.AgentRun{}).Where("id = ?", runID).Updates(updates).Error
}

func (r *AgentRunRepo) UpdateRunPlan(ctx context.Context, runID uint64, planJSON string) error {
	return r.db.WithContext(ctx).Model(&model.AgentRun{}).Where("id = ?", runID).Update("plan_json", planJSON).Error
}

// UpdateRunMessageID sets message_id (user message association) without touching history_id.
// history_id is reserved for the final assistant history row (see UpdateRunHistoryID).
func (r *AgentRunRepo) UpdateRunMessageID(ctx context.Context, runID, messageID uint64) error {
	return r.db.WithContext(ctx).
		Model(&model.AgentRun{}).
		Where("id = ?", runID).
		Update("message_id", messageID).Error
}

// UpdateRunHistoryID sets history_id (assistant message association) without touching message_id.
func (r *AgentRunRepo) UpdateRunHistoryID(ctx context.Context, runID, historyID uint64) error {
	return r.db.WithContext(ctx).
		Model(&model.AgentRun{}).
		Where("id = ?", runID).
		Update("history_id", historyID).Error
}

// CountAssistantHistoryByAgentRunID returns how many assistant history rows reference runID.
func (r *AgentRunRepo) CountAssistantHistoryByAgentRunID(ctx context.Context, runID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.AIChatHistory{}).
		Where("agent_run_id = ? AND role = ?", int64(runID), "assistant").
		Count(&count).Error
	return count, err
}

func (r *AgentRunRepo) CreateStep(ctx context.Context, step *model.AgentRunStep) error {
	return r.db.WithContext(ctx).Create(step).Error
}

func (r *AgentRunRepo) NextStepIndex(ctx context.Context, runID uint64) (int, error) {
	var maxIndex int
	err := r.db.WithContext(ctx).
		Model(&model.AgentRunStep{}).
		Where("run_id = ?", runID).
		Select("COALESCE(MAX(step_index), -1)").
		Scan(&maxIndex).Error
	if err != nil {
		return 0, err
	}
	return maxIndex + 1, nil
}

func (r *AgentRunRepo) ListRunsBySession(ctx context.Context, hrID, sessionID int64, limit int) ([]model.AgentRun, error) {
	var rows []model.AgentRun
	err := r.db.WithContext(ctx).
		Where("hr_id = ? AND session_id = ?", hrID, sessionID).
		Order("created_at DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *AgentRunRepo) ListStepsByRunIDs(ctx context.Context, runIDs []uint64) ([]model.AgentRunStep, error) {
	if len(runIDs) == 0 {
		return nil, nil
	}
	var rows []model.AgentRunStep
	err := r.db.WithContext(ctx).
		Where("run_id IN ?", runIDs).
		Order("run_id DESC, step_index ASC").
		Find(&rows).Error
	return rows, err
}
