package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/pagination"
)

type ApplicationRepo struct {
	db *gorm.DB
}

type MyApplicationRow struct {
	ApplicationID int64
	JobID         int64
	JobTitle      string
	Status        int32
	RoundNo       int32
	IsCurrent     int32
	AppliedAt     time.Time
}

type JobApplicationRow struct {
	ApplicationID int64
	UserID        int64
	JobID         int64
	JobTitle      string
	RealName      string
	Phone         string
	Education     string
	School        string
	Skills        string
	AppliedAt     time.Time
	OSSKey        string
	FileName      string
	FileType      string
	Status        int32
	RoundNo       int32
	IsCurrent     int32
}

type ApplicationDetailRow struct {
	ApplicationID  int64
	UserID         int64
	ResumeID       int64
	Status         int32
	RoundNo        int32
	IsCurrent      int32
	AppliedAt      time.Time
	RealName       string
	Phone          string
	Education      string
	School         string
	WorkExperience string
	Skills         string
	JobID          int64
	JobTitle       string
	Department     string
	Location       string
	SalaryRange    string
	Description    string
	Requirements   string
	FileName       string
	FileType       string
	FileSize       int64
	OSSKey         string
	ParsedText     string
}

type HotJobRow struct {
	Title string
	Total int64
}

type ApplicationStatusCountRow struct {
	Status int32
	Total  int64
}

type ApplicationTrendRow struct {
	Date  string
	Total int64
}

type ApplicationListFilter struct {
	JobID       int64
	Status      *int32
	CurrentOnly bool
}

type CandidateApplicationOptionRow struct {
	ApplicationID int64
	UserID        int64
	RealName      string
	Phone         string
	JobTitle      string
	Status        int32
	RoundNo       int32
	IsCurrent     int32
	AppliedAt     time.Time
}

func NewApplicationRepo(db *gorm.DB) *ApplicationRepo {
	return &ApplicationRepo{db: db}
}

func (r *ApplicationRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *ApplicationRepo) ExistsActive(ctx context.Context, userID, jobID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Application{}).
		Where("user_id = ? AND job_id = ? AND is_current = ? AND status <> ?", userID, jobID, 1, 3).
		Count(&count).Error
	return count > 0, err
}

func (r *ApplicationRepo) CreateNewRound(ctx context.Context, application *model.Application) error {
	return r.Transaction(ctx, func(tx *gorm.DB) error {
		return r.CreateNewRoundWithTx(ctx, tx, application)
	})
}

func (r *ApplicationRepo) CreateNewRoundWithTx(ctx context.Context, tx *gorm.DB, application *model.Application) error {
	var activeCount int64
	err := tx.WithContext(ctx).Model(&model.Application{}).
		Where("user_id = ? AND job_id = ? AND is_current = ? AND status <> ?", application.UserID, application.JobID, 1, 3).
		Count(&activeCount).Error
	if err != nil {
		return err
	}
	if activeCount > 0 {
		return gorm.ErrDuplicatedKey
	}

	var existingRounds int64
	err = tx.WithContext(ctx).Model(&model.Application{}).
		Where("user_id = ? AND job_id = ?", application.UserID, application.JobID).
		Count(&existingRounds).Error
	if err != nil {
		return err
	}

	err = tx.WithContext(ctx).Model(&model.Application{}).
		Where("user_id = ? AND job_id = ? AND is_current = ? AND status = ?", application.UserID, application.JobID, 1, 3).
		Update("is_current", 0).Error
	if err != nil {
		return err
	}

	application.RoundNo = int32(existingRounds) + 1
	application.IsCurrent = 1
	return tx.WithContext(ctx).Create(application).Error
}

func (r *ApplicationRepo) Create(ctx context.Context, application *model.Application) error {
	err := r.db.WithContext(ctx).Create(application).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return gorm.ErrDuplicatedKey
	}
	return err
}

func (r *ApplicationRepo) ListMy(ctx context.Context, userID int64, page, pageSize int32) ([]MyApplicationRow, int64, error) {
	var total int64
	base := r.db.WithContext(ctx).Table("applications").Where("applications.user_id = ?", userID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []MyApplicationRow
	err := base.Select("applications.id AS application_id, jobs.id AS job_id, jobs.title AS job_title, applications.status, applications.round_no, applications.is_current, applications.applied_at").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Order("applications.applied_at DESC").
		Offset(offset(page, pageSize)).
		Limit(int(pageSize)).
		Scan(&rows).Error
	return rows, total, err
}

// ListMyCursor returns candidate applications using cursor-based pagination.
func (r *ApplicationRepo) ListMyCursor(ctx context.Context, userID int64, cursor string, limit int32) ([]MyApplicationRow, string, bool, error) {
	t, id, err := pagination.DecodeCursor(cursor)
	if err != nil {
		return nil, "", false, err
	}
	query := r.db.WithContext(ctx).Table("applications").
		Select("applications.id AS application_id, jobs.id AS job_id, jobs.title AS job_title, applications.status, applications.round_no, applications.is_current, applications.applied_at").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("applications.user_id = ?", userID)
	if !t.IsZero() || id > 0 {
		query = query.Where("(applications.applied_at, applications.id) < (?, ?)", t, id)
	}
	fetchLimit := int(limit) + 1
	var rows []MyApplicationRow
	if err := query.Order("applications.applied_at DESC, applications.id DESC").Limit(fetchLimit).Scan(&rows).Error; err != nil {
		return nil, "", false, err
	}
	hasMore := len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}
	var nextCursor string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextCursor = pagination.EncodeCursor(last.AppliedAt, last.ApplicationID)
	}
	return rows, nextCursor, hasMore, nil
}

func (r *ApplicationRepo) ListByJob(ctx context.Context, jobID int64, page, pageSize int32) ([]JobApplicationRow, int64, error) {
	var total int64
	latestRound := r.db.WithContext(ctx).Table("applications AS a2").
		Select("MAX(a2.round_no)").
		Where("a2.user_id = applications.user_id AND a2.job_id = applications.job_id")
	base := r.db.WithContext(ctx).Table("applications").Where("applications.job_id = ? AND applications.round_no = (?)", jobID, latestRound)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []JobApplicationRow
	err := base.Select(`applications.id AS application_id, applications.user_id, jobs.id AS job_id, jobs.title AS job_title, candidate_profiles.real_name,
		candidate_profiles.phone, candidate_profiles.education, candidate_profiles.school, candidate_profiles.skills,
		applications.applied_at, resumes.oss_key, resumes.file_name, resumes.file_type, applications.status, applications.round_no, applications.is_current`).
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Joins("LEFT JOIN candidate_profiles ON candidate_profiles.user_id = applications.user_id").
		Joins("LEFT JOIN resumes ON resumes.user_id = applications.user_id AND resumes.is_valid = 1").
		Order("applications.applied_at DESC").
		Offset(offset(page, pageSize)).
		Limit(int(pageSize)).
		Scan(&rows).Error
	return rows, total, err
}

func (r *ApplicationRepo) UpdateStatusOwned(ctx context.Context, hrID, applicationID int64, status int32) (int64, error) {
	return r.UpdateStatusOwnedWithTx(ctx, r.db, hrID, applicationID, status)
}

func (r *ApplicationRepo) UpdateStatusOwnedWithTx(ctx context.Context, tx *gorm.DB, hrID, applicationID int64, status int32) (int64, error) {
	ownedJobIDs := tx.WithContext(ctx).Model(&model.Job{}).Select("id").Where("hr_id = ?", hrID)
	result := tx.WithContext(ctx).Model(&model.Application{}).
		Where("id = ? AND job_id IN (?)", applicationID, ownedJobIDs).
		Update("status", status)
	return result.RowsAffected, result.Error
}

func (r *ApplicationRepo) GetDetailOwned(ctx context.Context, hrID, applicationID int64) (*ApplicationDetailRow, error) {
	var row ApplicationDetailRow
	result := r.db.WithContext(ctx).Table("applications").
		Select(`applications.id AS application_id, applications.user_id, applications.resume_id,
			applications.status, applications.round_no, applications.is_current, applications.applied_at,
			candidate_profiles.real_name, candidate_profiles.phone, candidate_profiles.education,
			candidate_profiles.school, candidate_profiles.work_experience, candidate_profiles.skills,
			jobs.id AS job_id, jobs.title AS job_title, jobs.department, jobs.location,
			jobs.salary_range, jobs.description, jobs.requirements,
			resumes.file_name, resumes.file_type, resumes.file_size, resumes.oss_key, resumes.parsed_text`).
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Joins("LEFT JOIN candidate_profiles ON candidate_profiles.user_id = applications.user_id").
		Joins("LEFT JOIN resumes ON resumes.user_id = applications.user_id AND resumes.is_valid = 1").
		Where("applications.id = ? AND jobs.hr_id = ?", applicationID, hrID).
		Scan(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &row, nil
}

func (r *ApplicationRepo) ListAllByHR(ctx context.Context, hrID int64, page, pageSize int32) ([]JobApplicationRow, int64, error) {
	var total int64
	base := r.db.WithContext(ctx).Table("applications").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("jobs.hr_id = ?", hrID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []JobApplicationRow
	err := base.Select(`applications.id AS application_id, applications.user_id, jobs.id AS job_id, jobs.title AS job_title, candidate_profiles.real_name,
		candidate_profiles.phone, candidate_profiles.education, candidate_profiles.school, candidate_profiles.skills,
		applications.applied_at, resumes.oss_key, resumes.file_name, resumes.file_type, applications.status, applications.round_no, applications.is_current`).
		Joins("LEFT JOIN candidate_profiles ON candidate_profiles.user_id = applications.user_id").
		Joins("LEFT JOIN resumes ON resumes.user_id = applications.user_id AND resumes.is_valid = 1").
		Order("applications.applied_at DESC").
		Offset(offset(page, pageSize)).
		Limit(int(pageSize)).
		Scan(&rows).Error
	return rows, total, err
}

func (r *ApplicationRepo) TotalByHR(ctx context.Context, hrID int64) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Table("applications").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("jobs.hr_id = ?", hrID).
		Count(&total).Error
	return total, err
}

func (r *ApplicationRepo) TodayByHR(ctx context.Context, hrID, jobID int64) (int64, error) {
	start := startOfLocalDay(time.Now())
	var total int64
	query := r.db.WithContext(ctx).Table("applications").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("jobs.hr_id = ? AND applications.applied_at >= ?", hrID, start)
	if jobID > 0 {
		query = query.Where("applications.job_id = ?", jobID)
	}
	err := query.Count(&total).Error
	return total, err
}

func (r *ApplicationRepo) HotJobs(ctx context.Context, hrID int64, limit int) ([]HotJobRow, error) {
	var rows []HotJobRow
	err := r.db.WithContext(ctx).Table("applications").
		Select("jobs.title, COUNT(applications.id) AS total").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("jobs.hr_id = ?", hrID).
		Group("jobs.id, jobs.title").
		Order("total DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (r *ApplicationRepo) ListByHRFiltered(ctx context.Context, hrID int64, filter ApplicationListFilter, page, pageSize int32) ([]JobApplicationRow, int64, error) {
	var total int64
	base := r.db.WithContext(ctx).Table("applications").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("jobs.hr_id = ?", hrID)
	if filter.JobID > 0 {
		base = base.Where("applications.job_id = ?", filter.JobID)
	}
	if filter.Status != nil {
		base = base.Where("applications.status = ?", *filter.Status)
	}
	if filter.CurrentOnly {
		base = base.Where("applications.is_current = ?", 1)
	}
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []JobApplicationRow
	err := base.Select(`applications.id AS application_id, applications.user_id,
			jobs.id AS job_id, jobs.title AS job_title,
			candidate_profiles.real_name, candidate_profiles.phone, candidate_profiles.education,
			candidate_profiles.school, candidate_profiles.skills,
			applications.applied_at, resumes.oss_key, resumes.file_name, resumes.file_type,
			applications.status, applications.round_no, applications.is_current`).
		Joins("LEFT JOIN candidate_profiles ON candidate_profiles.user_id = applications.user_id").
		Joins("LEFT JOIN resumes ON resumes.user_id = applications.user_id AND resumes.is_valid = 1").
		Order("applications.applied_at DESC").
		Offset(offset(page, pageSize)).
		Limit(int(pageSize)).
		Scan(&rows).Error
	return rows, total, err
}

func (r *ApplicationRepo) StatusSummaryByHR(ctx context.Context, hrID, jobID int64) ([]ApplicationStatusCountRow, error) {
	var rows []ApplicationStatusCountRow
	query := r.db.WithContext(ctx).Table("applications").
		Select("applications.status, COUNT(applications.id) AS total").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("jobs.hr_id = ?", hrID)
	if jobID > 0 {
		query = query.Where("applications.job_id = ?", jobID)
	}
	err := query.Group("applications.status").Order("applications.status ASC").Scan(&rows).Error
	return rows, err
}

func (r *ApplicationRepo) TrendByHR(ctx context.Context, hrID, jobID int64, days int) ([]ApplicationTrendRow, error) {
	if days < 1 {
		days = 7
	}
	start := startOfLocalDay(time.Now()).AddDate(0, 0, -days+1)
	var rows []ApplicationTrendRow
	query := r.db.WithContext(ctx).Table("applications").
		Select("DATE_FORMAT(applications.applied_at, '%Y-%m-%d') AS date, COUNT(applications.id) AS total").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("jobs.hr_id = ? AND applications.applied_at >= ?", hrID, start)
	if jobID > 0 {
		query = query.Where("applications.job_id = ?", jobID)
	}
	err := query.Group("date").Order("date ASC").Scan(&rows).Error
	return rows, err
}

func startOfLocalDay(t time.Time) time.Time {
	y, m, d := t.In(time.Local).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

func (r *ApplicationRepo) SearchCandidateApplications(ctx context.Context, hrID int64, keyword string, limit int) ([]CandidateApplicationOptionRow, error) {
	var rows []CandidateApplicationOptionRow
	like := "%" + keyword + "%"
	err := r.db.WithContext(ctx).Table("applications").
		Select(`applications.id AS application_id, applications.user_id, candidate_profiles.real_name,
			candidate_profiles.phone, jobs.title AS job_title, applications.status,
			applications.round_no, applications.is_current, applications.applied_at`).
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Joins("LEFT JOIN candidate_profiles ON candidate_profiles.user_id = applications.user_id").
		Where("jobs.hr_id = ?", hrID).
		Where("(candidate_profiles.real_name = ? OR candidate_profiles.phone LIKE ? OR jobs.title LIKE ?)", keyword, like, like).
		Order("applications.is_current DESC, applications.applied_at DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

// GetDetailByUser returns an application detail row owned by a candidate user.
func (r *ApplicationRepo) GetDetailByUser(ctx context.Context, userID, applicationID int64) (*ApplicationDetailRow, error) {
	var row ApplicationDetailRow
	err := r.db.WithContext(ctx).Table("applications").
		Select(`applications.id AS application_id, applications.user_id, applications.resume_id,
			applications.status, applications.round_no, applications.is_current, applications.applied_at,
			jobs.id AS job_id, jobs.title AS job_title, jobs.department, jobs.location,
			jobs.salary_range, jobs.description, jobs.requirements`).
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("applications.id = ? AND applications.user_id = ?", applicationID, userID).
		Scan(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}
