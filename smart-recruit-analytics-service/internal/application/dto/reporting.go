package dto

import "smart-recruit-analytics-service/internal/domain/model"

type DashboardReport struct {
	OnlineJobs          int64
	OfflineJobs         int64
	TotalApplications   int64
	TodayApplications   int64
	UnreadNotifications int64
	PendingActions      int64
	Trend               []model.TrendPoint
	StageDistribution   []model.FunnelStage
	Warnings            []string
}

type FunnelReport struct {
	Stages []model.FunnelStage
}

type TimeInStageReport struct {
	Durations []model.StageDuration
}

type InterviewOfferMetricsReport struct {
	Metrics model.InterviewOfferMetrics
}

type ProjectionIngestResult struct {
	EventID        string
	CheckpointName string
}
