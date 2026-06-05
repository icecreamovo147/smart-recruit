package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/authz"
)

// AnalyticsRepo provides scope-aware analytics queries for dashboard and reports.
type AnalyticsRepo struct {
	db *gorm.DB
}

func NewAnalyticsRepo(db *gorm.DB) *AnalyticsRepo {
	return &AnalyticsRepo{db: db}
}

// ---- Dashboard helpers ----

// ScopeJobIDs returns the set of job IDs the user can access.
// scopeKeys is the user's active scope keys; if empty or full-access, returns nil (all jobs).
func (r *AnalyticsRepo) ScopeJobIDs(ctx context.Context, userID uint64, scopeKeys []string, deptIDs, locIDs []uint64) ([]uint64, error) {
	// Check for full access
	for _, sk := range scopeKeys {
		if sk == authz.ScopeRecruitingAll || sk == authz.ScopeSystemAll {
			return nil, nil // nil signals "no filter"
		}
	}

	ids := make(map[uint64]bool)
	for _, sk := range scopeKeys {
		switch sk {
		case authz.ScopeOwnJobs:
			var jobIDs []uint64
			if err := r.db.WithContext(ctx).Table("jobs").
				Where("hr_id = ?", userID).
				Pluck("id", &jobIDs).Error; err != nil {
				return nil, err
			}
			for _, id := range jobIDs {
				ids[id] = true
			}

		case authz.ScopeDepartment:
			if len(deptIDs) > 0 {
				var jobIDs []uint64
				if err := r.db.WithContext(ctx).Table("jobs").
					Where("department_id IN ?", deptIDs).
					Pluck("id", &jobIDs).Error; err != nil {
					return nil, err
				}
				for _, id := range jobIDs {
					ids[id] = true
				}
			}

		case authz.ScopeLocation:
			if len(locIDs) > 0 {
				var jobIDs []uint64
				if err := r.db.WithContext(ctx).Table("jobs").
					Where("location_id IN ?", locIDs).
					Pluck("id", &jobIDs).Error; err != nil {
					return nil, err
				}
				for _, id := range jobIDs {
					ids[id] = true
				}
			}

		case authz.ScopeAssignedInterviews:
			var jobIDs []uint64
			if r.db.Migrator().HasTable("interview_schedules") {
				if err := r.db.WithContext(ctx).Table("interview_schedules").
					Select("DISTINCT a.job_id").
					Joins("JOIN applications a ON a.id = interview_schedules.application_id").
					Where("interview_schedules.interviewer_id = ? AND interview_schedules.deleted_at IS NULL", userID).
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

// scopeFilter returns a WHERE clause and args based on job IDs.
// If jobIDs is nil, no filtering is applied (full access).
// Otherwise, filters to the given job IDs.
func (r *AnalyticsRepo) scopeFilter(jobIDs []uint64, tableAlias string) (string, []any) {
	if jobIDs == nil {
		return "", nil
	}
	if len(jobIDs) == 0 {
		return "1=0", nil // no access
	}
	return fmt.Sprintf("%s.job_id IN ?", tableAlias), []any{jobIDs}
}

// ---- Dashboard Summary ----

type DashboardKPI struct {
	OnlineJobs        int64
	OfflineJobs       int64
	TotalApplications int64
	TodayApplications int64
	PendingActions    int64
}

type StageCount struct {
	StageKey string
	Count    int64
}

type TrendPoint struct {
	Date        string
	Applications int64
}

// GetDashboardKPI returns KPI values scoped to the user.
func (r *AnalyticsRepo) GetDashboardKPI(ctx context.Context, userID uint64, scopeKeys []string, deptIDs, locIDs []uint64) (*DashboardKPI, error) {
	jobIDs, err := r.ScopeJobIDs(ctx, userID, scopeKeys, deptIDs, locIDs)
	if err != nil {
		return nil, err
	}

	kpi := &DashboardKPI{}

	// Online/Offline jobs
	jobQuery := r.db.WithContext(ctx).Model(&model.Job{})
	if jobIDs != nil {
		jobQuery = jobQuery.Where("id IN ?", jobIDs)
	}

	var totalOnline, totalOffline int64
	if err := jobQuery.Where("status = ?", 1).Count(&totalOnline).Error; err != nil {
		return nil, err
	}
	if err := jobQuery.Where("status = ?", 0).Count(&totalOffline).Error; err != nil {
		return nil, err
	}
	kpi.OnlineJobs = totalOnline
	kpi.OfflineJobs = totalOffline

	// Application counts
	appQuery := r.db.WithContext(ctx).Table("applications").
		Joins("JOIN jobs ON jobs.id = applications.job_id")
	where, args := r.scopeFilter(jobIDs, "applications")
	if where != "" {
		appQuery = appQuery.Where(where, args...)
	}

	if err := appQuery.Count(&kpi.TotalApplications).Error; err != nil {
		return nil, err
	}

	// Today applications
	todayStart := time.Now().Truncate(24 * time.Hour)
	todayQuery := appQuery.Where("applications.applied_at >= ?", todayStart)
	if err := todayQuery.Count(&kpi.TodayApplications).Error; err != nil {
		return nil, err
	}

	// Pending actions: applications in early stages (applied)
	pendingQuery := r.db.WithContext(ctx).Table("applications").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("applications.status_key IN ?", []string{"applied", "viewed"}).
		Where("applications.is_current = ?", 1)
	where2, args2 := r.scopeFilter(jobIDs, "applications")
	if where2 != "" {
		pendingQuery = pendingQuery.Where(where2, args2...)
	}
	if err := pendingQuery.Count(&kpi.PendingActions).Error; err != nil {
		return nil, err
	}

	return kpi, nil
}

// GetStageDistribution returns application counts per stage.
func (r *AnalyticsRepo) GetStageDistribution(ctx context.Context, userID uint64, scopeKeys []string, deptIDs, locIDs []uint64) ([]StageCount, error) {
	jobIDs, err := r.ScopeJobIDs(ctx, userID, scopeKeys, deptIDs, locIDs)
	if err != nil {
		return nil, err
	}

	query := r.db.WithContext(ctx).Table("applications").
		Select("applications.status_key, COUNT(*) AS count").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("applications.is_current = ?", 1)
	where, args := r.scopeFilter(jobIDs, "applications")
	if where != "" {
		query = query.Where(where, args...)
	}

	var rows []StageCount
	if err := query.Group("applications.status_key").Order("count DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetTrend returns daily application counts for the last N days.
func (r *AnalyticsRepo) GetTrend(ctx context.Context, userID uint64, scopeKeys []string, deptIDs, locIDs []uint64, days int) ([]TrendPoint, error) {
	jobIDs, err := r.ScopeJobIDs(ctx, userID, scopeKeys, deptIDs, locIDs)
	if err != nil {
		return nil, err
	}

	start := time.Now().AddDate(0, 0, -days+1).Truncate(24 * time.Hour)

	query := r.db.WithContext(ctx).Table("applications").
		Select("DATE_FORMAT(applications.applied_at, '%Y-%m-%d') AS date, COUNT(*) AS applications").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("applications.applied_at >= ?", start)
	where, args := r.scopeFilter(jobIDs, "applications")
	if where != "" {
		query = query.Where(where, args...)
	}

	var rows []TrendPoint
	if err := query.Group("date").Order("date ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ---- Funnel Report ----

// GetFunnelReport returns counts at each major funnel stage with conversion rates.
func (r *AnalyticsRepo) GetFunnelReport(ctx context.Context, userID uint64, scopeKeys []string, deptIDs, locIDs []uint64, startDate, endDate *time.Time, jobID int64) ([]StageCount, error) {
	jobIDs, err := r.ScopeJobIDs(ctx, userID, scopeKeys, deptIDs, locIDs)
	if err != nil {
		return nil, err
	}

	query := r.db.WithContext(ctx).Table("applications").
		Select("applications.status_key, COUNT(*) AS count").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("applications.is_current = ?", 1)
	where, args := r.scopeFilter(jobIDs, "applications")
	if where != "" {
		query = query.Where(where, args...)
	}
	if startDate != nil {
		query = query.Where("applications.applied_at >= ?", startDate)
	}
	if endDate != nil {
		query = query.Where("applications.applied_at <= ?", endDate)
	}
	if jobID > 0 {
		query = query.Where("applications.job_id = ?", jobID)
	}

	var rows []StageCount
	if err := query.Group("applications.status_key").Order("count DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ---- Time-in-Stage Report ----

type StageDurationRow struct {
	FromStatus      string
	ToStatus        string
	AvgDurationSecs float64
	TransitionCount int64
}

// GetTimeInStage returns average durations for each stage transition.
// Uses a CTE with ROW_NUMBER() window function to correctly match each transition
// with its immediately preceding transition for the same application.
// This avoids the bug where a LEFT JOIN on created_at < t.created_at would match
// ALL previous transitions, producing inflated AVG and COUNT values.
func (r *AnalyticsRepo) GetTimeInStage(ctx context.Context, userID uint64, scopeKeys []string, deptIDs, locIDs []uint64, startDate, endDate *time.Time, jobID int64) ([]StageDurationRow, error) {
	jobIDs, err := r.ScopeJobIDs(ctx, userID, scopeKeys, deptIDs, locIDs)
	if err != nil {
		return nil, err
	}

	// Use a CTE with ROW_NUMBER() to assign a sequential row number per application.
	// Self-join on rn = t.rn - 1 ensures only the direct predecessor is matched.
	baseSQL := `
	WITH numbered AS (
	    SELECT *, ROW_NUMBER() OVER (PARTITION BY application_id ORDER BY created_at) AS rn
	    FROM application_status_transitions
	)
	SELECT t.from_status, t.to_status,
	       AVG(TIMESTAMPDIFF(SECOND, t_prev.created_at, t.created_at)) AS avg_duration_secs,
	       COUNT(*) AS transition_count
	FROM numbered t
	JOIN applications a ON a.id = t.application_id
	JOIN jobs j ON j.id = a.job_id
	LEFT JOIN numbered t_prev ON t_prev.application_id = t.application_id AND t_prev.rn = t.rn - 1
	WHERE t_prev.id IS NOT NULL`

	var conditions []string
	var args []any

	scopeWhere, scopeArgs := r.scopeFilter(jobIDs, "a")
	if scopeWhere != "" {
		conditions = append(conditions, scopeWhere)
		args = append(args, scopeArgs...)
	}
	if startDate != nil {
		conditions = append(conditions, "t.created_at >= ?")
		args = append(args, *startDate)
	}
	if endDate != nil {
		conditions = append(conditions, "t.created_at <= ?")
		args = append(args, *endDate)
	}
	if jobID > 0 {
		conditions = append(conditions, "a.job_id = ?")
		args = append(args, jobID)
	}

	fullSQL := baseSQL
	if len(conditions) > 0 {
		fullSQL += " AND " + strings.Join(conditions, " AND ")
	}
	fullSQL += " GROUP BY t.from_status, t.to_status"

	var rows []StageDurationRow
	if err := r.db.WithContext(ctx).Raw(fullSQL, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ---- Interview & Offer Metrics ----

type InterviewMetrics struct {
	TotalInterviews    int64
	CompletedInterviews int64
	PositiveFeedbacks  int64
}

type OfferMetrics struct {
	TotalOffers   int64
	AcceptedOffers int64
	RejectedOffers int64
}

// GetInterviewMetrics returns interview-related metrics.
func (r *AnalyticsRepo) GetInterviewMetrics(ctx context.Context, userID uint64, scopeKeys []string, deptIDs, locIDs []uint64, startDate, endDate *time.Time, jobID int64) (*InterviewMetrics, error) {
	jobIDs, err := r.ScopeJobIDs(ctx, userID, scopeKeys, deptIDs, locIDs)
	if err != nil {
		return nil, err
	}

	// Base interview query scoped to accessible jobs
	baseQuery := r.db.WithContext(ctx).Table("interview_schedules").
		Select("interview_schedules.id").
		Joins("JOIN applications a ON a.id = interview_schedules.application_id").
		Joins("JOIN jobs j ON j.id = a.job_id")
	where, args := r.scopeFilter(jobIDs, "a")
	if where != "" {
		baseQuery = baseQuery.Where(where, args...)
	}
	if startDate != nil {
		baseQuery = baseQuery.Where("interview_schedules.created_at >= ?", *startDate)
	}
	if endDate != nil {
		baseQuery = baseQuery.Where("interview_schedules.created_at <= ?", *endDate)
	}
	if jobID > 0 {
		baseQuery = baseQuery.Where("a.job_id = ?", jobID)
	}

	metrics := &InterviewMetrics{}

	// Total interviews
	if err := baseQuery.Count(&metrics.TotalInterviews).Error; err != nil {
		return nil, err
	}

	// Completed interviews
	completedQuery := baseQuery.Where("interview_schedules.status = ?", "completed")
	if err := completedQuery.Count(&metrics.CompletedInterviews).Error; err != nil {
		return nil, err
	}

	// Positive feedbacks (from interview_feedback on completed interviews)
	feedbackQuery := r.db.WithContext(ctx).Table("interview_feedback").
		Select("interview_feedback.id").
		Joins("JOIN interview_schedules s ON s.id = interview_feedback.interview_id").
		Joins("JOIN applications a ON a.id = s.application_id").
		Joins("JOIN jobs j ON j.id = a.job_id").
		Where("interview_feedback.recommendation = ?", "positive")
	where2, args2 := r.scopeFilter(jobIDs, "a")
	if where2 != "" {
		feedbackQuery = feedbackQuery.Where(where2, args2...)
	}
	if startDate != nil {
		feedbackQuery = feedbackQuery.Where("s.created_at >= ?", *startDate)
	}
	if endDate != nil {
		feedbackQuery = feedbackQuery.Where("s.created_at <= ?", *endDate)
	}
	if jobID > 0 {
		feedbackQuery = feedbackQuery.Where("a.job_id = ?", jobID)
	}
	if err := feedbackQuery.Count(&metrics.PositiveFeedbacks).Error; err != nil {
		return nil, err
	}

	return metrics, nil
}

// GetOfferMetrics returns offer-related metrics.
func (r *AnalyticsRepo) GetOfferMetrics(ctx context.Context, userID uint64, scopeKeys []string, deptIDs, locIDs []uint64, startDate, endDate *time.Time, jobID int64) (*OfferMetrics, error) {
	jobIDs, err := r.ScopeJobIDs(ctx, userID, scopeKeys, deptIDs, locIDs)
	if err != nil {
		return nil, err
	}

	offerQuery := r.db.WithContext(ctx).Table("offers").
		Select("offers.id").
		Joins("JOIN jobs j ON j.id = offers.job_id")
	where, args := r.scopeFilter(jobIDs, "j")
	if where != "" {
		offerQuery = offerQuery.Where(where, args...)
	}
	if startDate != nil {
		offerQuery = offerQuery.Where("offers.created_at >= ?", *startDate)
	}
	if endDate != nil {
		offerQuery = offerQuery.Where("offers.created_at <= ?", *endDate)
	}
	if jobID > 0 {
		offerQuery = offerQuery.Where("j.id = ?", jobID)
	}

	metrics := &OfferMetrics{}

	if err := offerQuery.Count(&metrics.TotalOffers).Error; err != nil {
		return nil, err
	}

	acceptedQuery := offerQuery.Where("offers.status = ?", "accepted")
	if err := acceptedQuery.Count(&metrics.AcceptedOffers).Error; err != nil {
		return nil, err
	}

	rejectedQuery := offerQuery.Where("offers.status = ?", "rejected")
	if err := rejectedQuery.Count(&metrics.RejectedOffers).Error; err != nil {
		return nil, err
	}

	return metrics, nil
}

// GetUnreadNotificationCount returns unread notification count for a user.
func (r *AnalyticsRepo) GetUnreadNotificationCount(ctx context.Context, userID int64, accountType string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("receiver_id = ? AND receiver_account_type = ? AND is_read = ?", userID, accountType, 0).
		Count(&count).Error
	return count, err
}
