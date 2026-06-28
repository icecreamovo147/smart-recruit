package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

type AgentRunRepo struct {
	db *gorm.DB
}

func NewAgentRunRepo(db *gorm.DB) *AgentRunRepo {
	return &AgentRunRepo{db: db}
}

func (r *AgentRunRepo) CreateRun(ctx context.Context, run *model.AgentRun) error {
	return r.db.WithContext(ctx).Create(run).Error
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

func (r *AgentRunRepo) UpdateRunPlan(ctx context.Context, runID uint64, planJSON string) error {
	return r.db.WithContext(ctx).Model(&model.AgentRun{}).Where("id = ?", runID).Update("plan_json", planJSON).Error
}

func (r *AgentRunRepo) UpdateRunMessageID(ctx context.Context, runID, messageID uint64) error {
	return r.db.WithContext(ctx).
		Model(&model.AgentRun{}).
		Where("id = ?", runID).
		Updates(map[string]any{"message_id": messageID, "history_id": messageID}).Error
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
