package runtime

import (
	"context"
	"fmt"
)

var InterviewSmokeMethods = []string{
	"InterviewService.ScheduleInterview",
	"InterviewService.ListInterviewers",
	"InterviewService.GetInterview",
	"InterviewService.ListApplicationInterviews",
	"InterviewService.ListMyInterviews",
	"InterviewService.ListCandidateInterviews",
	"InterviewService.SubmitFeedback",
	"InterviewService.GetFeedback",
}

type InterviewCutoverPlan struct {
	Mode         string
	Target       string
	ReadyTargets []RouteEntry
	SmokeMethods []string
}

func BuildInterviewCutoverPlan(ctx context.Context, table RouteTable, logicTarget string, resolver TargetResolver) (InterviewCutoverPlan, error) {
	resolved, err := table.ResolveTargets(ctx, logicTarget, resolver)
	if err != nil {
		return InterviewCutoverPlan{}, err
	}
	entry, ok := resolved.Entry("interview")
	if !ok {
		return InterviewCutoverPlan{}, fmt.Errorf("interview route entry is required")
	}
	return InterviewCutoverPlan{
		Mode:         entry.Mode,
		Target:       entry.Target,
		ReadyTargets: resolved.ReadyTargets(logicTarget),
		SmokeMethods: append([]string(nil), InterviewSmokeMethods...),
	}, nil
}

func (plan InterviewCutoverPlan) ValidateCutover(interviewTarget string) error {
	if plan.Mode != "interview" {
		return fmt.Errorf("interview route mode = %q, want interview", plan.Mode)
	}
	if plan.Target != interviewTarget {
		return fmt.Errorf("interview route target = %q, want %q", plan.Target, interviewTarget)
	}
	if len(plan.SmokeMethods) == 0 {
		return fmt.Errorf("interview smoke methods are required")
	}
	for _, target := range plan.ReadyTargets {
		if target.Service == "interview" && target.Target == interviewTarget {
			return nil
		}
	}
	return fmt.Errorf("interview ready target %q is missing", interviewTarget)
}

func (plan InterviewCutoverPlan) ValidateRollback(logicTarget string) error {
	if plan.Mode != LogicService {
		return fmt.Errorf("interview rollback mode = %q, want logic", plan.Mode)
	}
	if plan.Target != logicTarget {
		return fmt.Errorf("interview rollback target = %q, want %q", plan.Target, logicTarget)
	}
	for _, target := range plan.ReadyTargets {
		if target.Service == "interview" {
			return fmt.Errorf("interview should not be a separate ready target during logic rollback")
		}
	}
	return nil
}
