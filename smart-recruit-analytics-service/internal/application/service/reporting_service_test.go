package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"smart-recruit-analytics-service/internal/application/query"
	"smart-recruit-analytics-service/internal/domain/model"
	"smart-recruit-analytics-service/internal/domain/policy"
)

func TestReportingServiceDashboardUsesJobReadAndFailSoftReads(t *testing.T) {
	repo := &fakeReportingRepo{
		kpi: model.DashboardKPI{
			OnlineJobs:        3,
			OfflineJobs:       1,
			TotalApplications: 12,
			TodayApplications: 2,
			PendingActions:    4,
		},
		unread:   6,
		trendErr: errors.New("trend unavailable"),
		stages:   []model.StageCount{{StageKey: "applied", Count: 8}},
	}
	authz := &fakeAuthorizer{}
	service := newTestReportingService(t, repo, authz, model.ScopeData{ScopeKeys: []string{model.ScopeRecruitingAll}})

	report, err := service.Dashboard(context.Background(), query.DashboardQuery{ActorID: 42, StaffUserID: 42})
	if err != nil {
		t.Fatalf("Dashboard returned %v", err)
	}
	if authz.permissions[0] != model.PermissionJobRead {
		t.Fatalf("permission = %q, want %q", authz.permissions[0], model.PermissionJobRead)
	}
	if report.OnlineJobs != 3 || report.UnreadNotifications != 6 || report.StageDistribution[0].StageLabel != "已投递" {
		t.Fatalf("report = %+v", report)
	}
	if !reflect.DeepEqual(report.Warnings, []string{"trend_unavailable"}) {
		t.Fatalf("warnings = %#v", report.Warnings)
	}
}

func TestReportingServiceFunnelUsesApplicationReadAndLegacyConversion(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	repo := &fakeReportingRepo{
		funnel: []model.StageCount{
			{StageKey: "hired", Count: 2},
			{StageKey: "applied", Count: 20},
			{StageKey: "viewed", Count: 10},
		},
	}
	authz := &fakeAuthorizer{}
	service := newTestReportingService(t, repo, authz, model.ScopeData{ScopeKeys: []string{model.ScopeRecruitingAll}})

	report, err := service.Funnel(context.Background(), query.FunnelQuery{ActorID: 7, StaffUserID: 7, StartDate: &start, EndDate: &end, JobID: 99})
	if err != nil {
		t.Fatalf("Funnel returned %v", err)
	}
	if authz.permissions[0] != model.PermissionApplicationRead {
		t.Fatalf("permission = %q, want %q", authz.permissions[0], model.PermissionApplicationRead)
	}
	if got := repo.lastFilter.JobID; got != 99 {
		t.Fatalf("job filter = %d, want 99", got)
	}
	if report.Stages[0].ConversionRate != 100 || report.Stages[1].ConversionRate != 50 || report.Stages[2].ConversionRate != 20 {
		t.Fatalf("stages = %+v", report.Stages)
	}
}

func TestReportingServiceTimeInStageAndInterviewOfferMetrics(t *testing.T) {
	repo := &fakeReportingRepo{
		durations: []model.StageDurationRow{{FromStatus: "interviewing", AvgDurationSecs: 5400, TransitionCount: 2}},
		interview: model.InterviewMetrics{
			TotalInterviews:     5,
			CompletedInterviews: 4,
			PositiveFeedbacks:   3,
		},
		offer: model.OfferMetrics{TotalOffers: 4, AcceptedOffers: 1, RejectedOffers: 2},
	}
	service := newTestReportingService(t, repo, &fakeAuthorizer{}, model.ScopeData{ScopeKeys: []string{model.ScopeRecruitingAll}})

	timeReport, err := service.TimeInStage(context.Background(), query.TimeInStageQuery{ActorID: 7, StaffUserID: 7})
	if err != nil {
		t.Fatalf("TimeInStage returned %v", err)
	}
	if timeReport.Durations[0].AvgHours != 1.5 || timeReport.Durations[0].StageLabel != "面试中" {
		t.Fatalf("durations = %+v", timeReport.Durations)
	}

	metricsReport, err := service.InterviewOfferMetrics(context.Background(), query.InterviewOfferMetricsQuery{ActorID: 7, StaffUserID: 7})
	if err != nil {
		t.Fatalf("InterviewOfferMetrics returned %v", err)
	}
	if metricsReport.Metrics.PassRate != 75 || metricsReport.Metrics.AcceptanceRate != 25 {
		t.Fatalf("metrics = %+v", metricsReport.Metrics)
	}
}

func TestReportingServiceRejectsActorMismatchWithoutSystemAll(t *testing.T) {
	service := newTestReportingService(t, &fakeReportingRepo{}, &fakeAuthorizer{}, model.ScopeData{ScopeKeys: []string{model.ScopeRecruitingAll}})

	_, err := service.Funnel(context.Background(), query.FunnelQuery{ActorID: 7, StaffUserID: 8})
	if !errors.Is(err, policy.ErrActorMismatch) {
		t.Fatalf("err = %v, want ErrActorMismatch", err)
	}
}

func TestProjectionServiceWritesOnlyProjectionStore(t *testing.T) {
	now := time.Date(2026, 7, 13, 10, 30, 0, 0, time.UTC)
	store := &fakeProjectionStore{}
	service, err := NewProjectionService(ProjectionDeps{Store: store, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("NewProjectionService returned %v", err)
	}

	result, err := service.Ingest(context.Background(), model.ProjectionEvent{EventID: "evt-1", EventType: "application.status_changed"}, "analytics-applications", "42")
	if err != nil {
		t.Fatalf("Ingest returned %v", err)
	}
	if result.EventID != "evt-1" || result.CheckpointName != "analytics-applications" {
		t.Fatalf("result = %+v", result)
	}
	if len(store.events) != 1 || len(store.checkpoints) != 1 {
		t.Fatalf("events=%d checkpoints=%d", len(store.events), len(store.checkpoints))
	}
	if store.checkpoints[0].UpdatedAt != now {
		t.Fatalf("checkpoint time = %v, want %v", store.checkpoints[0].UpdatedAt, now)
	}
}

func newTestReportingService(t *testing.T, repo *fakeReportingRepo, authz *fakeAuthorizer, scope model.ScopeData) *ReportingService {
	t.Helper()
	service, err := NewReportingService(ReportingDeps{
		Reports:    repo,
		Authorizer: authz,
		Scopes:     fakeScopeProvider{scope: scope},
	})
	if err != nil {
		t.Fatalf("NewReportingService returned %v", err)
	}
	return service
}

type fakeAuthorizer struct {
	permissions []string
}

func (f *fakeAuthorizer) AuthorizePermission(_ context.Context, _ uint64, permission string) error {
	f.permissions = append(f.permissions, permission)
	return nil
}

type fakeScopeProvider struct {
	scope model.ScopeData
}

func (f fakeScopeProvider) GetUserScopeData(_ context.Context, actorID uint64) (model.ScopeData, error) {
	scope := f.scope
	scope.ActorID = actorID
	return scope, nil
}

type fakeReportingRepo struct {
	kpi       model.DashboardKPI
	unread    int64
	trend     []model.TrendPoint
	trendErr  error
	stages    []model.StageCount
	funnel    []model.StageCount
	durations []model.StageDurationRow
	interview model.InterviewMetrics
	offer     model.OfferMetrics

	lastFilter model.ReportFilter
}

func (f *fakeReportingRepo) GetDashboardKPI(_ context.Context, filter model.ReportFilter) (model.DashboardKPI, error) {
	f.lastFilter = filter
	return f.kpi, nil
}

func (f *fakeReportingRepo) GetUnreadNotificationCount(context.Context, uint64, string) (int64, error) {
	return f.unread, nil
}

func (f *fakeReportingRepo) GetTrend(_ context.Context, _ model.ReportFilter, _ int) ([]model.TrendPoint, error) {
	return f.trend, f.trendErr
}

func (f *fakeReportingRepo) GetStageDistribution(context.Context, model.ReportFilter) ([]model.StageCount, error) {
	return f.stages, nil
}

func (f *fakeReportingRepo) GetFunnelReport(_ context.Context, filter model.ReportFilter) ([]model.StageCount, error) {
	f.lastFilter = filter
	return f.funnel, nil
}

func (f *fakeReportingRepo) GetTimeInStage(context.Context, model.ReportFilter) ([]model.StageDurationRow, error) {
	return f.durations, nil
}

func (f *fakeReportingRepo) GetInterviewMetrics(context.Context, model.ReportFilter) (model.InterviewMetrics, error) {
	return f.interview, nil
}

func (f *fakeReportingRepo) GetOfferMetrics(context.Context, model.ReportFilter) (model.OfferMetrics, error) {
	return f.offer, nil
}

type fakeProjectionStore struct {
	events      []model.ProjectionEvent
	checkpoints []model.ProjectionCheckpoint
}

func (f *fakeProjectionStore) SaveProjectionEvent(_ context.Context, event model.ProjectionEvent) error {
	f.events = append(f.events, event)
	return nil
}

func (f *fakeProjectionStore) SaveCheckpoint(_ context.Context, checkpoint model.ProjectionCheckpoint) error {
	f.checkpoints = append(f.checkpoints, checkpoint)
	return nil
}
