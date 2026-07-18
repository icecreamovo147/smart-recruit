package service

import (
	"context"
	"errors"
	"time"

	"smart-recruit-analytics-service/internal/application/dto"
	"smart-recruit-analytics-service/internal/application/port"
	"smart-recruit-analytics-service/internal/application/query"
	"smart-recruit-analytics-service/internal/domain/model"
	"smart-recruit-analytics-service/internal/domain/policy"
	"smart-recruit-analytics-service/internal/domain/repository"
)

var (
	ErrReportingRepositoryRequired = errors.New("analytics reporting repository is required")
	ErrAuthorizerRequired          = errors.New("analytics permission authorizer is required")
	ErrScopeProviderRequired       = errors.New("analytics scope provider is required")
	ErrProjectionStoreRequired     = errors.New("analytics projection store is required")
)

type ReportingDeps struct {
	Reports    repository.ReportingRepository
	Authorizer port.PermissionAuthorizer
	Scopes     port.ScopeProvider
}

type ReportingService struct {
	reports    repository.ReportingRepository
	authorizer port.PermissionAuthorizer
	scopes     port.ScopeProvider
}

func NewReportingService(deps ReportingDeps) (*ReportingService, error) {
	if deps.Reports == nil {
		return nil, ErrReportingRepositoryRequired
	}
	if deps.Authorizer == nil {
		return nil, ErrAuthorizerRequired
	}
	if deps.Scopes == nil {
		return nil, ErrScopeProviderRequired
	}
	return &ReportingService{
		reports:    deps.Reports,
		authorizer: deps.Authorizer,
		scopes:     deps.Scopes,
	}, nil
}

func (s *ReportingService) Dashboard(ctx context.Context, q query.DashboardQuery) (dto.DashboardReport, error) {
	scope, err := s.authorizedScope(ctx, q.ActorID, q.StaffUserID, model.PermissionJobRead)
	if err != nil {
		return dto.DashboardReport{}, err
	}
	filter := policy.BuildFilter(scope, nil, nil, 0)
	kpi, err := s.reports.GetDashboardKPI(ctx, filter)
	if err != nil {
		return dto.DashboardReport{}, err
	}

	report := dto.DashboardReport{
		OnlineJobs:        kpi.OnlineJobs,
		OfflineJobs:       kpi.OfflineJobs,
		TotalApplications: kpi.TotalApplications,
		TodayApplications: kpi.TodayApplications,
		PendingActions:    kpi.PendingActions,
	}
	if unread, err := s.reports.GetUnreadNotificationCount(ctx, scope.ActorID, "staff"); err == nil {
		report.UnreadNotifications = unread
	} else {
		report.Warnings = append(report.Warnings, "unread_notification_count_unavailable")
	}
	if trend, err := s.reports.GetTrend(ctx, filter, policy.DefaultDashboardTrendDays); err == nil {
		report.Trend = trend
	} else {
		report.Warnings = append(report.Warnings, "trend_unavailable")
	}
	if stages, err := s.reports.GetStageDistribution(ctx, filter); err == nil {
		report.StageDistribution = policy.BuildFunnelStages(stages)
	} else {
		report.Warnings = append(report.Warnings, "stage_distribution_unavailable")
	}
	return report, nil
}

func (s *ReportingService) Funnel(ctx context.Context, q query.FunnelQuery) (dto.FunnelReport, error) {
	scope, err := s.authorizedScope(ctx, q.ActorID, q.StaffUserID, model.PermissionApplicationRead)
	if err != nil {
		return dto.FunnelReport{}, err
	}
	counts, err := s.reports.GetFunnelReport(ctx, policy.BuildFilter(scope, q.StartDate, q.EndDate, q.JobID))
	if err != nil {
		return dto.FunnelReport{}, err
	}
	return dto.FunnelReport{Stages: policy.BuildFunnelStages(counts)}, nil
}

func (s *ReportingService) TimeInStage(ctx context.Context, q query.TimeInStageQuery) (dto.TimeInStageReport, error) {
	scope, err := s.authorizedScope(ctx, q.ActorID, q.StaffUserID, model.PermissionApplicationRead)
	if err != nil {
		return dto.TimeInStageReport{}, err
	}
	rows, err := s.reports.GetTimeInStage(ctx, policy.BuildFilter(scope, q.StartDate, q.EndDate, q.JobID))
	if err != nil {
		return dto.TimeInStageReport{}, err
	}
	return dto.TimeInStageReport{Durations: policy.BuildStageDurations(rows)}, nil
}

func (s *ReportingService) InterviewOfferMetrics(ctx context.Context, q query.InterviewOfferMetricsQuery) (dto.InterviewOfferMetricsReport, error) {
	scope, err := s.authorizedScope(ctx, q.ActorID, q.StaffUserID, model.PermissionApplicationRead)
	if err != nil {
		return dto.InterviewOfferMetricsReport{}, err
	}
	filter := policy.BuildFilter(scope, q.StartDate, q.EndDate, q.JobID)
	interview, err := s.reports.GetInterviewMetrics(ctx, filter)
	if err != nil {
		return dto.InterviewOfferMetricsReport{}, err
	}
	offer, err := s.reports.GetOfferMetrics(ctx, filter)
	if err != nil {
		return dto.InterviewOfferMetricsReport{}, err
	}
	return dto.InterviewOfferMetricsReport{Metrics: policy.BuildInterviewOfferMetrics(interview, offer)}, nil
}

func (s *ReportingService) authorizedScope(ctx context.Context, actorID uint64, requestedStaffUserID int64, permission string) (model.ScopeData, error) {
	if actorID == 0 {
		return model.ScopeData{}, policy.ErrActorRequired
	}
	if err := s.authorizer.AuthorizePermission(ctx, actorID, permission); err != nil {
		return model.ScopeData{}, err
	}
	scope, err := s.scopes.GetUserScopeData(ctx, actorID)
	if err != nil {
		return model.ScopeData{}, err
	}
	scope.ActorID = actorID
	if err := policy.ValidateActor(actorID, requestedStaffUserID, scope); err != nil {
		return model.ScopeData{}, err
	}
	return scope, nil
}

type ProjectionDeps struct {
	Store    repository.ProjectionStore
	Strategy model.ProjectionStrategy
	Now      func() time.Time
}

type ProjectionService struct {
	store    repository.ProjectionStore
	strategy model.ProjectionStrategy
	now      func() time.Time
}

func NewProjectionService(deps ProjectionDeps) (*ProjectionService, error) {
	if deps.Store == nil {
		return nil, ErrProjectionStoreRequired
	}
	strategy := deps.Strategy
	if strategy.Mode == "" && strategy.Source == "" {
		strategy = policy.DefaultProjectionStrategy()
	}
	if err := policy.ValidateProjectionStrategy(strategy); err != nil {
		return nil, err
	}
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	return &ProjectionService{store: deps.Store, strategy: strategy, now: now}, nil
}

func (s *ProjectionService) Ingest(ctx context.Context, event model.ProjectionEvent, checkpointName, cursor string) (dto.ProjectionIngestResult, error) {
	if err := policy.ValidateProjectionStrategy(s.strategy); err != nil {
		return dto.ProjectionIngestResult{}, err
	}
	if err := s.store.SaveProjectionEvent(ctx, event); err != nil {
		return dto.ProjectionIngestResult{}, err
	}
	if checkpointName != "" {
		if err := s.store.SaveCheckpoint(ctx, model.ProjectionCheckpoint{
			ProjectionName: checkpointName,
			Cursor:         cursor,
			UpdatedAt:      s.now(),
		}); err != nil {
			return dto.ProjectionIngestResult{}, err
		}
	}
	return dto.ProjectionIngestResult{EventID: event.EventID, CheckpointName: checkpointName}, nil
}
