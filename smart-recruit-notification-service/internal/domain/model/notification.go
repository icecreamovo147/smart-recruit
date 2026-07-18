package model

import "time"

const DefaultAccountType = "candidate"

type Notification struct {
	ID                  int64
	EventID             string
	ReceiverID          int64
	ReceiverAccountType string
	ReceiverRole        int32
	Type                string
	Title               string
	Content             string
	Link                string
	BizType             string
	BizID               int64
	IsRead              bool
	CreatedAt           time.Time
	ReadAt              *time.Time
}

func (n *Notification) NormalizeAccountType() {
	if n.ReceiverAccountType == "" {
		n.ReceiverAccountType = DefaultAccountType
	}
}

func (n *Notification) EnsureCreatedAt(now time.Time) {
	if n.CreatedAt.IsZero() {
		n.CreatedAt = now
	}
}

func (n *Notification) MarkRead(now time.Time) bool {
	if n.IsRead {
		return false
	}
	n.IsRead = true
	n.ReadAt = &now
	return true
}

type RealtimeEvent struct {
	Type             string `json:"type"`
	NotificationType string `json:"notification_type"`
	NotificationID   int64  `json:"notification_id"`
	Unread           int64  `json:"unread"`
	Title            string `json:"title"`
	Content          string `json:"content"`
	Link             string `json:"link"`
	CreatedAt        string `json:"created_at"`
}
