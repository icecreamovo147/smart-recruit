package port

import (
	"context"
	"time"

	"smart-recruit-notification-service/internal/domain/model"
)

type ActorVerifier interface {
	VerifyActorMatch(ctx context.Context, userID int64) error
}

type UnreadCache interface {
	GetUnreadCount(ctx context.Context, userID uint64, accountType string) (int64, bool)
	SetUnreadCount(ctx context.Context, userID uint64, accountType string, count int64)
	Invalidate(ctx context.Context, userID uint64, accountType string)
}

type RealtimePublisher interface {
	PublishNotificationEvent(ctx context.Context, userID uint64, accountType string, payload string) error
}

type UserDirectory interface {
	GetEmailRecipient(ctx context.Context, userID int64) (*model.EmailRecipient, error)
}

type TemplateData struct {
	RecipientName string
	JobTitle      string
	Content       string
	ActionURL     string
	ActionText    string
	InterviewDate string
	InterviewMode string
	InterviewLink string
	InterviewLoc  string
	OfferTitle    string
	ExpiryDate    string
}

type EmailRenderer interface {
	HasTemplate(notificationType string) bool
	Render(notificationType string, data TemplateData) (subject, html string, err error)
}

type EmailSender interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }
