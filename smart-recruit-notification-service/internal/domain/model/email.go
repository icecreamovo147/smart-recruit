package model

import "time"

const (
	EmailStatusSent    = "sent"
	EmailStatusSkipped = "skipped"
	EmailStatusFailed  = "failed"
)

type EmailLog struct {
	ID        int64
	EventID   string
	UserID    int64
	Email     string
	Type      string
	Subject   string
	Status    string
	ErrorMsg  string
	SentAt    time.Time
	CreatedAt time.Time
}

type EmailRecipient struct {
	UserID   int64
	Email    string
	Username string
}
