package repository

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

type InterviewRepo struct {
	db *gorm.DB
}

type InterviewWithDetailsRow struct {
	ID              int64
	ApplicationID   int64
	InterviewerID   int64
	RoundNo         int32
	Title           string
	Mode            string
	MeetingURL      string
	Location        string
	DurationMinutes int32
	CandidateNote   string
	InternalNote    string
	CancelReason    string
	ScheduledAt     *time.Time
	Status          string
	CreatedBy       *int64
	CreatedAt       time.Time
	UpdatedAt       time.Time

	// Joined fields
	InterviewerName      string
	ApplicationStatusKey string
	JobTitle             string
	CandidateName        string
	CandidatePhone       string
	ResumeOssKey         string
}

func NewInterviewRepo(db *gorm.DB) *InterviewRepo {
	return &InterviewRepo{db: db}
}

// baseInterviewSelect returns the common select expression for joined interview queries.
func baseInterviewSelect() string {
	return `interview_schedules.id, interview_schedules.application_id, interview_schedules.interviewer_id,
		interview_schedules.round_no, interview_schedules.title, interview_schedules.mode,
		interview_schedules.meeting_url, interview_schedules.location, interview_schedules.duration_minutes,
		interview_schedules.candidate_note, interview_schedules.internal_note, interview_schedules.cancel_reason,
		interview_schedules.scheduled_at, interview_schedules.status, interview_schedules.created_by,
		interview_schedules.created_at, interview_schedules.updated_at,
		u.username AS interviewer_name,
		a.status_key AS application_status_key,
		j.title AS job_title,
		COALESCE(cp.real_name, CONCAT('候选人', a.user_id)) AS candidate_name,
		COALESCE(cp.phone, '') AS candidate_phone,
		COALESCE(res.oss_key, '') AS resume_oss_key`
}

func (r *InterviewRepo) baseJoins() *gorm.DB {
	return r.db.Table("interview_schedules").
		Select(baseInterviewSelect()).
		Joins("JOIN users u ON u.id = interview_schedules.interviewer_id").
		Joins("JOIN applications a ON a.id = interview_schedules.application_id").
		Joins("JOIN jobs j ON j.id = a.job_id").
		Joins("LEFT JOIN candidate_profiles cp ON cp.user_id = a.user_id").
		Joins("LEFT JOIN resumes res ON res.user_id = a.user_id AND res.id = (SELECT MAX(r2.id) FROM resumes r2 WHERE r2.user_id = a.user_id)")
}

// ── Schedule CRUD ──────────────────────────────────────────────────────

func (r *InterviewRepo) Create(ctx context.Context, s *model.InterviewSchedule) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *InterviewRepo) CreateWithTx(ctx context.Context, tx *gorm.DB, s *model.InterviewSchedule) error {
	return tx.WithContext(ctx).Create(s).Error
}

func (r *InterviewRepo) GetByID(ctx context.Context, id int64) (*InterviewWithDetailsRow, error) {
	var row InterviewWithDetailsRow
	err := r.baseJoins().
		Where("interview_schedules.id = ? AND interview_schedules.deleted_at IS NULL", id).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

func (r *InterviewRepo) GetModelByID(ctx context.Context, id int64) (*model.InterviewSchedule, error) {
	var s model.InterviewSchedule
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&s).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *InterviewRepo) Update(ctx context.Context, s *model.InterviewSchedule) error {
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *InterviewRepo) UpdateWithTx(ctx context.Context, tx *gorm.DB, s *model.InterviewSchedule) error {
	return tx.WithContext(ctx).Save(s).Error
}

// GetMaxRoundNo returns the highest round_no among all active interviews for an application.
// Returns 0 if no interviews exist yet.
func (r *InterviewRepo) GetMaxRoundNo(ctx context.Context, applicationID int64) (int32, error) {
	var maxRound sql.NullInt32
	err := r.db.WithContext(ctx).
		Model(&model.InterviewSchedule{}).
		Where("application_id = ? AND deleted_at IS NULL", applicationID).
		Select("MAX(round_no)").
		Scan(&maxRound).Error
	if err != nil {
		return 0, err
	}
	if maxRound.Valid {
		return maxRound.Int32, nil
	}
	return 0, nil
}

// ListByApplication returns all interviews for an application.
func (r *InterviewRepo) ListByApplication(ctx context.Context, applicationID int64) ([]InterviewWithDetailsRow, error) {
	var rows []InterviewWithDetailsRow
	err := r.baseJoins().
		Where("interview_schedules.application_id = ? AND interview_schedules.deleted_at IS NULL", applicationID).
		Order("interview_schedules.round_no ASC, interview_schedules.created_at ASC").
		Scan(&rows).Error
	return rows, err
}

// ListByInterviewer returns interviews assigned to an interviewer.
func (r *InterviewRepo) ListByInterviewer(ctx context.Context, interviewerID int64, status string) ([]InterviewWithDetailsRow, error) {
	var rows []InterviewWithDetailsRow
	query := r.baseJoins().
		Where("interview_schedules.interviewer_id = ? AND interview_schedules.deleted_at IS NULL", interviewerID)
	if status != "" {
		query = query.Where("interview_schedules.status = ?", status)
	}
	err := query.Order("interview_schedules.scheduled_at DESC, interview_schedules.created_at DESC").
		Scan(&rows).Error
	return rows, err
}

// ListByCandidate returns interviews visible to a candidate (via their user_id in applications).
func (r *InterviewRepo) ListByCandidate(ctx context.Context, userID int64) ([]InterviewWithDetailsRow, error) {
	var rows []InterviewWithDetailsRow
	err := r.baseJoins().
		Where("a.user_id = ? AND interview_schedules.deleted_at IS NULL", userID).
		Where("interview_schedules.status NOT IN (?)", []string{"cancelled"}).
		Order("interview_schedules.scheduled_at DESC, interview_schedules.created_at DESC").
		Scan(&rows).Error
	return rows, err
}

// ── Interview Feedback ─────────────────────────────────────────────────

func (r *InterviewRepo) CreateFeedback(ctx context.Context, f *model.InterviewFeedback) error {
	return r.db.WithContext(ctx).Create(f).Error
}

func (r *InterviewRepo) CreateFeedbackWithTx(ctx context.Context, tx *gorm.DB, f *model.InterviewFeedback) error {
	return tx.WithContext(ctx).Create(f).Error
}

func (r *InterviewRepo) GetFeedbackByInterview(ctx context.Context, interviewID int64) (*model.InterviewFeedback, error) {
	var f model.InterviewFeedback
	err := r.db.WithContext(ctx).
		Where("interview_id = ?", interviewID).
		First(&f).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *InterviewRepo) GetFeedbackByInterviewAndInterviewer(ctx context.Context, interviewID, interviewerID int64) (*model.InterviewFeedback, error) {
	var f model.InterviewFeedback
	err := r.db.WithContext(ctx).
		Where("interview_id = ? AND interviewer_id = ?", interviewID, interviewerID).
		First(&f).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *InterviewRepo) HasFeedback(ctx context.Context, interviewID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.InterviewFeedback{}).
		Where("interview_id = ?", interviewID).
		Count(&count).Error
	return count > 0, err
}

func (r *InterviewRepo) FeedbackExistsByInterviewer(ctx context.Context, interviewID, interviewerID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.InterviewFeedback{}).
		Where("interview_id = ? AND interviewer_id = ?", interviewID, interviewerID).
		Count(&count).Error
	return count > 0, err
}

// ListFeedbackByInterviews returns feedback records for multiple interview IDs in a single query.
func (r *InterviewRepo) ListFeedbackByInterviews(ctx context.Context, interviewIDs []int64) ([]model.InterviewFeedback, error) {
	if len(interviewIDs) == 0 {
		return nil, nil
	}
	var feedbacks []model.InterviewFeedback
	err := r.db.WithContext(ctx).
		Where("interview_id IN ?", interviewIDs).
		Find(&feedbacks).Error
	return feedbacks, err
}

// CancelPendingByApplication cancels all pending/scheduled interviews for an application.
// This is called when the application is rejected or withdrawn.
func (r *InterviewRepo) CancelPendingByApplication(ctx context.Context, tx *gorm.DB, applicationID int64, reason string) error {
	return tx.WithContext(ctx).
		Model(&model.InterviewSchedule{}).
		Where("application_id = ? AND status IN ? AND deleted_at IS NULL", applicationID, []string{"pending", "scheduled"}).
		Updates(map[string]interface{}{
			"status":        "cancelled",
			"cancel_reason": reason,
		}).Error
}

// Transaction wraps a function in a DB transaction.
func (r *InterviewRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}
