package persistence

import (
	"context"
	"database/sql"
	"strings"
	"time"

	candidatetools "smart-recruit-ai-agent-service/internal/application/candidate_tools"
)

type candidateAIApplicationDetailRow struct {
	ApplicationID int64          `gorm:"column:application_id"`
	JobID         int64          `gorm:"column:job_id"`
	JobTitle      string         `gorm:"column:job_title"`
	Department    sql.NullString `gorm:"column:department"`
	Location      sql.NullString `gorm:"column:location"`
	SalaryRange   sql.NullString `gorm:"column:salary_range"`
	Status        int32          `gorm:"column:status"`
	StatusKey     string         `gorm:"column:status_key"`
	RoundNo       int32          `gorm:"column:round_no"`
	AppliedAt     time.Time      `gorm:"column:applied_at"`
}

type candidateAIJobDetailRow struct {
	JobID        int64          `gorm:"column:job_id"`
	Title        string         `gorm:"column:title"`
	Department   sql.NullString `gorm:"column:department"`
	Location     sql.NullString `gorm:"column:location"`
	SalaryRange  sql.NullString `gorm:"column:salary_range"`
	Description  sql.NullString `gorm:"column:description"`
	Requirements sql.NullString `gorm:"column:requirements"`
	Status       int32          `gorm:"column:status"`
	HasApplied   bool           `gorm:"column:has_applied"`
}

// ListMyApplicationsForAI returns the candidate's applications for AI tools.
func (s *NativeStore) ListMyApplicationsForAI(ctx context.Context, userID int64, limit int32) ([]candidatetools.ApplicationListItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 50
	}
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
	items := make([]candidatetools.ApplicationListItem, 0, len(rows))
	for _, row := range rows {
		appliedAt := ""
		if !row.AppliedAt.IsZero() {
			appliedAt = row.AppliedAt.Format("2006-01-02 15:04")
		}
		items = append(items, candidatetools.ApplicationListItem{
			ApplicationID: row.ApplicationID,
			JobID:         row.JobID,
			JobTitle:      row.JobTitle,
			Status:        row.Status,
			StatusText:    candidateApplicationStatusText(row.StatusKey, row.Status),
			RoundNo:       row.RoundNo,
			AppliedAt:     appliedAt,
		})
	}
	return items, nil
}

// GetMyApplicationDetailForAI returns one application owned by userID.
func (s *NativeStore) GetMyApplicationDetailForAI(ctx context.Context, userID, applicationID int64) (candidatetools.ApplicationDetail, bool, error) {
	var row candidateAIApplicationDetailRow
	err := s.db.WithContext(ctx).Table("applications a").
		Select("a.id AS application_id, a.job_id, COALESCE(j.title, '') AS job_title, j.department, j.location, j.salary_range, a.status, a.status_key, a.round_no, a.applied_at").
		Joins("LEFT JOIN jobs j ON j.id = a.job_id").
		Where("a.id = ? AND a.user_id = ?", applicationID, userID).
		Limit(1).
		Scan(&row).Error
	if err != nil {
		return candidatetools.ApplicationDetail{}, false, err
	}
	if row.ApplicationID == 0 {
		return candidatetools.ApplicationDetail{}, false, nil
	}
	return candidatetools.ApplicationDetail{
		ApplicationID: row.ApplicationID,
		JobID:         row.JobID,
		JobTitle:      row.JobTitle,
		Department:    nullString(row.Department),
		Location:      nullString(row.Location),
		SalaryRange:   nullString(row.SalaryRange),
		Status:        row.Status,
		StatusText:    candidateApplicationStatusText(row.StatusKey, row.Status),
		RoundNo:       row.RoundNo,
		AppliedAt:     formatTime(row.AppliedAt),
	}, true, nil
}

// GetMyResumeTextForAI returns full resume parsed text for the candidate.
func (s *NativeStore) GetMyResumeTextForAI(ctx context.Context, userID int64) (candidatetools.ResumeText, error) {
	var row candidateResumeReadRow
	err := s.db.WithContext(ctx).Table("resumes").
		Select("id AS resume_id, file_name, parsed_text").
		Where("user_id = ? AND is_valid = ?", userID, 1).
		Order("uploaded_at DESC, id DESC").
		Limit(1).
		Scan(&row).Error
	if err != nil {
		return candidatetools.ResumeText{}, err
	}
	if row.ResumeID == 0 {
		return candidatetools.ResumeText{
			Available: false,
			Message:   "你还没有上传简历。请先上传简历后，我才能基于你的简历内容提供岗位推荐和优化建议。",
		}, nil
	}
	text := strings.TrimSpace(nullString(row.ParsedText))
	if text == "" {
		return candidatetools.ResumeText{
			Available: false,
			FileName:  row.FileName,
			Message:   "你的简历已上传，但解析文本暂不可用。请尝试重新上传简历，或等待系统完成解析后再使用此功能。",
		}, nil
	}
	if len([]rune(text)) < 20 {
		return candidatetools.ResumeText{
			Available: false,
			FileName:  row.FileName,
			Message:   "简历解析文本内容过短，可能无法提供有效的分析和推荐。请检查上传的简历文件是否完整。",
		}, nil
	}
	return candidatetools.ResumeText{
		Available:  true,
		FileName:   row.FileName,
		TextLength: len([]rune(text)),
		ResumeText: text,
	}, nil
}

// ListJobsForCandidateAI lists open jobs with has_applied mark for the candidate.
func (s *NativeStore) ListJobsForCandidateAI(ctx context.Context, userID int64, limit int32) ([]candidatetools.JobListItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	var rows []candidateJobReadRow
	err := s.db.WithContext(ctx).Table("jobs j").
		Select("j.id AS job_id, j.title, j.department, j.location, j.salary_range, j.status, CASE WHEN a.id IS NULL THEN false ELSE true END AS has_applied").
		Joins("LEFT JOIN applications a ON a.job_id = j.id AND a.user_id = ? AND a.is_current = 1", userID).
		Where("j.status = ?", 1).
		Order("j.created_at DESC, j.id DESC").
		Limit(int(limit)).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	items := make([]candidatetools.JobListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, candidatetools.JobListItem{
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

// GetJobDetailForCandidateAI returns job detail with has_applied for the candidate.
func (s *NativeStore) GetJobDetailForCandidateAI(ctx context.Context, userID, jobID int64) (candidatetools.JobDetail, bool, error) {
	var row candidateAIJobDetailRow
	err := s.db.WithContext(ctx).Table("jobs j").
		Select("j.id AS job_id, j.title, j.department, j.location, j.salary_range, j.description, j.requirements, j.status, CASE WHEN a.id IS NULL THEN false ELSE true END AS has_applied").
		Joins("LEFT JOIN applications a ON a.job_id = j.id AND a.user_id = ? AND a.is_current = 1", userID).
		Where("j.id = ?", jobID).
		Limit(1).
		Scan(&row).Error
	if err != nil {
		return candidatetools.JobDetail{}, false, err
	}
	if row.JobID == 0 {
		return candidatetools.JobDetail{}, false, nil
	}
	return candidatetools.JobDetail{
		JobID:        row.JobID,
		Title:        row.Title,
		Department:   nullString(row.Department),
		Location:     nullString(row.Location),
		SalaryRange:  nullString(row.SalaryRange),
		Description:  nullString(row.Description),
		Requirements: nullString(row.Requirements),
		Status:       row.Status,
		StatusText:   candidateJobStatusText(row.Status),
		HasApplied:   row.HasApplied,
	}, true, nil
}
