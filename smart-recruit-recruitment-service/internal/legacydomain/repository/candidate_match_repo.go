package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"smart-recruit-recruitment-service/internal/legacydomain/model"
)

type CandidateMatchRepo struct {
	db *gorm.DB
}

func NewCandidateMatchRepo(db *gorm.DB) *CandidateMatchRepo {
	return &CandidateMatchRepo{db: db}
}

type CandidateMatchSnapshot struct {
	Evaluation model.CandidateMatchEvaluation
	Evidence   []model.CandidateMatchEvidence
}

func (r *CandidateMatchRepo) SaveEvaluationVersion(ctx context.Context, snapshot *CandidateMatchSnapshot) error {
	if snapshot == nil {
		return fmt.Errorf("candidate match snapshot is nil")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		evaluation := &snapshot.Evaluation
		if evaluation.EvaluatedAt.IsZero() {
			evaluation.EvaluatedAt = time.Now()
		}

		if evaluation.ID == 0 {
			var version int32
			if err := tx.Model(&model.CandidateMatchEvaluation{}).
				Where("application_id = ?", evaluation.ApplicationID).
				Select("COALESCE(MAX(evaluation_version), 0)").
				Scan(&version).Error; err != nil {
				return fmt.Errorf("get next candidate match version: %w", err)
			}
			if err := tx.Model(&model.CandidateMatchEvaluation{}).
				Where("application_id = ? AND is_latest = 1", evaluation.ApplicationID).
				Update("is_latest", 0).Error; err != nil {
				return fmt.Errorf("clear latest candidate match evaluation: %w", err)
			}
			evaluation.EvaluationVersion = version + 1
			evaluation.IsLatest = 1
			if err := tx.Create(evaluation).Error; err != nil {
				return fmt.Errorf("create candidate match evaluation: %w", err)
			}
		} else {
			if err := tx.Model(&model.CandidateMatchEvaluation{}).
				Where("id = ?", evaluation.ID).
				Updates(map[string]any{
					"job_id":               evaluation.JobID,
					"candidate_user_id":    evaluation.CandidateUserID,
					"resume_profile_id":    evaluation.ResumeProfileID,
					"agent_run_id":         evaluation.AgentRunID,
					"overall_score":        evaluation.OverallScore,
					"recommendation":       evaluation.Recommendation,
					"summary":              evaluation.Summary,
					"strengths_json":       evaluation.StrengthsJSON,
					"risks_json":           evaluation.RisksJSON,
					"score_breakdown_json": evaluation.ScoreBreakdownJSON,
					"model_name":           evaluation.ModelName,
					"evaluated_at":         evaluation.EvaluatedAt,
				}).Error; err != nil {
				return fmt.Errorf("update candidate match evaluation: %w", err)
			}
		}

		if err := tx.Where("evaluation_id = ?", evaluation.ID).Delete(&model.CandidateMatchEvidence{}).Error; err != nil {
			return fmt.Errorf("delete candidate match evidence: %w", err)
		}
		for i := range snapshot.Evidence {
			snapshot.Evidence[i].EvaluationID = evaluation.ID
			snapshot.Evidence[i].MetadataJSON = normalizeCandidateMatchEvidenceMetadata(snapshot.Evidence[i].MetadataJSON)
			if err := tx.Create(&snapshot.Evidence[i]).Error; err != nil {
				return fmt.Errorf("create candidate match evidence: %w", err)
			}
		}
		return nil
	})
}

func (r *CandidateMatchRepo) GetLatestByApplicationID(ctx context.Context, applicationID int64) (*model.CandidateMatchEvaluation, error) {
	var evaluation model.CandidateMatchEvaluation
	err := r.db.WithContext(ctx).
		Where("application_id = ? AND is_latest = 1", applicationID).
		Order("evaluation_version DESC").
		First(&evaluation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &evaluation, err
}

func (r *CandidateMatchRepo) GetByApplicationIDAndVersion(ctx context.Context, applicationID int64, version int32) (*model.CandidateMatchEvaluation, error) {
	var evaluation model.CandidateMatchEvaluation
	err := r.db.WithContext(ctx).
		Where("application_id = ? AND evaluation_version = ?", applicationID, version).
		First(&evaluation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &evaluation, err
}

func (r *CandidateMatchRepo) ListLatestByApplicationIDs(ctx context.Context, applicationIDs []int64) ([]model.CandidateMatchEvaluation, error) {
	if len(applicationIDs) == 0 {
		return nil, nil
	}
	var evaluations []model.CandidateMatchEvaluation
	err := r.db.WithContext(ctx).
		Where("application_id IN ? AND is_latest = 1", applicationIDs).
		Order("overall_score DESC, application_id ASC").
		Find(&evaluations).Error
	return evaluations, err
}

func (r *CandidateMatchRepo) GetSnapshot(ctx context.Context, evaluationID uint64) (*CandidateMatchSnapshot, error) {
	var evaluation model.CandidateMatchEvaluation
	if err := r.db.WithContext(ctx).First(&evaluation, evaluationID).Error; err != nil {
		return nil, err
	}
	var evidence []model.CandidateMatchEvidence
	if err := r.db.WithContext(ctx).
		Where("evaluation_id = ?", evaluationID).
		Order("id ASC").
		Find(&evidence).Error; err != nil {
		return nil, err
	}
	return &CandidateMatchSnapshot{Evaluation: evaluation, Evidence: evidence}, nil
}

func normalizeCandidateMatchEvidenceMetadata(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}"
	}
	if json.Valid([]byte(raw)) {
		return raw
	}
	wrapped, err := json.Marshal(map[string]string{"value": raw})
	if err != nil {
		return "{}"
	}
	return string(wrapped)
}
