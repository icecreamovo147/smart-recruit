package persistence

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"smart-recruit-analytics-service/internal/domain/model"
	"smart-recruit-platform-go/businessclock"
	platformmetadata "smart-recruit-platform-go/metadata"
)

type ReportingRepository struct {
	db  *gorm.DB
	now func() time.Time
}

func NewReportingRepository(db *gorm.DB) *ReportingRepository {
	return &ReportingRepository{db: db, now: businessclock.Now}
}

func (r *ReportingRepository) GetDashboardKPI(ctx context.Context, filter model.ReportFilter) (model.DashboardKPI, error) {
	jobIDs, err := r.scopeJobIDs(ctx, filter)
	if err != nil {
		return model.DashboardKPI{}, err
	}
	kpi := model.DashboardKPI{}

	jobQuery := r.db.WithContext(ctx).Table("jobs")
	if jobIDs != nil {
		jobQuery = jobQuery.Where("id IN ?", jobIDs)
	}
	if err := jobQuery.Where("status = ?", 1).Count(&kpi.OnlineJobs).Error; err != nil {
		return model.DashboardKPI{}, err
	}
	if err := jobQuery.Where("status = ?", 0).Count(&kpi.OfflineJobs).Error; err != nil {
		return model.DashboardKPI{}, err
	}

	appQuery := r.db.WithContext(ctx).Table("applications").
		Joins("JOIN jobs ON jobs.id = applications.job_id")
	if where, args := scopeFilter(jobIDs, "applications"); where != "" {
		appQuery = appQuery.Where(where, args...)
	}
	if err := appQuery.Count(&kpi.TotalApplications).Error; err != nil {
		return model.DashboardKPI{}, err
	}

	todayStart := businessclock.StartOfDay(r.now())
	if err := appQuery.Where("applications.applied_at >= ?", todayStart).Count(&kpi.TodayApplications).Error; err != nil {
		return model.DashboardKPI{}, err
	}

	pendingQuery := r.db.WithContext(ctx).Table("applications").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("applications.status_key IN ?", []string{"applied", "viewed"}).
		Where("applications.is_current = ?", 1)
	if where, args := scopeFilter(jobIDs, "applications"); where != "" {
		pendingQuery = pendingQuery.Where(where, args...)
	}
	if err := pendingQuery.Count(&kpi.PendingActions).Error; err != nil {
		return model.DashboardKPI{}, err
	}
	return kpi, nil
}

func (r *ReportingRepository) GetUnreadNotificationCount(ctx context.Context, userID uint64, accountType string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("notifications").
		Where("receiver_id = ? AND receiver_account_type = ? AND is_read = ?", int64(userID), accountType, 0).
		Count(&count).Error
	return count, err
}

func (r *ReportingRepository) GetTrend(ctx context.Context, filter model.ReportFilter, days int) ([]model.TrendPoint, error) {
	jobIDs, err := r.scopeJobIDs(ctx, filter)
	if err != nil {
		return nil, err
	}
	start := businessclock.StartOfDay(r.now()).AddDate(0, 0, -days+1)

	query := r.db.WithContext(ctx).Table("applications").
		Select("DATE_FORMAT(applications.applied_at, '%Y-%m-%d') AS date, COUNT(*) AS applications").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("applications.applied_at >= ?", start)
	if where, args := scopeFilter(jobIDs, "applications"); where != "" {
		query = query.Where(where, args...)
	}

	var rows []model.TrendPoint
	if err := query.Group("date").Order("date ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *ReportingRepository) GetStageDistribution(ctx context.Context, filter model.ReportFilter) ([]model.StageCount, error) {
	jobIDs, err := r.scopeJobIDs(ctx, filter)
	if err != nil {
		return nil, err
	}
	query := r.db.WithContext(ctx).Table("applications").
		Select("applications.status_key, COUNT(*) AS count").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("applications.is_current = ?", 1)
	if where, args := scopeFilter(jobIDs, "applications"); where != "" {
		query = query.Where(where, args...)
	}
	var rows []model.StageCount
	if err := query.Group("applications.status_key").Order("count DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *ReportingRepository) GetFunnelReport(ctx context.Context, filter model.ReportFilter) ([]model.StageCount, error) {
	jobIDs, err := r.scopeJobIDs(ctx, filter)
	if err != nil {
		return nil, err
	}
	query := r.db.WithContext(ctx).Table("applications").
		Select("applications.status_key, COUNT(*) AS count").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("applications.is_current = ?", 1)
	if where, args := scopeFilter(jobIDs, "applications"); where != "" {
		query = query.Where(where, args...)
	}
	query = applyReportFilter(query, filter, "applications.applied_at", "applications.job_id")
	var rows []model.StageCount
	if err := query.Group("applications.status_key").Order("count DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *ReportingRepository) GetTimeInStage(ctx context.Context, filter model.ReportFilter) ([]model.StageDurationRow, error) {
	jobIDs, err := r.scopeJobIDs(ctx, filter)
	if err != nil {
		return nil, err
	}
	baseSQL := `
	WITH numbered AS (
	    SELECT *, ROW_NUMBER() OVER (PARTITION BY application_id ORDER BY created_at) AS rn
	    FROM application_status_transitions WHERE tenant_id = ?
	)
	SELECT t.from_status, t.to_status,
	       AVG(TIMESTAMPDIFF(SECOND, t_prev.created_at, t.created_at)) AS avg_duration_secs,
	       COUNT(*) AS transition_count
	FROM numbered t
	JOIN applications a ON a.id = t.application_id AND a.tenant_id = t.tenant_id
	JOIN jobs j ON j.id = a.job_id AND j.tenant_id = t.tenant_id
	LEFT JOIN numbered t_prev ON t_prev.application_id = t.application_id AND t_prev.rn = t.rn - 1
	WHERE t_prev.id IS NOT NULL`

	var conditions []string
	tenantID := platformmetadata.GetAuthTenantID(ctx)
	if tenantID <= 0 {
		return nil, fmt.Errorf("tenant context is required")
	}
	args := []interface{}{tenantID}
	if where, scopeArgs := scopeFilter(jobIDs, "a"); where != "" {
		conditions = append(conditions, where)
		args = append(args, scopeArgs...)
	}
	if filter.StartDate != nil {
		conditions = append(conditions, "t.created_at >= ?")
		args = append(args, *filter.StartDate)
	}
	if filter.EndDate != nil {
		conditions = append(conditions, "t.created_at <= ?")
		args = append(args, *filter.EndDate)
	}
	if filter.JobID > 0 {
		conditions = append(conditions, "a.job_id = ?")
		args = append(args, filter.JobID)
	}

	fullSQL := baseSQL
	if len(conditions) > 0 {
		fullSQL += " AND " + strings.Join(conditions, " AND ")
	}
	fullSQL += " GROUP BY t.from_status, t.to_status"

	var rows []model.StageDurationRow
	if err := r.db.WithContext(ctx).Raw(fullSQL, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *ReportingRepository) GetInterviewMetrics(ctx context.Context, filter model.ReportFilter) (model.InterviewMetrics, error) {
	jobIDs, err := r.scopeJobIDs(ctx, filter)
	if err != nil {
		return model.InterviewMetrics{}, err
	}
	baseQuery := r.db.WithContext(ctx).Table("interview_schedules").
		Select("interview_schedules.id").
		Joins("JOIN applications a ON a.id = interview_schedules.application_id").
		Joins("JOIN jobs j ON j.id = a.job_id")
	if where, args := scopeFilter(jobIDs, "a"); where != "" {
		baseQuery = baseQuery.Where(where, args...)
	}
	baseQuery = applyReportFilter(baseQuery, filter, "interview_schedules.created_at", "a.job_id")

	metrics := model.InterviewMetrics{}
	if err := baseQuery.Count(&metrics.TotalInterviews).Error; err != nil {
		return model.InterviewMetrics{}, err
	}
	if err := baseQuery.Where("interview_schedules.status = ?", "completed").Count(&metrics.CompletedInterviews).Error; err != nil {
		return model.InterviewMetrics{}, err
	}

	feedbackQuery := r.db.WithContext(ctx).Table("interview_feedback").
		Select("interview_feedback.id").
		Joins("JOIN interview_schedules s ON s.id = interview_feedback.interview_id").
		Joins("JOIN applications a ON a.id = s.application_id").
		Joins("JOIN jobs j ON j.id = a.job_id").
		Where("interview_feedback.recommendation = ?", "positive")
	if where, args := scopeFilter(jobIDs, "a"); where != "" {
		feedbackQuery = feedbackQuery.Where(where, args...)
	}
	feedbackQuery = applyReportFilter(feedbackQuery, filter, "s.created_at", "a.job_id")
	if err := feedbackQuery.Count(&metrics.PositiveFeedbacks).Error; err != nil {
		return model.InterviewMetrics{}, err
	}
	return metrics, nil
}

func (r *ReportingRepository) GetOfferMetrics(ctx context.Context, filter model.ReportFilter) (model.OfferMetrics, error) {
	jobIDs, err := r.scopeJobIDs(ctx, filter)
	if err != nil {
		return model.OfferMetrics{}, err
	}
	offerQuery := r.db.WithContext(ctx).Table("offers").
		Select("offers.id").
		Joins("JOIN jobs j ON j.id = offers.job_id")
	if where, args := scopeFilter(jobIDs, "j"); where != "" {
		offerQuery = offerQuery.Where(where, args...)
	}
	offerQuery = applyReportFilter(offerQuery, filter, "offers.created_at", "j.id")

	metrics := model.OfferMetrics{}
	if err := offerQuery.Count(&metrics.TotalOffers).Error; err != nil {
		return model.OfferMetrics{}, err
	}
	if err := offerQuery.Where("offers.status = ?", "accepted").Count(&metrics.AcceptedOffers).Error; err != nil {
		return model.OfferMetrics{}, err
	}
	if err := offerQuery.Where("offers.status = ?", "rejected").Count(&metrics.RejectedOffers).Error; err != nil {
		return model.OfferMetrics{}, err
	}
	return metrics, nil
}

func (r *ReportingRepository) scopeJobIDs(ctx context.Context, filter model.ReportFilter) ([]uint64, error) {
	for _, scope := range filter.ScopeKeys {
		if scope == model.ScopeRecruitingAll || scope == model.ScopeSystemAll {
			return nil, nil
		}
	}
	ids := make(map[uint64]bool)
	for _, scope := range filter.ScopeKeys {
		switch scope {
		case "own_jobs":
			if err := r.addJobIDs(ctx, ids, "hr_id = ?", filter.ActorID); err != nil {
				return nil, err
			}
		case "department":
			if len(filter.DepartmentIDs) > 0 {
				if err := r.addJobIDs(ctx, ids, "department_id IN ?", filter.DepartmentIDs); err != nil {
					return nil, err
				}
			}
		case "location":
			if len(filter.LocationIDs) > 0 {
				if err := r.addJobIDs(ctx, ids, "location_id IN ?", filter.LocationIDs); err != nil {
					return nil, err
				}
			}
		case "assigned_interviews":
			var jobIDs []uint64
			if r.db.Migrator().HasTable("interview_schedules") {
				if err := r.db.WithContext(ctx).Table("interview_schedules").
					Select("DISTINCT a.job_id").
					Joins("JOIN applications a ON a.id = interview_schedules.application_id").
					Where("interview_schedules.interviewer_id = ? AND interview_schedules.deleted_at IS NULL", filter.ActorID).
					Pluck("a.job_id", &jobIDs).Error; err != nil {
					return nil, err
				}
			}
			for _, id := range jobIDs {
				ids[id] = true
			}
		}
	}
	result := make([]uint64, 0, len(ids))
	for id := range ids {
		result = append(result, id)
	}
	return result, nil
}

func (r *ReportingRepository) addJobIDs(ctx context.Context, ids map[uint64]bool, where string, args ...interface{}) error {
	var jobIDs []uint64
	if err := r.db.WithContext(ctx).Table("jobs").Where(where, args...).Pluck("id", &jobIDs).Error; err != nil {
		return err
	}
	for _, id := range jobIDs {
		ids[id] = true
	}
	return nil
}

func scopeFilter(jobIDs []uint64, tableAlias string) (string, []interface{}) {
	if jobIDs == nil {
		return "", nil
	}
	if len(jobIDs) == 0 {
		return "1=0", nil
	}
	return fmt.Sprintf("%s.job_id IN ?", tableAlias), []interface{}{jobIDs}
}

func applyReportFilter(query *gorm.DB, filter model.ReportFilter, dateColumn, jobColumn string) *gorm.DB {
	if filter.StartDate != nil {
		query = query.Where(dateColumn+" >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where(dateColumn+" <= ?", *filter.EndDate)
	}
	if filter.JobID > 0 {
		query = query.Where(jobColumn+" = ?", filter.JobID)
	}
	return query
}
