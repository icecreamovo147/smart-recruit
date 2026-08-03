package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"smart-recruit-analytics-service/internal/application/port"
	appservice "smart-recruit-analytics-service/internal/application/service"
	"smart-recruit-analytics-service/internal/domain/model"
	"smart-recruit-platform-go/errs"
	"smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
)

func TestReportingAPIMapsDashboardResponse(t *testing.T) {
	api := newTestReportingAPI(t, &fakeRepo{
		kpi:    model.DashboardKPI{OnlineJobs: 2, OfflineJobs: 1, TotalApplications: 8, TodayApplications: 3, PendingActions: 4},
		unread: 5,
		trend:  []model.TrendPoint{{Date: "2026-07-13", Applications: 3}},
		stages: []model.StageCount{{StageKey: "applied", Count: 8}},
	}, fakeAuthorizer{}, fakeScopes{scope: model.ScopeData{ScopeKeys: []string{model.ScopeRecruitingAll}}})

	resp, err := api.GetDashboardReport(metadata.WithAuthActor(context.Background(), 7, "staff"), &pb.GetDashboardReportRequest{StaffUserId: 7})
	if err != nil {
		t.Fatalf("GetDashboardReport returned error: %v", err)
	}
	if resp.Code != errs.OK || resp.Msg != "common.success" {
		t.Fatalf("response code/msg = %d/%q", resp.Code, resp.Msg)
	}
	if resp.OnlineJobs != 2 || resp.UnreadNotifications != 5 || resp.Trend[0].Applications != 3 {
		t.Fatalf("response = %+v", resp)
	}
	if resp.StageDistribution[0].StageLabel != "已投递" {
		t.Fatalf("stage label = %q", resp.StageDistribution[0].StageLabel)
	}
}

func TestReportingAPIMapsFunnelTimeAndMetrics(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	repo := &fakeRepo{
		funnel:    []model.StageCount{{StageKey: "applied", Count: 10}, {StageKey: "viewed", Count: 5}},
		durations: []model.StageDurationRow{{FromStatus: "screening", AvgDurationSecs: 7200, TransitionCount: 2}},
		interview: model.InterviewMetrics{TotalInterviews: 6, CompletedInterviews: 4, PositiveFeedbacks: 3},
		offer:     model.OfferMetrics{TotalOffers: 5, AcceptedOffers: 2, RejectedOffers: 1},
	}
	api := newTestReportingAPI(t, repo, fakeAuthorizer{}, fakeScopes{scope: model.ScopeData{ScopeKeys: []string{model.ScopeRecruitingAll}}})
	ctx := metadata.WithAuthActor(context.Background(), 7, "staff")

	funnel, err := api.GetFunnelReport(ctx, &pb.GetFunnelReportRequest{StaffUserId: 7, StartDate: start.Format(time.RFC3339), JobId: 99})
	if err != nil {
		t.Fatalf("GetFunnelReport returned error: %v", err)
	}
	if funnel.Stages[1].ConversionRate != 50 || repo.lastFilter.JobID != 99 {
		t.Fatalf("funnel = %+v lastFilter = %+v", funnel.Stages, repo.lastFilter)
	}

	timeReport, err := api.GetTimeInStageReport(ctx, &pb.GetTimeInStageReportRequest{StaffUserId: 7})
	if err != nil {
		t.Fatalf("GetTimeInStageReport returned error: %v", err)
	}
	if timeReport.Durations[0].AvgHours != 2 {
		t.Fatalf("durations = %+v", timeReport.Durations)
	}

	metrics, err := api.GetInterviewOfferMetrics(ctx, &pb.GetInterviewOfferMetricsRequest{StaffUserId: 7})
	if err != nil {
		t.Fatalf("GetInterviewOfferMetrics returned error: %v", err)
	}
	if metrics.PassRate != 75 || metrics.AcceptanceRate != 40 {
		t.Fatalf("metrics = %+v", metrics)
	}
}

func TestReportingAPIReturnsForbiddenForPermissionFailures(t *testing.T) {
	api := newTestReportingAPI(t, &fakeRepo{}, fakeAuthorizer{err: port.ErrPermissionDenied}, fakeScopes{})

	resp, err := api.GetFunnelReport(metadata.WithAuthActor(context.Background(), 7, "staff"), &pb.GetFunnelReportRequest{StaffUserId: 7})
	if err != nil {
		t.Fatalf("GetFunnelReport returned error: %v", err)
	}
	if resp.Code != errs.ErrForbidden {
		t.Fatalf("code = %d, want forbidden", resp.Code)
	}
}

func newTestReportingAPI(t *testing.T, repo *fakeRepo, authorizer fakeAuthorizer, scopes fakeScopes) *ReportingAPI {
	t.Helper()
	service, err := appservice.NewReportingService(appservice.ReportingDeps{
		Reports:    repo,
		Authorizer: authorizer,
		Scopes:     scopes,
	})
	if err != nil {
		t.Fatalf("NewReportingService returned %v", err)
	}
	return NewReportingAPI(service)
}

type fakeAuthorizer struct {
	err error
}

func (f fakeAuthorizer) AuthorizePermission(context.Context, uint64, string) error {
	if f.err != nil {
		return f.err
	}
	return nil
}

type fakeScopes struct {
	scope model.ScopeData
	err   error
}

func (f fakeScopes) GetUserScopeData(_ context.Context, actorID uint64) (model.ScopeData, error) {
	if f.err != nil {
		return model.ScopeData{}, f.err
	}
	scope := f.scope
	scope.ActorID = actorID
	return scope, nil
}

type fakeRepo struct {
	kpi       model.DashboardKPI
	unread    int64
	trend     []model.TrendPoint
	stages    []model.StageCount
	funnel    []model.StageCount
	durations []model.StageDurationRow
	interview model.InterviewMetrics
	offer     model.OfferMetrics

	lastFilter model.ReportFilter
	err        error
}

func (f *fakeRepo) fail() error {
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeRepo) GetDashboardKPI(context.Context, model.ReportFilter) (model.DashboardKPI, error) {
	return f.kpi, f.fail()
}

func (f *fakeRepo) GetUnreadNotificationCount(context.Context, uint64, string) (int64, error) {
	return f.unread, nil
}

func (f *fakeRepo) GetTrend(context.Context, model.ReportFilter, int) ([]model.TrendPoint, error) {
	return f.trend, nil
}

func (f *fakeRepo) GetStageDistribution(context.Context, model.ReportFilter) ([]model.StageCount, error) {
	return f.stages, nil
}

func (f *fakeRepo) GetFunnelReport(_ context.Context, filter model.ReportFilter) ([]model.StageCount, error) {
	f.lastFilter = filter
	return f.funnel, f.fail()
}

func (f *fakeRepo) GetTimeInStage(context.Context, model.ReportFilter) ([]model.StageDurationRow, error) {
	return f.durations, f.fail()
}

func (f *fakeRepo) GetInterviewMetrics(context.Context, model.ReportFilter) (model.InterviewMetrics, error) {
	return f.interview, f.fail()
}

func (f *fakeRepo) GetOfferMetrics(context.Context, model.ReportFilter) (model.OfferMetrics, error) {
	return f.offer, f.fail()
}

var errFake = errors.New("fake error")
