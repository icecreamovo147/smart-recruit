package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"

	"smart-recruit-interview-service/internal/domain/model"
	"smart-recruit-interview-service/internal/domain/repository"
)

type txContextKey struct{}

func ContextWithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

func TxFromContext(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(txContextKey{}).(*gorm.DB)
	return tx, ok && tx != nil
}

type InterviewRepository struct {
	db *gorm.DB
}

func NewInterviewRepository(db *gorm.DB) *InterviewRepository {
	return &InterviewRepository{db: db}
}

func (r *InterviewRepository) Create(ctx context.Context, interview *model.Interview) error {
	record := toInterviewRecord(interview)
	if err := r.db.WithContext(ctx).Create(record).Error; err != nil {
		return err
	}
	copyInterviewFields(interview, record)
	return nil
}

func (r *InterviewRepository) Save(ctx context.Context, interview *model.Interview) error {
	return r.db.WithContext(ctx).Save(toInterviewRecord(interview)).Error
}

func (r *InterviewRepository) CreateFeedback(ctx context.Context, feedback *model.Feedback) error {
	record := toFeedbackRecord(feedback)
	if err := r.db.WithContext(ctx).Create(record).Error; err != nil {
		return err
	}
	feedback.ID = record.ID
	feedback.UpdatedAt = record.UpdatedAt
	return nil
}

func (r *InterviewRepository) CancelActiveByApplication(ctx context.Context, applicationID int64, reason string) error {
	if tx, ok := TxFromContext(ctx); ok {
		return cancelActiveByApplication(ctx, tx, applicationID, reason)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return cancelActiveByApplication(ctx, tx, applicationID, reason)
	})
}

func (r *InterviewRepository) FindByID(ctx context.Context, interviewID int64) (*model.Interview, error) {
	var record interviewScheduleRecord
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", interviewID).
		First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return fromInterviewRecord(&record), nil
}

func (r *InterviewRepository) FindDetailsByID(ctx context.Context, interviewID int64) (*repository.InterviewDetails, error) {
	var row interviewWithDetailsRow
	err := r.baseJoins(ctx).
		Where("interview_schedules.id = ? AND interview_schedules.deleted_at IS NULL", interviewID).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return fromDetailsRow(&row), nil
}

func (r *InterviewRepository) MaxRoundNo(ctx context.Context, applicationID int64) (int32, error) {
	var maxRound sql.NullInt32
	err := r.db.WithContext(ctx).
		Model(&interviewScheduleRecord{}).
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

func (r *InterviewRepository) ListByApplication(ctx context.Context, applicationID int64) ([]repository.InterviewDetails, error) {
	var rows []interviewWithDetailsRow
	err := r.baseJoins(ctx).
		Where("interview_schedules.application_id = ? AND interview_schedules.deleted_at IS NULL", applicationID).
		Order("interview_schedules.round_no ASC, interview_schedules.created_at ASC").
		Scan(&rows).Error
	return fromDetailsRows(rows), err
}

func (r *InterviewRepository) ListByInterviewer(ctx context.Context, interviewerID int64, status model.InterviewStatus) ([]repository.InterviewDetails, error) {
	var rows []interviewWithDetailsRow
	query := r.baseJoins(ctx).
		Where("interview_schedules.interviewer_id = ? AND interview_schedules.deleted_at IS NULL", interviewerID)
	if status != "" {
		query = query.Where("interview_schedules.status = ?", string(status))
	}
	err := query.Order("interview_schedules.scheduled_at DESC, interview_schedules.created_at DESC").
		Scan(&rows).Error
	return fromDetailsRows(rows), err
}

func (r *InterviewRepository) ListByCandidate(ctx context.Context, candidateUserID int64) ([]repository.InterviewDetails, error) {
	var rows []interviewWithDetailsRow
	err := r.baseJoins(ctx).
		Where("a.user_id = ? AND interview_schedules.deleted_at IS NULL", candidateUserID).
		Where("interview_schedules.status NOT IN (?)", []string{string(model.InterviewStatusCancelled)}).
		Order("interview_schedules.scheduled_at DESC, interview_schedules.created_at DESC").
		Scan(&rows).Error
	return fromDetailsRows(rows), err
}

func (r *InterviewRepository) ListFeedbackByInterviews(ctx context.Context, interviewIDs []int64) ([]model.Feedback, error) {
	if len(interviewIDs) == 0 {
		return nil, nil
	}
	var records []interviewFeedbackRecord
	err := r.db.WithContext(ctx).
		Where("interview_id IN ?", interviewIDs).
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	feedbacks := make([]model.Feedback, 0, len(records))
	for i := range records {
		feedbacks = append(feedbacks, *fromFeedbackRecord(&records[i]))
	}
	return feedbacks, nil
}

func (r *InterviewRepository) FeedbackExistsByInterviewer(ctx context.Context, interviewID int64, interviewerID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&interviewFeedbackRecord{}).
		Where("interview_id = ? AND interviewer_id = ?", interviewID, interviewerID).
		Count(&count).Error
	return count > 0, err
}

func (r *InterviewRepository) FindFeedbackByInterviewAndInterviewer(ctx context.Context, interviewID int64, interviewerID int64) (*model.Feedback, error) {
	var record interviewFeedbackRecord
	err := r.db.WithContext(ctx).
		Where("interview_id = ? AND interviewer_id = ?", interviewID, interviewerID).
		First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return fromFeedbackRecord(&record), nil
}

func (r *InterviewRepository) Transaction(ctx context.Context, fn func(context.Context, repository.InterviewWriter) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := ContextWithTx(ctx, tx)
		return fn(txCtx, txWriter{tx: tx})
	})
}

func (r *InterviewRepository) baseJoins(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table("interview_schedules").
		Select(baseInterviewSelect()).
		Joins("JOIN users u ON u.id = interview_schedules.interviewer_id").
		Joins("JOIN applications a ON a.id = interview_schedules.application_id AND a.tenant_id = interview_schedules.tenant_id").
		Joins("JOIN jobs j ON j.id = a.job_id AND j.tenant_id = interview_schedules.tenant_id").
		Joins("LEFT JOIN candidate_profiles cp ON cp.user_id = a.user_id").
		Joins("LEFT JOIN resumes res ON res.user_id = a.user_id AND res.id = (SELECT MAX(r2.id) FROM resumes r2 WHERE r2.user_id = a.user_id)")
}

type txWriter struct {
	tx *gorm.DB
}

func (w txWriter) Create(ctx context.Context, interview *model.Interview) error {
	record := toInterviewRecord(interview)
	if err := w.tx.WithContext(ctx).Create(record).Error; err != nil {
		return err
	}
	copyInterviewFields(interview, record)
	return nil
}

func (w txWriter) Save(ctx context.Context, interview *model.Interview) error {
	return w.tx.WithContext(ctx).Save(toInterviewRecord(interview)).Error
}

func (w txWriter) CreateFeedback(ctx context.Context, feedback *model.Feedback) error {
	record := toFeedbackRecord(feedback)
	if err := w.tx.WithContext(ctx).Create(record).Error; err != nil {
		return err
	}
	feedback.ID = record.ID
	feedback.UpdatedAt = record.UpdatedAt
	return nil
}

func (w txWriter) CancelActiveByApplication(ctx context.Context, applicationID int64, reason string) error {
	return cancelActiveByApplication(ctx, w.tx, applicationID, reason)
}

func cancelActiveByApplication(ctx context.Context, tx *gorm.DB, applicationID int64, reason string) error {
	return tx.WithContext(ctx).
		Model(&interviewScheduleRecord{}).
		Where("application_id = ? AND status IN ? AND deleted_at IS NULL", applicationID, []string{string(model.InterviewStatusPending), string(model.InterviewStatusScheduled)}).
		Updates(map[string]interface{}{
			"status":        string(model.InterviewStatusCancelled),
			"cancel_reason": reason,
		}).Error
}

func baseInterviewSelect() string {
	return `interview_schedules.id, interview_schedules.tenant_id, interview_schedules.application_id, interview_schedules.interviewer_id,
		interview_schedules.round_no, interview_schedules.title, interview_schedules.mode,
		interview_schedules.meeting_url, interview_schedules.location, interview_schedules.duration_minutes,
		interview_schedules.candidate_note, interview_schedules.internal_note, interview_schedules.cancel_reason,
		interview_schedules.scheduled_at, interview_schedules.status, interview_schedules.created_by,
		interview_schedules.created_at, interview_schedules.updated_at,
		u.username AS interviewer_name,
		a.user_id AS candidate_user_id,
		a.status_key AS application_status_key,
		j.title AS job_title,
		COALESCE(cp.real_name, CONCAT('候选人', a.user_id)) AS candidate_name,
		COALESCE(cp.phone, '') AS candidate_phone,
		COALESCE(res.oss_key, '') AS resume_oss_key`
}

type interviewScheduleRecord struct {
	ID              int64      `gorm:"primaryKey"`
	TenantID        int64      `gorm:"column:tenant_id"`
	ApplicationID   int64      `gorm:"column:application_id;not null"`
	InterviewerID   int64      `gorm:"column:interviewer_id;not null"`
	RoundNo         int32      `gorm:"column:round_no;default:1"`
	Title           string     `gorm:"column:title"`
	Mode            string     `gorm:"column:mode"`
	MeetingURL      string     `gorm:"column:meeting_url"`
	Location        string     `gorm:"column:location"`
	DurationMinutes int32      `gorm:"column:duration_minutes"`
	CandidateNote   string     `gorm:"column:candidate_note"`
	InternalNote    string     `gorm:"column:internal_note"`
	CancelReason    string     `gorm:"column:cancel_reason"`
	ScheduledAt     *time.Time `gorm:"column:scheduled_at"`
	Status          string     `gorm:"column:status"`
	CreatedBy       *int64     `gorm:"column:created_by"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
	DeletedAt       *time.Time `gorm:"column:deleted_at"`
}

func (interviewScheduleRecord) TableName() string { return "interview_schedules" }

type interviewFeedbackRecord struct {
	ID                  int64     `gorm:"primaryKey"`
	TenantID            int64     `gorm:"column:tenant_id"`
	InterviewID         int64     `gorm:"column:interview_id"`
	ApplicationID       int64     `gorm:"column:application_id"`
	InterviewerID       int64     `gorm:"column:interviewer_id"`
	Recommendation      string    `gorm:"column:recommendation"`
	Score               int32     `gorm:"column:score"`
	DimensionScoresJSON string    `gorm:"column:dimension_scores_json"`
	Comments            string    `gorm:"column:comments"`
	SubmittedAt         time.Time `gorm:"column:submitted_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at"`
}

func (interviewFeedbackRecord) TableName() string { return "interview_feedback" }

type interviewWithDetailsRow struct {
	ID              int64
	TenantID        int64
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

	InterviewerName      string
	ApplicationStatusKey string
	JobTitle             string
	CandidateUserID      int64
	CandidateName        string
	CandidatePhone       string
	ResumeOssKey         string
}

func toInterviewRecord(interview *model.Interview) *interviewScheduleRecord {
	if interview == nil {
		return nil
	}
	return &interviewScheduleRecord{
		ID:              interview.ID,
		TenantID:        interview.TenantID,
		ApplicationID:   interview.ApplicationID,
		InterviewerID:   interview.InterviewerID,
		RoundNo:         interview.RoundNo,
		ScheduledAt:     interview.ScheduledAt,
		Status:          string(interview.Status),
		CreatedBy:       interview.CreatedBy,
		Title:           interview.Title,
		Mode:            interview.Mode,
		MeetingURL:      interview.MeetingURL,
		Location:        interview.Location,
		DurationMinutes: interview.DurationMinutes,
		CandidateNote:   interview.CandidateNote,
		InternalNote:    interview.InternalNote,
		CancelReason:    interview.CancelReason,
		CreatedAt:       interview.CreatedAt,
		UpdatedAt:       interview.UpdatedAt,
	}
}

func fromInterviewRecord(interview *interviewScheduleRecord) *model.Interview {
	if interview == nil {
		return nil
	}
	return &model.Interview{
		ID:              interview.ID,
		TenantID:        interview.TenantID,
		ApplicationID:   interview.ApplicationID,
		InterviewerID:   interview.InterviewerID,
		RoundNo:         interview.RoundNo,
		ScheduledAt:     interview.ScheduledAt,
		Status:          model.InterviewStatus(interview.Status),
		CreatedBy:       interview.CreatedBy,
		Title:           interview.Title,
		Mode:            interview.Mode,
		MeetingURL:      interview.MeetingURL,
		Location:        interview.Location,
		DurationMinutes: interview.DurationMinutes,
		CandidateNote:   interview.CandidateNote,
		InternalNote:    interview.InternalNote,
		CancelReason:    interview.CancelReason,
		CreatedAt:       interview.CreatedAt,
		UpdatedAt:       interview.UpdatedAt,
	}
}

func copyInterviewFields(target *model.Interview, source *interviewScheduleRecord) {
	target.ID = source.ID
	target.TenantID = source.TenantID
	target.CreatedAt = source.CreatedAt
	target.UpdatedAt = source.UpdatedAt
}

func fromDetailsRow(row *interviewWithDetailsRow) *repository.InterviewDetails {
	if row == nil {
		return nil
	}
	return &repository.InterviewDetails{
		Interview:             *fromInterviewRecord(row.toInterviewRecord()),
		ApplicationStatusKey:  model.ApplicationStatus(row.ApplicationStatusKey),
		JobTitle:              row.JobTitle,
		CandidateUserID:       row.CandidateUserID,
		CandidateName:         row.CandidateName,
		CandidatePhone:        row.CandidatePhone,
		InterviewerName:       row.InterviewerName,
		ResumeOssKey:          row.ResumeOssKey,
		HasFeedbackForRequest: false,
	}
}

func fromDetailsRows(rows []interviewWithDetailsRow) []repository.InterviewDetails {
	details := make([]repository.InterviewDetails, 0, len(rows))
	for i := range rows {
		details = append(details, *fromDetailsRow(&rows[i]))
	}
	return details
}

func (row interviewWithDetailsRow) toInterviewRecord() *interviewScheduleRecord {
	return &interviewScheduleRecord{
		ID:              row.ID,
		TenantID:        row.TenantID,
		ApplicationID:   row.ApplicationID,
		InterviewerID:   row.InterviewerID,
		RoundNo:         row.RoundNo,
		Title:           row.Title,
		Mode:            row.Mode,
		MeetingURL:      row.MeetingURL,
		Location:        row.Location,
		DurationMinutes: row.DurationMinutes,
		CandidateNote:   row.CandidateNote,
		InternalNote:    row.InternalNote,
		CancelReason:    row.CancelReason,
		ScheduledAt:     row.ScheduledAt,
		Status:          row.Status,
		CreatedBy:       row.CreatedBy,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func toFeedbackRecord(feedback *model.Feedback) *interviewFeedbackRecord {
	if feedback == nil {
		return nil
	}
	return &interviewFeedbackRecord{
		ID:                  feedback.ID,
		InterviewID:         feedback.InterviewID,
		ApplicationID:       feedback.ApplicationID,
		InterviewerID:       feedback.InterviewerID,
		Recommendation:      feedback.Recommendation,
		Score:               feedback.Score,
		DimensionScoresJSON: feedback.DimensionScoresJSON,
		Comments:            feedback.Comments,
		SubmittedAt:         feedback.SubmittedAt,
		UpdatedAt:           feedback.UpdatedAt,
	}
}

func fromFeedbackRecord(feedback *interviewFeedbackRecord) *model.Feedback {
	if feedback == nil {
		return nil
	}
	return &model.Feedback{
		ID:                  feedback.ID,
		InterviewID:         feedback.InterviewID,
		ApplicationID:       feedback.ApplicationID,
		InterviewerID:       feedback.InterviewerID,
		Recommendation:      feedback.Recommendation,
		Score:               feedback.Score,
		DimensionScoresJSON: feedback.DimensionScoresJSON,
		Comments:            feedback.Comments,
		SubmittedAt:         feedback.SubmittedAt,
		UpdatedAt:           feedback.UpdatedAt,
	}
}
