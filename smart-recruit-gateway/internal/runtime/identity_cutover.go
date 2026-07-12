package runtime

import (
	"context"
	"fmt"
)

var IdentitySmokeMethods = []string{
	"AuthService.Login",
	"AuthService.RefreshToken",
	"AuthService.GetPrincipal",
	"AuthService.RecordAuthDecision",
	"AdminService.ListRoles",
	"AdminService.GetUserRoles",
	"AdminService.AssignUserRole",
	"AdminService.QueryAuthAuditLogs",
}

type IdentityCutoverPlan struct {
	Mode         string
	Target       string
	ReadyTargets []RouteEntry
	SmokeMethods []string
}

func BuildIdentityCutoverPlan(ctx context.Context, table RouteTable, logicTarget string, resolver TargetResolver) (IdentityCutoverPlan, error) {
	resolved, err := table.ResolveTargets(ctx, logicTarget, resolver)
	if err != nil {
		return IdentityCutoverPlan{}, err
	}
	entry, ok := resolved.Entry("identity")
	if !ok {
		return IdentityCutoverPlan{}, fmt.Errorf("identity route entry is required")
	}
	return IdentityCutoverPlan{
		Mode:         entry.Mode,
		Target:       entry.Target,
		ReadyTargets: resolved.ReadyTargets(logicTarget),
		SmokeMethods: append([]string(nil), IdentitySmokeMethods...),
	}, nil
}

func (plan IdentityCutoverPlan) ValidateCutover(identityTarget string) error {
	if plan.Mode != "identity" {
		return fmt.Errorf("identity route mode = %q, want identity", plan.Mode)
	}
	if plan.Target != identityTarget {
		return fmt.Errorf("identity route target = %q, want %q", plan.Target, identityTarget)
	}
	if len(plan.SmokeMethods) == 0 {
		return fmt.Errorf("identity smoke methods are required")
	}
	for _, target := range plan.ReadyTargets {
		if target.Service == "identity" && target.Target == identityTarget {
			return nil
		}
	}
	return fmt.Errorf("identity ready target %q is missing", identityTarget)
}

func (plan IdentityCutoverPlan) ValidateRollback(logicTarget string) error {
	if plan.Mode != LogicService {
		return fmt.Errorf("identity rollback mode = %q, want logic", plan.Mode)
	}
	if plan.Target != logicTarget {
		return fmt.Errorf("identity rollback target = %q, want %q", plan.Target, logicTarget)
	}
	for _, target := range plan.ReadyTargets {
		if target.Service == "identity" {
			return fmt.Errorf("identity should not be a separate ready target during logic rollback")
		}
	}
	return nil
}
