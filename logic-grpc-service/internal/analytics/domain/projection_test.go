package domain

import "testing"

func TestAnalyticsProjectionVocabulary(t *testing.T) {
	t.Parallel()

	if ProjectionDashboard != "dashboard" {
		t.Fatalf("dashboard projection drifted: %q", ProjectionDashboard)
	}
	if ProjectionInterviewOffer != "interview_offer" {
		t.Fatalf("interview/offer projection drifted: %q", ProjectionInterviewOffer)
	}
	if QueryTimeInStageReport != "time_in_stage_report" {
		t.Fatalf("time-in-stage query drifted: %q", QueryTimeInStageReport)
	}
	if ProjectionSourceDomainEvent != "domain_event" {
		t.Fatalf("domain event source drifted: %q", ProjectionSourceDomainEvent)
	}
}
