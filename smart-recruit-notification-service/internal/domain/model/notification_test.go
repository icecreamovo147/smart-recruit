package model

import (
	"testing"
	"time"
)

func TestNotificationDefaultsAndReadState(t *testing.T) {
	n := Notification{}
	n.NormalizeAccountType()
	if n.ReceiverAccountType != DefaultAccountType {
		t.Fatalf("ReceiverAccountType = %q, want %q", n.ReceiverAccountType, DefaultAccountType)
	}

	now := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	if !n.MarkRead(now) {
		t.Fatal("first MarkRead should report a state change")
	}
	if !n.IsRead || n.ReadAt == nil || !n.ReadAt.Equal(now) {
		t.Fatalf("unexpected read state: %#v", n)
	}
	if n.MarkRead(now.Add(time.Hour)) {
		t.Fatal("second MarkRead should be idempotent")
	}
	if !n.ReadAt.Equal(now) {
		t.Fatalf("ReadAt changed on idempotent mark: %v", n.ReadAt)
	}
}

func TestEnsureCreatedAtOnlySetsMissingTimestamp(t *testing.T) {
	initial := time.Date(2026, 7, 13, 11, 0, 0, 0, time.UTC)
	next := initial.Add(time.Hour)
	n := Notification{CreatedAt: initial}
	n.EnsureCreatedAt(next)
	if !n.CreatedAt.Equal(initial) {
		t.Fatalf("CreatedAt = %v, want original %v", n.CreatedAt, initial)
	}

	n = Notification{}
	n.EnsureCreatedAt(next)
	if !n.CreatedAt.Equal(next) {
		t.Fatalf("CreatedAt = %v, want %v", n.CreatedAt, next)
	}
}
