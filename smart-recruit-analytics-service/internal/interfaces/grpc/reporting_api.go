package grpc

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"smart-recruit-analytics-service/internal/application/port"
	"smart-recruit-analytics-service/internal/application/query"
	appservice "smart-recruit-analytics-service/internal/application/service"
	"smart-recruit-analytics-service/internal/domain/model"
	"smart-recruit-analytics-service/internal/domain/policy"
	"smart-recruit-platform-go/errs"
	"smart-recruit-platform-go/logger"
	"smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
)

type ReportingAPI struct {
	service *appservice.ReportingService
}

func NewReportingAPI(service *appservice.ReportingService) *ReportingAPI {
	return &ReportingAPI{service: service}
}

func (a *ReportingAPI) GetDashboardReport(ctx context.Context, req *pb.GetDashboardReportRequest) (*pb.GetDashboardReportResponse, error) {
	actorID := metadata.GetAuthUserID(ctx)
	report, err := a.service.Dashboard(ctx, query.DashboardQuery{
		ActorID:     uint64(actorID),
		StaffUserID: req.GetStaffUserId(),
	})
	if err != nil {
		return dashboardError(err)
	}
	return &pb.GetDashboardReportResponse{
		Code:                errs.OK,
		Msg:                 "common.success",
		OnlineJobs:          report.OnlineJobs,
		OfflineJobs:         report.OfflineJobs,
		TotalApplications:   report.TotalApplications,
		TodayApplications:   report.TodayApplications,
		UnreadNotifications: report.UnreadNotifications,
		PendingActions:      report.PendingActions,
		Trend:               trendToProto(report.Trend),
		StageDistribution:   dashboardStagesToProto(report.StageDistribution),
	}, nil
}

func (a *ReportingAPI) GetFunnelReport(ctx context.Context, req *pb.GetFunnelReportRequest) (*pb.GetFunnelReportResponse, error) {
	actorID := metadata.GetAuthUserID(ctx)
	startDate, endDate := parseDateRange(req.GetStartDate(), req.GetEndDate())
	report, err := a.service.Funnel(ctx, query.FunnelQuery{
		ActorID:     uint64(actorID),
		StaffUserID: req.GetStaffUserId(),
		StartDate:   startDate,
		EndDate:     endDate,
		JobID:       req.GetJobId(),
	})
	if err != nil {
		return funnelError(err)
	}
	return &pb.GetFunnelReportResponse{
		Code:   errs.OK,
		Msg:    "common.success",
		Stages: funnelStagesToProto(report.Stages),
	}, nil
}

func (a *ReportingAPI) GetTimeInStageReport(ctx context.Context, req *pb.GetTimeInStageReportRequest) (*pb.GetTimeInStageReportResponse, error) {
	actorID := metadata.GetAuthUserID(ctx)
	startDate, endDate := parseDateRange(req.GetStartDate(), req.GetEndDate())
	report, err := a.service.TimeInStage(ctx, query.TimeInStageQuery{
		ActorID:     uint64(actorID),
		StaffUserID: req.GetStaffUserId(),
		StartDate:   startDate,
		EndDate:     endDate,
		JobID:       req.GetJobId(),
	})
	if err != nil {
		return timeInStageError(err)
	}
	return &pb.GetTimeInStageReportResponse{
		Code:      errs.OK,
		Msg:       "common.success",
		Durations: durationsToProto(report.Durations),
	}, nil
}

func (a *ReportingAPI) GetInterviewOfferMetrics(ctx context.Context, req *pb.GetInterviewOfferMetricsRequest) (*pb.GetInterviewOfferMetricsResponse, error) {
	actorID := metadata.GetAuthUserID(ctx)
	startDate, endDate := parseDateRange(req.GetStartDate(), req.GetEndDate())
	report, err := a.service.InterviewOfferMetrics(ctx, query.InterviewOfferMetricsQuery{
		ActorID:     uint64(actorID),
		StaffUserID: req.GetStaffUserId(),
		StartDate:   startDate,
		EndDate:     endDate,
		JobID:       req.GetJobId(),
	})
	if err != nil {
		return interviewOfferError(err)
	}
	metrics := report.Metrics
	return &pb.GetInterviewOfferMetricsResponse{
		Code:                errs.OK,
		Msg:                 "common.success",
		TotalInterviews:     metrics.TotalInterviews,
		CompletedInterviews: metrics.CompletedInterviews,
		PositiveFeedbacks:   metrics.PositiveFeedbacks,
		PassRate:            metrics.PassRate,
		TotalOffers:         metrics.TotalOffers,
		AcceptedOffers:      metrics.AcceptedOffers,
		RejectedOffers:      metrics.RejectedOffers,
		AcceptanceRate:      metrics.AcceptanceRate,
	}, nil
}

func dashboardError(err error) (*pb.GetDashboardReportResponse, error) {
	if isForbidden(err) {
		return &pb.GetDashboardReportResponse{Code: errs.ErrForbidden, Msg: "common.forbidden"}, nil
	}
	logger.L().Error("analytics: dashboard query failed", zap.Error(err))
	return &pb.GetDashboardReportResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
}

func funnelError(err error) (*pb.GetFunnelReportResponse, error) {
	if isForbidden(err) {
		return &pb.GetFunnelReportResponse{Code: errs.ErrForbidden, Msg: "common.forbidden"}, nil
	}
	logger.L().Error("analytics: funnel query failed", zap.Error(err))
	return &pb.GetFunnelReportResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
}

func timeInStageError(err error) (*pb.GetTimeInStageReportResponse, error) {
	if isForbidden(err) {
		return &pb.GetTimeInStageReportResponse{Code: errs.ErrForbidden, Msg: "common.forbidden"}, nil
	}
	logger.L().Error("analytics: time-in-stage query failed", zap.Error(err))
	return &pb.GetTimeInStageReportResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
}

func interviewOfferError(err error) (*pb.GetInterviewOfferMetricsResponse, error) {
	if isForbidden(err) {
		return &pb.GetInterviewOfferMetricsResponse{Code: errs.ErrForbidden, Msg: "common.forbidden"}, nil
	}
	logger.L().Error("analytics: interview-offer metrics query failed", zap.Error(err))
	return &pb.GetInterviewOfferMetricsResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
}

func isForbidden(err error) bool {
	return errors.Is(err, policy.ErrActorRequired) ||
		errors.Is(err, policy.ErrActorMismatch) ||
		errors.Is(err, port.ErrPermissionDenied)
}

func parseDateRange(start, end string) (*time.Time, *time.Time) {
	var startDate, endDate *time.Time
	if start != "" {
		if parsed, err := time.Parse(time.RFC3339, start); err == nil {
			startDate = &parsed
		}
	}
	if end != "" {
		if parsed, err := time.Parse(time.RFC3339, end); err == nil {
			endDate = &parsed
		}
	}
	return startDate, endDate
}

func trendToProto(points []model.TrendPoint) []*pb.DashboardTrendPoint {
	result := make([]*pb.DashboardTrendPoint, len(points))
	for i, point := range points {
		result[i] = &pb.DashboardTrendPoint{Date: point.Date, Applications: point.Applications}
	}
	return result
}

func dashboardStagesToProto(stages []model.FunnelStage) []*pb.DashboardReportItem {
	result := make([]*pb.DashboardReportItem, len(stages))
	for i, stage := range stages {
		result[i] = &pb.DashboardReportItem{
			StageKey:   stage.StageKey,
			StageLabel: stage.StageLabel,
			Count:      stage.Count,
		}
	}
	return result
}

func funnelStagesToProto(stages []model.FunnelStage) []*pb.FunnelStage {
	result := make([]*pb.FunnelStage, len(stages))
	for i, stage := range stages {
		result[i] = &pb.FunnelStage{
			StageKey:       stage.StageKey,
			StageLabel:     stage.StageLabel,
			Count:          stage.Count,
			ConversionRate: stage.ConversionRate,
		}
	}
	return result
}

func durationsToProto(durations []model.StageDuration) []*pb.StageDuration {
	result := make([]*pb.StageDuration, len(durations))
	for i, duration := range durations {
		result[i] = &pb.StageDuration{
			StageKey:        duration.StageKey,
			StageLabel:      duration.StageLabel,
			AvgHours:        duration.AvgHours,
			TransitionCount: duration.TransitionCount,
		}
	}
	return result
}
