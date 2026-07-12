package runtime

import (
	"context"
	"fmt"
)

var NotificationSmokeMethods = []string{
	"NotificationService.ListNotifications",
	"NotificationService.UnreadNotificationCount",
	"NotificationService.NotificationSummary",
	"NotificationService.MarkNotificationRead",
	"NotificationService.MarkAllNotificationsRead",
	"Realtime.NotificationEventChannel",
}

type NotificationCutoverPlan struct {
	Mode         string
	Target       string
	ReadyTargets []RouteEntry
	SmokeMethods []string
}

func BuildNotificationCutoverPlan(ctx context.Context, table RouteTable, logicTarget string, resolver TargetResolver) (NotificationCutoverPlan, error) {
	resolved, err := table.ResolveTargets(ctx, logicTarget, resolver)
	if err != nil {
		return NotificationCutoverPlan{}, err
	}
	entry, ok := resolved.Entry("notification")
	if !ok {
		return NotificationCutoverPlan{}, fmt.Errorf("notification route entry is required")
	}
	return NotificationCutoverPlan{
		Mode:         entry.Mode,
		Target:       entry.Target,
		ReadyTargets: resolved.ReadyTargets(logicTarget),
		SmokeMethods: append([]string(nil), NotificationSmokeMethods...),
	}, nil
}

func (plan NotificationCutoverPlan) ValidateCutover(notificationTarget string) error {
	if plan.Mode != "notification" {
		return fmt.Errorf("notification route mode = %q, want notification", plan.Mode)
	}
	if plan.Target != notificationTarget {
		return fmt.Errorf("notification route target = %q, want %q", plan.Target, notificationTarget)
	}
	if len(plan.SmokeMethods) == 0 {
		return fmt.Errorf("notification smoke methods are required")
	}
	for _, target := range plan.ReadyTargets {
		if target.Service == "notification" && target.Target == notificationTarget {
			return nil
		}
	}
	return fmt.Errorf("notification ready target %q is missing", notificationTarget)
}

func (plan NotificationCutoverPlan) ValidateRollback(logicTarget string) error {
	if plan.Mode != LogicService {
		return fmt.Errorf("notification rollback mode = %q, want logic", plan.Mode)
	}
	if plan.Target != logicTarget {
		return fmt.Errorf("notification rollback target = %q, want %q", plan.Target, logicTarget)
	}
	for _, target := range plan.ReadyTargets {
		if target.Service == "notification" {
			return fmt.Errorf("notification should not be a separate ready target during logic rollback")
		}
	}
	return nil
}
