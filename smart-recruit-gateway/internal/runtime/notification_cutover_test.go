package runtime

import (
	"context"
	"testing"
)

func TestNotificationCutoverPlanCoversListUnreadAndRealtimeSmoke(t *testing.T) {
	table, err := NewRouteTable(map[string]string{"notification": "notification"})
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	plan, err := BuildNotificationCutoverPlan(context.Background(), table, "logic:50051", StaticTargetResolver{
		"notification": "notification:50065",
	})
	if err != nil {
		t.Fatalf("BuildNotificationCutoverPlan returned error: %v", err)
	}
	if err := plan.ValidateCutover("notification:50065"); err != nil {
		t.Fatalf("ValidateCutover returned error: %v", err)
	}
	required := map[string]bool{
		"NotificationService.ListNotifications":        false,
		"NotificationService.UnreadNotificationCount":  false,
		"NotificationService.NotificationSummary":      false,
		"NotificationService.MarkNotificationRead":     false,
		"NotificationService.MarkAllNotificationsRead": false,
		"Realtime.NotificationEventChannel":            false,
	}
	for _, method := range plan.SmokeMethods {
		if _, ok := required[method]; ok {
			required[method] = true
		}
	}
	for method, seen := range required {
		if !seen {
			t.Fatalf("notification smoke plan missing %s", method)
		}
	}
}

func TestNotificationRollbackPlanUsesLogicOnly(t *testing.T) {
	table, err := NewRouteTable(map[string]string{"notification": "logic"})
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	plan, err := BuildNotificationCutoverPlan(context.Background(), table, "logic:50051", nil)
	if err != nil {
		t.Fatalf("BuildNotificationCutoverPlan returned error: %v", err)
	}
	if err := plan.ValidateRollback("logic:50051"); err != nil {
		t.Fatalf("ValidateRollback returned error: %v", err)
	}
}
