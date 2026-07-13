package model

import (
	"encoding/json"
	"time"
)

const (
	PermissionJobRead         = "job.read"
	PermissionApplicationRead = "application.read"

	ScopeSystemAll     = "system_all"
	ScopeRecruitingAll = "recruiting_all"
)

type ScopeData struct {
	ActorID       uint64
	ScopeKeys     []string
	DepartmentIDs []uint64
	LocationIDs   []uint64
}

func (s ScopeData) HasScope(scope string) bool {
	for _, key := range s.ScopeKeys {
		if key == scope {
			return true
		}
	}
	return false
}

type ReportFilter struct {
	ActorID       uint64
	ScopeKeys     []string
	DepartmentIDs []uint64
	LocationIDs   []uint64
	StartDate     *time.Time
	EndDate       *time.Time
	JobID         int64
}

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
	Date         string
	Applications int64
}

type FunnelStage struct {
	StageKey       string
	StageLabel     string
	Count          int64
	ConversionRate float64
}

type StageDurationRow struct {
	FromStatus      string
	ToStatus        string
	AvgDurationSecs float64
	TransitionCount int64
}

type StageDuration struct {
	StageKey        string
	StageLabel      string
	AvgHours        float64
	TransitionCount int64
}

type InterviewMetrics struct {
	TotalInterviews     int64
	CompletedInterviews int64
	PositiveFeedbacks   int64
}

type OfferMetrics struct {
	TotalOffers    int64
	AcceptedOffers int64
	RejectedOffers int64
}

type InterviewOfferMetrics struct {
	TotalInterviews     int64
	CompletedInterviews int64
	PositiveFeedbacks   int64
	PassRate            float64
	TotalOffers         int64
	AcceptedOffers      int64
	RejectedOffers      int64
	AcceptanceRate      float64
}

type ProjectionEvent struct {
	EventID       string
	EventType     string
	AggregateType string
	AggregateID   string
	OccurredAt    time.Time
	Payload       json.RawMessage
}

type ProjectionCheckpoint struct {
	ProjectionName string
	Cursor         string
	UpdatedAt      time.Time
}

type TransitionalRead struct {
	Owner            string
	Tables           []string
	Reason           string
	RemovalCondition string
}

type ProjectionStrategy struct {
	Mode                string
	Source              string
	TransitionalReads   []TransitionalRead
	TransactionalWrites bool
}
