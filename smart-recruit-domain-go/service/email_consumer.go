package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"smart-recruit-domain-go/email"
	"smart-recruit-domain-go/model"
	"smart-recruit-domain-go/mq"
	"smart-recruit-domain-go/repository"
	"smart-recruit-platform-go/logger"
)

// emailPayload is the MQ message structure for email events.
type emailPayload struct {
	EventID             string `json:"event_id"`
	ReceiverID          int64  `json:"receiver_id"`
	ReceiverAccountType string `json:"receiver_account_type"`
	Type                string `json:"type"`
	Title               string `json:"title"`
	Content             string `json:"content"`
	Link                string `json:"link"`
	BizType             string `json:"biz_type"`
	BizID               int64  `json:"biz_id"`
	// Extra fields for template rendering
	JobTitle      string `json:"job_title,omitempty"`
	RecipientName string `json:"recipient_name,omitempty"`
	InterviewDate string `json:"interview_date,omitempty"`
	InterviewMode string `json:"interview_mode,omitempty"`
	InterviewLink string `json:"interview_link,omitempty"`
	InterviewLoc  string `json:"interview_loc,omitempty"`
	OfferTitle    string `json:"offer_title,omitempty"`
	ExpiryDate    string `json:"expiry_date,omitempty"`
}

// EmailConsumer consumes email.send events from RabbitMQ and sends HTML emails.
type EmailConsumer struct {
	userRepo *repository.UserRepo
	emailLog *repository.EmailLogRepo
	renderer *email.Renderer
	sender   email.Sender
	inbox    *repository.InboxRepo
}

// NewEmailConsumer creates a new EmailConsumer.
func NewEmailConsumer(
	userRepo *repository.UserRepo,
	emailLog *repository.EmailLogRepo,
	renderer *email.Renderer,
	sender email.Sender,
) *EmailConsumer {
	return &EmailConsumer{
		userRepo: userRepo,
		emailLog: emailLog,
		renderer: renderer,
		sender:   sender,
	}
}

func (c *EmailConsumer) WithInbox(inbox *repository.InboxRepo) *EmailConsumer {
	c.inbox = inbox
	return c
}

// Start registers the consumer on the email queue.
func (c *EmailConsumer) Start(ctx context.Context, mqConn *mq.Conn) error {
	return mqConn.Consume(ctx, mqConn.EmailQueue(), func(ctx context.Context, body []byte) error {
		return consumeWithInbox(ctx, c.inbox, "email-consumer", body, func() error {
			return c.handle(ctx, body)
		})
	})
}

func (c *EmailConsumer) handle(ctx context.Context, body []byte) error {
	var p emailPayload
	if err := json.Unmarshal(body, &p); err != nil {
		logger.L().Error("email consumer: invalid payload", zap.Error(err))
		return fmt.Errorf("invalid payload: %w", err)
	}

	// Check if template exists for this notification type.
	if !c.renderer.HasTemplate(p.Type) {
		logger.L().Info("email consumer: no template for type, skipping",
			zap.String("type", p.Type), zap.String("event_id", p.EventID))
		return nil
	}

	// Idempotency check: skip if already sent.
	if p.EventID != "" {
		exists, err := c.emailLog.ExistsByEventID(ctx, p.EventID)
		if err != nil {
			logger.L().Warn("email consumer: idempotency check failed", zap.Error(err))
			return err
		}
		if exists {
			logger.L().Info("email consumer: duplicate event, skipping",
				zap.String("event_id", p.EventID))
			// Record as skipped; if logging fails, warn but do not retry (skip is correct).
			if err := c.recordLog(ctx, p, "", "skipped", nil); err != nil {
				logger.L().Warn("email consumer: failed to record skip log", zap.Error(err))
			}
			return nil
		}
	}

	// Lookup receiver email.
	user, err := c.userRepo.GetByID(ctx, p.ReceiverID)
	if err != nil {
		logger.L().Error("email consumer: lookup user failed",
			zap.Int64("receiver_id", p.ReceiverID), zap.Error(err))
		return err
	}
	if user == nil || user.Email == "" {
		logger.L().Info("email consumer: user has no email, skipping",
			zap.Int64("receiver_id", p.ReceiverID), zap.String("type", p.Type))
		// Record as skipped.
		if err := c.recordLog(ctx, p, "", "skipped", nil); err != nil {
			logger.L().Warn("email consumer: failed to record skip log", zap.Error(err))
		}
		return nil
	}

	// Build template data.
	recipientName := p.RecipientName
	if recipientName == "" {
		recipientName = user.Username
	}
	data := email.TemplateData{
		RecipientName: recipientName,
		JobTitle:      p.JobTitle,
		Content:       p.Content,
		ActionURL:     p.Link,
		InterviewDate: p.InterviewDate,
		InterviewMode: p.InterviewMode,
		InterviewLink: p.InterviewLink,
		InterviewLoc:  p.InterviewLoc,
		OfferTitle:    p.OfferTitle,
		ExpiryDate:    p.ExpiryDate,
	}

	// Set action button text based on type.
	switch p.Type {
	case "interview_scheduled":
		data.ActionText = "查看面试详情"
	case "interview_updated":
		data.ActionText = "查看最新安排"
	case "interview_cancelled":
		data.ActionText = "查看详情"
	case "offer_sent":
		data.ActionText = "查看并处理 Offer"
	case "offer_withdrawn":
		data.ActionText = "查看详情"
	case "interview_assigned":
		data.ActionText = "查看面试详情"
	case "new_application":
		data.ActionText = "查看简历"
	case "application_approved":
		data.ActionText = "查看详情"
	case "application_rejected":
		data.ActionText = "查看详情"
	case "application_withdrawn":
		data.ActionText = "查看投递"
	case "application_hired":
		data.ActionText = "查看详情"
	}

	// Render email.
	subject, htmlBody, err := c.renderer.Render(p.Type, data)
	if err != nil {
		logger.L().Error("email consumer: render template failed",
			zap.String("type", p.Type), zap.Error(err))
		return err
	}

	// Send email.
	if err := c.sender.Send(ctx, user.Email, subject, htmlBody); err != nil {
		logger.L().Error("email consumer: send failed",
			zap.String("to", email.MaskEmail(user.Email)), zap.String("type", p.Type), zap.Error(err))
		if err := c.recordLog(ctx, p, user.Email, "failed", &err); err != nil {
			return err
		}
		return err
	}

	logger.L().Info("email sent successfully",
		zap.String("to", email.MaskEmail(user.Email)),
		zap.String("type", p.Type),
		zap.String("event_id", p.EventID),
	)
	if err := c.recordLog(ctx, p, user.Email, "sent", nil); err != nil {
		return err
	}
	return nil
}

func (c *EmailConsumer) recordLog(ctx context.Context, p emailPayload, addr, status string, sendErr *error) error {
	log := &model.EmailLog{
		EventID: p.EventID,
		UserID:  p.ReceiverID,
		Email:   addr,
		Type:    p.Type,
		Subject: p.Title,
		Status:  status,
		SentAt:  time.Now(),
	}
	if sendErr != nil {
		msg := (*sendErr).Error()
		log.ErrorMsg = &msg
	}
	if err := c.emailLog.Create(ctx, log); err != nil {
		logger.L().Warn("email consumer: record log failed", zap.Error(err))
		return err
	}
	return nil
}
