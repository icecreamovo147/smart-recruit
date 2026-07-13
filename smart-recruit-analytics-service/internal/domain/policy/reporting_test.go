package policy

import (
	"errors"
	"testing"

	"smart-recruit-analytics-service/internal/domain/model"
)

func TestBuildFunnelStagesKeepsLegacyOrderAndConversion(t *testing.T) {
	stages := BuildFunnelStages([]model.StageCount{
		{StageKey: "offer_sent", Count: 4},
		{StageKey: "applied", Count: 10},
		{StageKey: "viewed", Count: 5},
		{StageKey: "unknown", Count: 99},
	})

	if len(stages) != 3 {
		t.Fatalf("len(stages) = %d, want 3", len(stages))
	}
	if stages[0].StageKey != "applied" || stages[0].ConversionRate != 100 {
		t.Fatalf("first stage = %+v, want applied at 100%%", stages[0])
	}
	if stages[1].StageKey != "viewed" || stages[1].ConversionRate != 50 {
		t.Fatalf("second stage = %+v, want viewed at 50%%", stages[1])
	}
	if stages[2].StageKey != "offer_sent" || stages[2].ConversionRate != 80 {
		t.Fatalf("third stage = %+v, want offer_sent at 80%%", stages[2])
	}
	if stages[2].StageLabel != "Offer已发" {
		t.Fatalf("offer_sent label = %q", stages[2].StageLabel)
	}
}

func TestBuildTimeAndOfferRates(t *testing.T) {
	durations := BuildStageDurations([]model.StageDurationRow{{FromStatus: "screening", AvgDurationSecs: 7200, TransitionCount: 3}})
	if durations[0].AvgHours != 2 || durations[0].StageLabel != "筛选中" {
		t.Fatalf("duration = %+v, want 2 hours screening", durations[0])
	}

	metrics := BuildInterviewOfferMetrics(
		model.InterviewMetrics{TotalInterviews: 8, CompletedInterviews: 4, PositiveFeedbacks: 3},
		model.OfferMetrics{TotalOffers: 5, AcceptedOffers: 2, RejectedOffers: 1},
	)
	if metrics.PassRate != 75 || metrics.AcceptanceRate != 40 {
		t.Fatalf("metrics = %+v, want pass 75 and acceptance 40", metrics)
	}
}

func TestValidateProjectionStrategyRejectsTransactionalWrites(t *testing.T) {
	strategy := DefaultProjectionStrategy()
	strategy.TransactionalWrites = true

	err := ValidateProjectionStrategy(strategy)
	if !errors.Is(err, ErrTransactionalWritesForbidden) {
		t.Fatalf("err = %v, want ErrTransactionalWritesForbidden", err)
	}
}

func TestValidateActorAllowsSystemAllMismatch(t *testing.T) {
	err := ValidateActor(7, 9, model.ScopeData{ScopeKeys: []string{model.ScopeSystemAll}})
	if err != nil {
		t.Fatalf("ValidateActor returned %v", err)
	}

	err = ValidateActor(7, 9, model.ScopeData{ScopeKeys: []string{model.ScopeRecruitingAll}})
	if !errors.Is(err, ErrActorMismatch) {
		t.Fatalf("err = %v, want ErrActorMismatch", err)
	}
}
