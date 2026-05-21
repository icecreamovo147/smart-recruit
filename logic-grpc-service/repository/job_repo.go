package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/pagination"
)

type JobRepo struct {
	db *gorm.DB
}

func NewJobRepo(db *gorm.DB) *JobRepo {
	return &JobRepo{db: db}
}

func (r *JobRepo) Create(ctx context.Context, job *model.Job) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *JobRepo) UpdateOwned(ctx context.Context, hrID, jobID int64, fields map[string]any) (int64, error) {
	result := r.db.WithContext(ctx).Model(&model.Job{}).Where("id = ? AND hr_id = ?", jobID, hrID).Updates(fields)
	return result.RowsAffected, result.Error
}

func (r *JobRepo) OfflineOwned(ctx context.Context, hrID, jobID int64) (int64, error) {
	result := r.db.WithContext(ctx).Model(&model.Job{}).Where("id = ? AND hr_id = ?", jobID, hrID).Update("status", 0)
	return result.RowsAffected, result.Error
}

func (r *JobRepo) OnlineOwned(ctx context.Context, hrID, jobID int64) (int64, error) {
	result := r.db.WithContext(ctx).Model(&model.Job{}).Where("id = ? AND hr_id = ?", jobID, hrID).Update("status", 1)
	return result.RowsAffected, result.Error
}

func (r *JobRepo) GetByID(ctx context.Context, jobID int64) (*model.Job, error) {
	var job model.Job
	err := r.db.WithContext(ctx).Where("id = ?", jobID).First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &job, err
}

func (r *JobRepo) GetOwned(ctx context.Context, hrID, jobID int64) (*model.Job, error) {
	var job model.Job
	err := r.db.WithContext(ctx).Where("id = ? AND hr_id = ?", jobID, hrID).First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &job, err
}

func (r *JobRepo) BelongsToHR(ctx context.Context, hrID, jobID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Job{}).Where("id = ? AND hr_id = ?", jobID, hrID).Count(&count).Error
	return count > 0, err
}

func (r *JobRepo) ListByHR(ctx context.Context, hrID int64, page, pageSize int32) ([]model.Job, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&model.Job{}).Where("hr_id = ?", hrID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var jobs []model.Job
	err := query.Order("created_at DESC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Find(&jobs).Error
	return jobs, total, err
}

func (r *JobRepo) SearchByHR(ctx context.Context, hrID int64, keyword string, status *int32, page, pageSize int32) ([]model.Job, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Job{}).Where("hr_id = ?", hrID)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR department LIKE ? OR location LIKE ? OR description LIKE ? OR requirements LIKE ?", like, like, like, like, like)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var jobs []model.Job
	err := query.Order("created_at DESC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Find(&jobs).Error
	return jobs, total, err
}

func (r *JobRepo) ListPublic(ctx context.Context, keyword string, page, pageSize int32) ([]model.Job, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Job{}).Where("status = ?", 1)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR department LIKE ? OR location LIKE ?", like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var jobs []model.Job
	err := query.Order("created_at DESC").Offset(offset(page, pageSize)).Limit(int(pageSize)).Find(&jobs).Error
	return jobs, total, err
}

// ListPublicCursor returns jobs using cursor-based pagination.
// Returns the result slice, next_cursor, has_more, and error.
func (r *JobRepo) ListPublicCursor(ctx context.Context, keyword string, cursor string, limit int32) ([]model.Job, string, bool, error) {
	t, id, err := pagination.DecodeCursor(cursor)
	if err != nil {
		return nil, "", false, err
	}
	query := r.db.WithContext(ctx).Model(&model.Job{}).Where("status = ?", 1)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR department LIKE ? OR location LIKE ?", like, like, like)
	}
	// Filter by cursor: rows before the cursor position
	if !t.IsZero() || id > 0 {
		query = query.Where("(created_at, id) < (?, ?)", t, id)
	}
	fetchLimit := int(limit) + 1 // extra row to detect has_more
	var jobs []model.Job
	if err := query.Order("created_at DESC, id DESC").Limit(fetchLimit).Find(&jobs).Error; err != nil {
		return nil, "", false, err
	}
	hasMore := len(jobs) > int(limit)
	if hasMore {
		jobs = jobs[:limit]
	}
	var nextCursor string
	if hasMore && len(jobs) > 0 {
		last := jobs[len(jobs)-1]
		nextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	return jobs, nextCursor, hasMore, nil
}

// ListByHRCursor returns HR-owned jobs using cursor-based pagination.
func (r *JobRepo) ListByHRCursor(ctx context.Context, hrID int64, cursor string, limit int32) ([]model.Job, string, bool, error) {
	t, id, err := pagination.DecodeCursor(cursor)
	if err != nil {
		return nil, "", false, err
	}
	query := r.db.WithContext(ctx).Model(&model.Job{}).Where("hr_id = ?", hrID)
	if !t.IsZero() || id > 0 {
		query = query.Where("(created_at, id) < (?, ?)", t, id)
	}
	fetchLimit := int(limit) + 1
	var jobs []model.Job
	if err := query.Order("created_at DESC, id DESC").Limit(fetchLimit).Find(&jobs).Error; err != nil {
		return nil, "", false, err
	}
	hasMore := len(jobs) > int(limit)
	if hasMore {
		jobs = jobs[:limit]
	}
	var nextCursor string
	if hasMore && len(jobs) > 0 {
		last := jobs[len(jobs)-1]
		nextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	return jobs, nextCursor, hasMore, nil
}

func (r *JobRepo) ApplicationCount(ctx context.Context, jobID int64) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.Application{}).Where("job_id = ?", jobID).Count(&total).Error
	return total, err
}

// BatchApplicationCounts returns a map of job_id → count in a single query.
func (r *JobRepo) BatchApplicationCounts(ctx context.Context, jobIDs []int64) (map[int64]int64, error) {
	if len(jobIDs) == 0 {
		return map[int64]int64{}, nil
	}
	type row struct {
		JobID int64
		Count int64
	}
	var rows []row
	err := r.db.WithContext(ctx).Model(&model.Application{}).
		Select("job_id, COUNT(*) AS count").
		Where("job_id IN ?", jobIDs).
		Group("job_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int64]int64, len(jobIDs))
	for _, r := range rows {
		result[r.JobID] = r.Count
	}
	return result, nil
}

func offset(page, pageSize int32) int {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	return int((page - 1) * pageSize)
}

// JobWithApplicationMark is a job row with a has_applied flag for a specific user.
type JobWithApplicationMark struct {
	model.Job
	HasApplied bool
}

// ListForCandidateWithApplicationMark returns all jobs with a flag indicating
// whether the given user has applied to each job.
func (r *JobRepo) ListForCandidateWithApplicationMark(ctx context.Context, userID int64) ([]JobWithApplicationMark, error) {
	var rows []JobWithApplicationMark
	err := r.db.WithContext(ctx).Raw(`
		SELECT j.*, CASE WHEN a.id IS NOT NULL THEN true ELSE false END AS has_applied
		FROM jobs j
		LEFT JOIN applications a ON a.job_id = j.id AND a.user_id = ? AND a.is_current = 1
		WHERE j.status = 1
		ORDER BY j.status DESC, j.updated_at DESC`, userID).Scan(&rows).Error
	return rows, err
}

// LookupDepartment returns a department by id for populating job text snapshots.
func (r *JobRepo) LookupDepartment(ctx context.Context, id int64) (*model.Department, error) {
	var dep model.Department
	err := r.db.WithContext(ctx).Where("id = ? AND is_active = 1", id).First(&dep).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &dep, err
}

// LookupLocation returns a job location by id for populating job text snapshots.
func (r *JobRepo) LookupLocation(ctx context.Context, id int64) (*model.JobLocation, error) {
	var loc model.JobLocation
	err := r.db.WithContext(ctx).Where("id = ? AND is_active = 1", id).First(&loc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &loc, err
}

// GetForCandidate returns a job detail with a has_applied flag for a candidate user.
// Only returns jobs with status=1 (open). Returns nil job for offline jobs.
func (r *JobRepo) GetForCandidate(ctx context.Context, userID, jobID int64) (*model.Job, bool, error) {
	var hasApplied bool
	err := r.db.WithContext(ctx).Raw(`
		SELECT EXISTS(SELECT 1 FROM applications WHERE job_id = ? AND user_id = ? AND is_current = 1)`,
		jobID, userID).Scan(&hasApplied).Error
	if err != nil {
		return nil, false, err
	}
	job, err := r.GetByID(ctx, jobID)
	if err != nil {
		return nil, false, err
	}
	if job == nil || job.Status != 1 {
		return nil, hasApplied, nil
	}
	return job, hasApplied, nil
}
