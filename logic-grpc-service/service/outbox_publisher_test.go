package service

import (
	"encoding/json"
	"testing"
)

func TestBuildOutboxEventInjectsEventIDIntoPayload(t *testing.T) {
	event, err := buildOutboxEvent("notification.create", "application", 1, "notification.create", notificationPayload{
		ReceiverID:   1,
		ReceiverRole: 1,
		Type:         "application_approved",
		Title:        "投递进展更新",
		Content:      "通过筛选",
		BizType:      "application",
		BizID:        10,
	})
	if err != nil {
		t.Fatalf("build outbox event: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
		t.Fatalf("payload should be json: %v", err)
	}
	if payload["event_id"] != event.EventID {
		t.Fatalf("payload event_id=%v, want %s", payload["event_id"], event.EventID)
	}
}
