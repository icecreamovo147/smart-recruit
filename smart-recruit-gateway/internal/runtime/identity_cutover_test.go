package runtime

import (
	"context"
	"testing"
)

func TestIdentityCutoverPlanCoversAuthRBACPrincipalAndAudit(t *testing.T) {
	table, err := NewRouteTable(map[string]string{"identity": "identity"})
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	plan, err := BuildIdentityCutoverPlan(context.Background(), table, "logic:50051", StaticTargetResolver{
		"identity": "identity:50061",
	})
	if err != nil {
		t.Fatalf("BuildIdentityCutoverPlan returned error: %v", err)
	}
	if err := plan.ValidateCutover("identity:50061"); err != nil {
		t.Fatalf("ValidateCutover returned error: %v", err)
	}
	required := map[string]bool{
		"AuthService.Login":               false,
		"AuthService.RefreshToken":        false,
		"AuthService.GetPrincipal":        false,
		"AuthService.RecordAuthDecision":  false,
		"AdminService.ListRoles":          false,
		"AdminService.GetUserRoles":       false,
		"AdminService.AssignUserRole":     false,
		"AdminService.QueryAuthAuditLogs": false,
	}
	for _, method := range plan.SmokeMethods {
		if _, ok := required[method]; ok {
			required[method] = true
		}
	}
	for method, seen := range required {
		if !seen {
			t.Fatalf("identity smoke plan missing %s", method)
		}
	}
}

func TestIdentityRollbackPlanUsesLogicOnly(t *testing.T) {
	table, err := NewRouteTable(map[string]string{"identity": "logic"})
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	plan, err := BuildIdentityCutoverPlan(context.Background(), table, "logic:50051", nil)
	if err != nil {
		t.Fatalf("BuildIdentityCutoverPlan returned error: %v", err)
	}
	if err := plan.ValidateRollback("logic:50051"); err != nil {
		t.Fatalf("ValidateRollback returned error: %v", err)
	}
}
