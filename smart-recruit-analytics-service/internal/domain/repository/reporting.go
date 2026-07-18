package repository

import (
	"context"

	"smart-recruit-analytics-service/internal/domain/model"
)

type ReportingRepository interface {
	GetDashboardKPI(context.Context, model.ReportFilter) (model.DashboardKPI, error)
	GetUnreadNotificationCount(context.Context, uint64, string) (int64, error)
	GetTrend(context.Context, model.ReportFilter, int) ([]model.TrendPoint, error)
	GetStageDistribution(context.Context, model.ReportFilter) ([]model.StageCount, error)
	GetFunnelReport(context.Context, model.ReportFilter) ([]model.StageCount, error)
	GetTimeInStage(context.Context, model.ReportFilter) ([]model.StageDurationRow, error)
	GetInterviewMetrics(context.Context, model.ReportFilter) (model.InterviewMetrics, error)
	GetOfferMetrics(context.Context, model.ReportFilter) (model.OfferMetrics, error)
}

type ProjectionStore interface {
	SaveProjectionEvent(context.Context, model.ProjectionEvent) error
	SaveCheckpoint(context.Context, model.ProjectionCheckpoint) error
}
