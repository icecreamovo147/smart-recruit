package runtime

import (
	"context"
	"testing"
)

func TestInterviewCutoverPlanCoversScheduleFeedbackAndInterviewerTasks(t *testing.T) {
	table, err := NewRouteTable(map[string]string{"interview": "interview"})
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	plan, err := BuildInterviewCutoverPlan(context.Background(), table, "logic:50051", StaticTargetResolver{
		"interview": "interview:50063",
	})
	if err != nil {
		t.Fatalf("BuildInterviewCutoverPlan returned error: %v", err)
	}
	if err := plan.ValidateCutover("interview:50063"); err != nil {
		t.Fatalf("ValidateCutover returned error: %v", err)
	}
	required := map[string]bool{
		"InterviewService.ScheduleInterview":         false,
		"InterviewService.ListInterviewers":          false,
		"InterviewService.GetInterview":              false,
		"InterviewService.ListApplicationInterviews": false,
		"InterviewService.ListMyInterviews":          false,
		"InterviewService.ListCandidateInterviews":   false,
		"InterviewService.SubmitFeedback":            false,
		"InterviewService.GetFeedback":               false,
	}
	for _, method := range plan.SmokeMethods {
		if _, ok := required[method]; ok {
			required[method] = true
		}
	}
	for method, seen := range required {
		if !seen {
			t.Fatalf("interview smoke plan missing %s", method)
		}
	}
}

func TestInterviewRollbackPlanUsesLogicOnly(t *testing.T) {
	table, err := NewRouteTable(map[string]string{"interview": "logic"})
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	plan, err := BuildInterviewCutoverPlan(context.Background(), table, "logic:50051", nil)
	if err != nil {
		t.Fatalf("BuildInterviewCutoverPlan returned error: %v", err)
	}
	if err := plan.ValidateRollback("logic:50051"); err != nil {
		t.Fatalf("ValidateRollback returned error: %v", err)
	}
}
