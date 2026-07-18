package query

import "time"

type DashboardQuery struct {
	ActorID     uint64
	StaffUserID int64
}

type FunnelQuery struct {
	ActorID     uint64
	StaffUserID int64
	StartDate   *time.Time
	EndDate     *time.Time
	JobID       int64
}

type TimeInStageQuery struct {
	ActorID     uint64
	StaffUserID int64
	StartDate   *time.Time
	EndDate     *time.Time
	JobID       int64
}

type InterviewOfferMetricsQuery struct {
	ActorID     uint64
	StaffUserID int64
	StartDate   *time.Time
	EndDate     *time.Time
	JobID       int64
}
