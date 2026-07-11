package application

import (
	"context"
	"encoding/json"
	"time"

	"logic-grpc-service/repository"
)

// ReportingReadModelRepository is the Analytics-owned query/read-model port.
type ReportingReadModelRepository interface {
	ScopeJobIDs(ctx context.Context, userID uint64, scopeKeys []string, deptIDs, locIDs []uint64) ([]uint64, error)
	GetDashboardKPI(ctx context.Context, userID uint64, scopeKeys []string, deptIDs, locIDs []uint64) (*repository.DashboardKPI, error)
	GetStageDistribution(ctx context.Context, userID uint64, scopeKeys []string, deptIDs, locIDs []uint64) ([]repository.StageCount, error)
	GetTrend(ctx context.Context, userID uint64, scopeKeys []string, deptIDs, locIDs []uint64, days int) ([]repository.TrendPoint, error)
	GetFunnelReport(ctx context.Context, userID uint64, scopeKeys []string, deptIDs, locIDs []uint64, startDate, endDate *time.Time, jobID int64) ([]repository.StageCount, error)
	GetTimeInStage(ctx context.Context, userID uint64, scopeKeys []string, deptIDs, locIDs []uint64, startDate, endDate *time.Time, jobID int64) ([]repository.StageDurationRow, error)
	GetInterviewMetrics(ctx context.Context, userID uint64, scopeKeys []string, deptIDs, locIDs []uint64, startDate, endDate *time.Time, jobID int64) (*repository.InterviewMetrics, error)
	GetOfferMetrics(ctx context.Context, userID uint64, scopeKeys []string, deptIDs, locIDs []uint64, startDate, endDate *time.Time, jobID int64) (*repository.OfferMetrics, error)
	GetUnreadNotificationCount(ctx context.Context, userID int64, accountType string) (int64, error)
}

// ProjectionCheckpointStore is the future Analytics-owned checkpoint port for event projections.
type ProjectionCheckpointStore interface {
	GetCheckpoint(ctx context.Context, projectionName string) (string, error)
	SaveCheckpoint(ctx context.Context, projectionName string, cursor string) error
}

type ProjectionEventRecord struct {
	ProjectionName string
	Source         string
	EventID        string
	EventType      string
	AggregateType  string
	AggregateID    string
	Producer       string
	IdempotencyKey string
	CorrelationID  string
	CausationID    string
	TraceID        string
	Payload        json.RawMessage
	Metadata       map[string]string
	OccurredAt     time.Time
	ProjectedAt    time.Time
}

type ProjectionEventStore interface {
	SaveProjectionEvent(ctx context.Context, event ProjectionEventRecord) (bool, error)
}
