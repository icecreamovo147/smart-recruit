package domain

import "strings"

// ProjectionName identifies an Analytics-owned projection or read model.
type ProjectionName string

const (
	ProjectionDashboard      ProjectionName = "dashboard"
	ProjectionFunnel         ProjectionName = "funnel"
	ProjectionTimeInStage    ProjectionName = "time_in_stage"
	ProjectionInterviewOffer ProjectionName = "interview_offer"
	ProjectionAuthAudit      ProjectionName = "auth_audit"
)

// QueryName identifies an Analytics-owned reporting query surface.
type QueryName string

const (
	QueryDashboardReport       QueryName = "dashboard_report"
	QueryFunnelReport          QueryName = "funnel_report"
	QueryTimeInStageReport     QueryName = "time_in_stage_report"
	QueryInterviewOfferMetrics QueryName = "interview_offer_metrics"
	QueryAuthAuditLogs         QueryName = "auth_audit_logs"
)

// ProjectionSource describes the final source class for Analytics read models.
type ProjectionSource string

const (
	ProjectionSourceDomainEvent ProjectionSource = "domain_event"
	ProjectionSourceReadModel   ProjectionSource = "read_model"
)

// ProjectionForEventType maps source-domain events into Analytics-owned projections.
func ProjectionForEventType(eventType string) ProjectionName {
	switch {
	case strings.HasPrefix(eventType, "interview."), strings.HasPrefix(eventType, "offer."):
		return ProjectionInterviewOffer
	case strings.Contains(eventType, "status"), strings.HasPrefix(eventType, "application."):
		return ProjectionFunnel
	case strings.HasPrefix(eventType, "auth."), strings.Contains(eventType, "audit"):
		return ProjectionAuthAudit
	default:
		return ProjectionDashboard
	}
}
