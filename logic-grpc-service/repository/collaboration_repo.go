package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

type CollaborationRepo struct {
	db *gorm.DB
}

func NewCollaborationRepo(db *gorm.DB) *CollaborationRepo {
	return &CollaborationRepo{db: db}
}

// ── Notes ──────────────────────────────────────────────────────────────

func (r *CollaborationRepo) CreateNote(ctx context.Context, note *model.CandidateNote) error {
	return r.db.WithContext(ctx).Create(note).Error
}

func (r *CollaborationRepo) ListNotes(ctx context.Context, candidateUserID uint64, applicationID *uint64) ([]model.CandidateNote, error) {
	query := r.db.WithContext(ctx).
		Where("candidate_user_id = ?", candidateUserID)
	if applicationID != nil && *applicationID > 0 {
		query = query.Where("application_id = ?", *applicationID)
	}
	var notes []model.CandidateNote
	err := query.Order("created_at DESC").Find(&notes).Error
	return notes, err
}

// ── Tags ───────────────────────────────────────────────────────────────

func (r *CollaborationRepo) CreateTag(ctx context.Context, tag *model.CandidateTag) error {
	return r.db.WithContext(ctx).Create(tag).Error
}

func (r *CollaborationRepo) ListTags(ctx context.Context) ([]model.CandidateTag, error) {
	var tags []model.CandidateTag
	err := r.db.WithContext(ctx).Order("name ASC").Find(&tags).Error
	return tags, err
}

func (r *CollaborationRepo) AssignTag(ctx context.Context, assignment *model.CandidateTagAssignment) error {
	return r.db.WithContext(ctx).Create(assignment).Error
}

func (r *CollaborationRepo) UnassignTag(ctx context.Context, tagID uint64, candidateUserID uint64) error {
	return r.db.WithContext(ctx).
		Where("tag_id = ? AND candidate_user_id = ?", tagID, candidateUserID).
		Delete(&model.CandidateTagAssignment{}).Error
}

func (r *CollaborationRepo) ListCandidateTags(ctx context.Context, candidateUserID uint64) ([]model.CandidateTag, error) {
	var tags []model.CandidateTag
	err := r.db.WithContext(ctx).
		Table("candidate_tags").
		Select("candidate_tags.*").
		Joins("JOIN candidate_tag_assignments cta ON cta.tag_id = candidate_tags.id").
		Where("cta.candidate_user_id = ?", candidateUserID).
		Order("candidate_tags.name ASC").
		Scan(&tags).Error
	return tags, err
}

// ── Follow-up Tasks ────────────────────────────────────────────────────

func (r *CollaborationRepo) CreateTask(ctx context.Context, task *model.FollowUpTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *CollaborationRepo) GetTask(ctx context.Context, taskID uint64) (*model.FollowUpTask, error) {
	var task model.FollowUpTask
	err := r.db.WithContext(ctx).Where("id = ?", taskID).First(&task).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &task, nil
}

func (r *CollaborationRepo) ListTasks(ctx context.Context, filter TaskFilter) ([]model.FollowUpTask, error) {
	query := r.db.WithContext(ctx).Model(&model.FollowUpTask{})
	if filter.CandidateUserID != nil && *filter.CandidateUserID > 0 {
		query = query.Where("candidate_user_id = ?", *filter.CandidateUserID)
	}
	if filter.AssigneeUserID != nil && *filter.AssigneeUserID > 0 {
		query = query.Where("assignee_user_id = ?", *filter.AssigneeUserID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	var tasks []model.FollowUpTask
	err := query.Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

func (r *CollaborationRepo) UpdateTaskStatus(ctx context.Context, taskID uint64, status string, completedAt *time.Time) error {
	updates := map[string]any{"status": status}
	if completedAt != nil {
		updates["completed_at"] = completedAt
	}
	return r.db.WithContext(ctx).Model(&model.FollowUpTask{}).
		Where("id = ?", taskID).
		Updates(updates).Error
}

// ── Aggregation ────────────────────────────────────────────────────────

// CountApplicationsByUser returns the count of applications for a candidate.
func (r *CollaborationRepo) CountApplicationsByUser(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Application{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

// CountInterviewsByCandidate returns the count of interviews for a candidate.
func (r *CollaborationRepo) CountInterviewsByCandidate(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("interview_schedules").
		Joins("JOIN applications a ON a.id = interview_schedules.application_id").
		Where("a.user_id = ? AND interview_schedules.deleted_at IS NULL", userID).
		Count(&count).Error
	return count, err
}

// CountOffersByCandidate returns the count of offers for a candidate.
func (r *CollaborationRepo) CountOffersByCandidate(ctx context.Context, candidateUserID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Offer{}).
		Where("candidate_user_id = ?", candidateUserID).
		Count(&count).Error
	return count, err
}

// GetLatestActivity returns the latest updated_at timestamp across applications.
func (r *CollaborationRepo) GetLatestActivity(ctx context.Context, userID uint64) (*time.Time, error) {
	var latest time.Time
	// Use ORDER BY + LIMIT 1 instead of MAX() for SQLite compatibility
	err := r.db.WithContext(ctx).Model(&model.Application{}).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(1).
		Pluck("updated_at", &latest).Error
	if err != nil {
		return nil, err
	}
	if latest.IsZero() {
		return nil, nil
	}
	return &latest, nil
}

type TaskFilter struct {
	CandidateUserID *uint64
	AssigneeUserID  *uint64
	Status          string
}
