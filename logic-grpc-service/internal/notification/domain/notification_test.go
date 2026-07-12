package domain

import "testing"

func TestNotificationAccountTypesRemainCompatible(t *testing.T) {
	t.Parallel()

	if AccountTypeCandidate != "candidate" {
		t.Fatalf("candidate account type drifted: %q", AccountTypeCandidate)
	}
	if AccountTypeStaff != "staff" {
		t.Fatalf("staff account type drifted: %q", AccountTypeStaff)
	}
	if AccountTypeService != "service" {
		t.Fatalf("service account type drifted: %q", AccountTypeService)
	}
}

func TestNotificationReadStatesRemainCompatible(t *testing.T) {
	t.Parallel()

	if NotificationUnread != 0 {
		t.Fatalf("unread state drifted: %d", NotificationUnread)
	}
	if NotificationRead != 1 {
		t.Fatalf("read state drifted: %d", NotificationRead)
	}
}

func TestRealtimeEventTypeValuesRemainCompatible(t *testing.T) {
	t.Parallel()

	if RealtimeEventNotificationCreated != "notification_created" {
		t.Fatalf("created realtime event drifted: %q", RealtimeEventNotificationCreated)
	}
}
