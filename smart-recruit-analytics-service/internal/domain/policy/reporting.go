package policy

import (
	"errors"
	"fmt"
	"time"

	"smart-recruit-analytics-service/internal/domain/model"
)

const (
	DefaultDashboardTrendDays = 7
	ProjectionModeReadModel   = "domain-event-projection-read-models"
)

var (
	ErrActorRequired                = errors.New("authenticated analytics actor is required")
	ErrActorMismatch                = errors.New("analytics actor does not match requested staff user")
	ErrProjectionSourceRequired     = errors.New("analytics projection source is required")
	ErrProjectionModeRequired       = errors.New("analytics projection mode is required")
	ErrTransactionalWritesForbidden = errors.New("analytics projection must not write transactional business state")
)

var FunnelOrder = []string{
	"applied",
	"viewed",
	"screening",
	"screen_passed",
	"interview_pending",
	"interviewing",
	"interview_passed",
	"offer_pending",
	"offer_sent",
	"hired",
}

func ValidateActor(actorID uint64, requestedStaffUserID int64, scope model.ScopeData) error {
	if actorID == 0 {
		return ErrActorRequired
	}
	if requestedStaffUserID <= 0 || uint64(requestedStaffUserID) == actorID {
		return nil
	}
	if scope.HasScope(model.ScopeSystemAll) {
		return nil
	}
	return fmt.Errorf("%w: authenticated user %d != requested user %d", ErrActorMismatch, actorID, requestedStaffUserID)
}

func BuildFilter(scope model.ScopeData, startDate, endDate *time.Time, jobID int64) model.ReportFilter {
	return model.ReportFilter{
		ActorID:       scope.ActorID,
		ScopeKeys:     append([]string(nil), scope.ScopeKeys...),
		DepartmentIDs: append([]uint64(nil), scope.DepartmentIDs...),
		LocationIDs:   append([]uint64(nil), scope.LocationIDs...),
		StartDate:     startDate,
		EndDate:       endDate,
		JobID:         jobID,
	}
}

func BuildFunnelStages(counts []model.StageCount) []model.FunnelStage {
	stageMap := make(map[string]int64, len(counts))
	for _, stage := range counts {
		stageMap[stage.StageKey] = stage.Count
	}

	stages := make([]model.FunnelStage, 0, len(FunnelOrder))
	var previous int64
	first := true
	for _, key := range FunnelOrder {
		count, ok := stageMap[key]
		if !ok {
			continue
		}
		rate := 0.0
		if first {
			rate = 100.0
			first = false
		} else if previous > 0 {
			rate = float64(count) / float64(previous) * 100.0
		}
		previous = count
		stages = append(stages, model.FunnelStage{
			StageKey:       key,
			StageLabel:     StageLabel(key),
			Count:          count,
			ConversionRate: rate,
		})
	}
	return stages
}

func BuildStageDurations(rows []model.StageDurationRow) []model.StageDuration {
	durations := make([]model.StageDuration, 0, len(rows))
	for _, row := range rows {
		durations = append(durations, model.StageDuration{
			StageKey:        row.FromStatus,
			StageLabel:      StageLabel(row.FromStatus),
			AvgHours:        row.AvgDurationSecs / 3600.0,
			TransitionCount: row.TransitionCount,
		})
	}
	return durations
}

func BuildInterviewOfferMetrics(interview model.InterviewMetrics, offer model.OfferMetrics) model.InterviewOfferMetrics {
	metrics := model.InterviewOfferMetrics{
		TotalInterviews:     interview.TotalInterviews,
		CompletedInterviews: interview.CompletedInterviews,
		PositiveFeedbacks:   interview.PositiveFeedbacks,
		TotalOffers:         offer.TotalOffers,
		AcceptedOffers:      offer.AcceptedOffers,
		RejectedOffers:      offer.RejectedOffers,
	}
	if interview.CompletedInterviews > 0 {
		metrics.PassRate = float64(interview.PositiveFeedbacks) / float64(interview.CompletedInterviews) * 100.0
	}
	if offer.TotalOffers > 0 {
		metrics.AcceptanceRate = float64(offer.AcceptedOffers) / float64(offer.TotalOffers) * 100.0
	}
	return metrics
}

func StageLabel(key string) string {
	switch key {
	case "applied":
		return "已投递"
	case "viewed":
		return "已查看"
	case "screening":
		return "筛选中"
	case "screen_passed":
		return "筛选通过"
	case "interview_pending":
		return "待安排面试"
	case "interviewing":
		return "面试中"
	case "interview_passed":
		return "面试通过"
	case "offer_pending":
		return "待发Offer"
	case "offer_sent":
		return "Offer已发"
	case "hired":
		return "已入职"
	case "rejected":
		return "淘汰"
	case "withdrawn":
		return "候选人撤回"
	default:
		return key
	}
}

func DefaultProjectionStrategy() model.ProjectionStrategy {
	return model.ProjectionStrategy{
		Mode:   ProjectionModeReadModel,
		Source: "analytics-owned projection/read-model",
		TransitionalReads: []model.TransitionalRead{
			{
				Owner:            "recruitment",
				Tables:           []string{"jobs", "applications", "application_status_transitions"},
				Reason:           "dashboard, funnel, and time-in-stage reports still read source-domain tables until projection-backed read models are complete",
				RemovalCondition: "replace with Analytics-owned projections sourced from application lifecycle events",
			},
			{
				Owner:            "interview",
				Tables:           []string{"interview_schedules", "interview_feedback"},
				Reason:           "interview metrics remain transitional read queries",
				RemovalCondition: "replace with Analytics-owned projections sourced from interview events",
			},
			{
				Owner:            "offer",
				Tables:           []string{"offers"},
				Reason:           "offer metrics remain transitional read queries",
				RemovalCondition: "replace with Analytics-owned projections sourced from offer events",
			},
			{
				Owner:            "notification",
				Tables:           []string{"notifications"},
				Reason:           "dashboard unread count remains a transitional read",
				RemovalCondition: "replace with Analytics-owned notification read-state projection or owner service query",
			},
			{
				Owner:            "identity",
				Tables:           []string{"rbac scopes and permission assignments"},
				Reason:           "reporting queries require Identity-owned authorization and data-scope decisions",
				RemovalCondition: "replace direct reads with Identity query ports or signed scope snapshots",
			},
		},
	}
}

func ValidateProjectionStrategy(strategy model.ProjectionStrategy) error {
	if strategy.Source == "" {
		return ErrProjectionSourceRequired
	}
	if strategy.Mode == "" {
		return ErrProjectionModeRequired
	}
	if strategy.Mode != ProjectionModeReadModel {
		return fmt.Errorf("analytics projection mode = %q, want %s", strategy.Mode, ProjectionModeReadModel)
	}
	if strategy.TransactionalWrites {
		return ErrTransactionalWritesForbidden
	}
	return nil
}
