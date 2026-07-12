package runtime

import (
	"context"
	"fmt"
)

var AnalyticsSmokeMethods = []string{
	"AdminService.GetDashboardReport",
	"AdminService.GetFunnelReport",
	"AdminService.GetTimeInStageReport",
	"AdminService.GetInterviewOfferMetrics",
}

type AnalyticsCutoverPlan struct {
	Mode         string
	Target       string
	ReadyTargets []RouteEntry
	SmokeMethods []string
}

func BuildAnalyticsCutoverPlan(ctx context.Context, table RouteTable, logicTarget string, resolver TargetResolver) (AnalyticsCutoverPlan, error) {
	resolved, err := table.ResolveTargets(ctx, logicTarget, resolver)
	if err != nil {
		return AnalyticsCutoverPlan{}, err
	}
	entry, ok := resolved.Entry("analytics")
	if !ok {
		return AnalyticsCutoverPlan{}, fmt.Errorf("analytics route entry is required")
	}
	return AnalyticsCutoverPlan{
		Mode:         entry.Mode,
		Target:       entry.Target,
		ReadyTargets: resolved.ReadyTargets(logicTarget),
		SmokeMethods: append([]string(nil), AnalyticsSmokeMethods...),
	}, nil
}

func (plan AnalyticsCutoverPlan) ValidateCutover(analyticsTarget string) error {
	if plan.Mode != "analytics" {
		return fmt.Errorf("analytics route mode = %q, want analytics", plan.Mode)
	}
	if plan.Target != analyticsTarget {
		return fmt.Errorf("analytics route target = %q, want %q", plan.Target, analyticsTarget)
	}
	if len(plan.SmokeMethods) == 0 {
		return fmt.Errorf("analytics smoke methods are required")
	}
	for _, target := range plan.ReadyTargets {
		if target.Service == "analytics" && target.Target == analyticsTarget {
			return nil
		}
	}
	return fmt.Errorf("analytics ready target %q is missing", analyticsTarget)
}

func (plan AnalyticsCutoverPlan) ValidateRollback(logicTarget string) error {
	if plan.Mode != LogicService {
		return fmt.Errorf("analytics rollback mode = %q, want logic", plan.Mode)
	}
	if plan.Target != logicTarget {
		return fmt.Errorf("analytics rollback target = %q, want %q", plan.Target, logicTarget)
	}
	for _, target := range plan.ReadyTargets {
		if target.Service == "analytics" {
			return fmt.Errorf("analytics should not be a separate ready target during logic rollback")
		}
	}
	return nil
}
