package runtime

import (
	"context"
	"fmt"
)

var RecruitmentSmokeMethods = []string{
	"JobService.ListPublicJobs",
	"JobService.GetJobDetail",
	"JobService.ListHRJobs",
	"CandidateService.GetProfile",
	"CandidateService.GetResume",
	"ApplicationService.ApplyJob",
	"ApplicationService.ListMyApplications",
	"ApplicationService.ListJobApplications",
}

type RecruitmentCutoverPlan struct {
	Mode         string
	Target       string
	ReadyTargets []RouteEntry
	SmokeMethods []string
}

func BuildRecruitmentCutoverPlan(ctx context.Context, table RouteTable, logicTarget string, resolver TargetResolver) (RecruitmentCutoverPlan, error) {
	resolved, err := table.ResolveTargets(ctx, logicTarget, resolver)
	if err != nil {
		return RecruitmentCutoverPlan{}, err
	}
	entry, ok := resolved.Entry("recruitment")
	if !ok {
		return RecruitmentCutoverPlan{}, fmt.Errorf("recruitment route entry is required")
	}
	return RecruitmentCutoverPlan{Mode: entry.Mode, Target: entry.Target, ReadyTargets: resolved.ReadyTargets(logicTarget), SmokeMethods: append([]string(nil), RecruitmentSmokeMethods...)}, nil
}

func (plan RecruitmentCutoverPlan) ValidateCutover(recruitmentTarget string) error {
	if plan.Mode != "recruitment" {
		return fmt.Errorf("recruitment route mode = %q, want recruitment", plan.Mode)
	}
	if plan.Target != recruitmentTarget {
		return fmt.Errorf("recruitment route target = %q, want %q", plan.Target, recruitmentTarget)
	}
	for _, target := range plan.ReadyTargets {
		if target.Service == "recruitment" && target.Target == recruitmentTarget {
			return nil
		}
	}
	return fmt.Errorf("recruitment ready target %q is missing", recruitmentTarget)
}

func (plan RecruitmentCutoverPlan) ValidateRollback(logicTarget string) error {
	if plan.Mode != LogicService {
		return fmt.Errorf("recruitment rollback mode = %q, want logic", plan.Mode)
	}
	if plan.Target != logicTarget {
		return fmt.Errorf("recruitment rollback target = %q, want %q", plan.Target, logicTarget)
	}
	for _, target := range plan.ReadyTargets {
		if target.Service == "recruitment" {
			return fmt.Errorf("recruitment should not be a separate ready target during logic rollback")
		}
	}
	return nil
}
