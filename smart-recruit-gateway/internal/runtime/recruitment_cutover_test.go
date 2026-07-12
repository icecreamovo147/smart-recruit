package runtime

import (
	"context"
	"testing"
)

func TestRecruitmentCutoverPlanCoversJobsProfileResumeAndApplication(t *testing.T) {
	table, err := NewRouteTable(map[string]string{"recruitment": "recruitment"})
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	plan, err := BuildRecruitmentCutoverPlan(context.Background(), table, "logic:50051", StaticTargetResolver{
		"recruitment": "recruitment:50062",
	})
	if err != nil {
		t.Fatalf("BuildRecruitmentCutoverPlan returned error: %v", err)
	}
	if err := plan.ValidateCutover("recruitment:50062"); err != nil {
		t.Fatalf("ValidateCutover returned error: %v", err)
	}
	required := map[string]bool{
		"JobService.ListPublicJobs":              false,
		"JobService.GetJobDetail":                false,
		"JobService.ListHRJobs":                  false,
		"CandidateService.GetProfile":            false,
		"CandidateService.GetResume":             false,
		"ApplicationService.ApplyJob":            false,
		"ApplicationService.ListMyApplications":  false,
		"ApplicationService.ListJobApplications": false,
	}
	for _, method := range plan.SmokeMethods {
		if _, ok := required[method]; ok {
			required[method] = true
		}
	}
	for method, seen := range required {
		if !seen {
			t.Fatalf("recruitment smoke plan missing %s", method)
		}
	}
}

func TestRecruitmentRollbackPlanUsesLogicOnly(t *testing.T) {
	table, err := NewRouteTable(map[string]string{"recruitment": "logic"})
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	plan, err := BuildRecruitmentCutoverPlan(context.Background(), table, "logic:50051", nil)
	if err != nil {
		t.Fatalf("BuildRecruitmentCutoverPlan returned error: %v", err)
	}
	if err := plan.ValidateRollback("logic:50051"); err != nil {
		t.Fatalf("ValidateRollback returned error: %v", err)
	}
}
