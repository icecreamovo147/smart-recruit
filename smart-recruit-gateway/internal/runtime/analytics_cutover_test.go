package runtime

import (
	"context"
	"testing"
)

func TestAnalyticsCutoverPlanCoversDashboardFunnelStageAndMetricsSmoke(t *testing.T) {
	table, err := NewRouteTable(map[string]string{"analytics": "analytics"})
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	plan, err := BuildAnalyticsCutoverPlan(context.Background(), table, "logic:50051", StaticTargetResolver{
		"analytics": "analytics:50067",
	})
	if err != nil {
		t.Fatalf("BuildAnalyticsCutoverPlan returned error: %v", err)
	}
	if err := plan.ValidateCutover("analytics:50067"); err != nil {
		t.Fatalf("ValidateCutover returned error: %v", err)
	}
	required := map[string]bool{
		"AdminService.GetDashboardReport":       false,
		"AdminService.GetFunnelReport":          false,
		"AdminService.GetTimeInStageReport":     false,
		"AdminService.GetInterviewOfferMetrics": false,
	}
	for _, method := range plan.SmokeMethods {
		if _, ok := required[method]; ok {
			required[method] = true
		}
	}
	for method, seen := range required {
		if !seen {
			t.Fatalf("analytics smoke plan missing %s", method)
		}
	}
}

func TestAnalyticsRollbackPlanUsesLogicOnly(t *testing.T) {
	table, err := NewRouteTable(map[string]string{"analytics": "logic"})
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	plan, err := BuildAnalyticsCutoverPlan(context.Background(), table, "logic:50051", nil)
	if err != nil {
		t.Fatalf("BuildAnalyticsCutoverPlan returned error: %v", err)
	}
	if err := plan.ValidateRollback("logic:50051"); err != nil {
		t.Fatalf("ValidateRollback returned error: %v", err)
	}
}
