package service

import (
	"context"

	"smart-recruit-notification-service/internal/application/command"
	"smart-recruit-notification-service/internal/application/port"
	"smart-recruit-notification-service/internal/domain/model"
)

func (s *Service) HandleEmailMessage(ctx context.Context, msg command.EmailMessage) error {
	if s.renderer == nil || !s.renderer.HasTemplate(msg.Type) {
		return nil
	}
	if msg.EventID != "" && s.emailLogs != nil {
		exists, err := s.emailLogs.ExistsByEventID(ctx, msg.EventID)
		if err != nil {
			return err
		}
		if exists {
			_ = s.recordEmailLog(ctx, msg, "", model.EmailStatusSkipped, "")
			return nil
		}
	}
	recipient, err := s.users.GetEmailRecipient(ctx, msg.ReceiverID)
	if err != nil {
		return err
	}
	if recipient == nil || recipient.Email == "" {
		_ = s.recordEmailLog(ctx, msg, "", model.EmailStatusSkipped, "")
		return nil
	}
	data := emailTemplateData(msg, recipient)
	subject, html, err := s.renderer.Render(msg.Type, data)
	if err != nil {
		return err
	}
	if err := s.sender.Send(ctx, recipient.Email, subject, html); err != nil {
		if logErr := s.recordEmailLog(ctx, msg, recipient.Email, model.EmailStatusFailed, err.Error()); logErr != nil {
			return logErr
		}
		return err
	}
	return s.recordEmailLog(ctx, msg, recipient.Email, model.EmailStatusSent, "")
}

func (s *Service) recordEmailLog(ctx context.Context, msg command.EmailMessage, addr, status, errMsg string) error {
	if s.emailLogs == nil {
		return nil
	}
	log := &model.EmailLog{
		EventID: msg.EventID,
		UserID:  msg.ReceiverID,
		Email:   addr,
		Type:    msg.Type,
		Subject: msg.Title,
		Status:  status,
		SentAt:  s.clock.Now(),
	}
	if errMsg != "" {
		log.ErrorMsg = errMsg
	}
	return s.emailLogs.Create(ctx, log)
}

func emailTemplateData(msg command.EmailMessage, recipient *model.EmailRecipient) port.TemplateData {
	recipientName := msg.RecipientName
	if recipientName == "" {
		recipientName = recipient.Username
	}
	return port.TemplateData{
		RecipientName: recipientName,
		JobTitle:      msg.JobTitle,
		Content:       msg.Content,
		ActionURL:     msg.Link,
		ActionText:    emailActionText(msg.Type),
		InterviewDate: msg.InterviewDate,
		InterviewMode: msg.InterviewMode,
		InterviewLink: msg.InterviewLink,
		InterviewLoc:  msg.InterviewLoc,
		OfferTitle:    msg.OfferTitle,
		ExpiryDate:    msg.ExpiryDate,
	}
}

func emailActionText(notificationType string) string {
	switch notificationType {
	case "interview_scheduled":
		return "查看面试详情"
	case "interview_updated":
		return "查看最新安排"
	case "interview_cancelled":
		return "查看详情"
	case "offer_sent":
		return "查看并处理 Offer"
	case "offer_withdrawn":
		return "查看详情"
	case "interview_assigned":
		return "查看面试详情"
	case "new_application":
		return "查看简历"
	case "application_approved":
		return "查看详情"
	case "application_rejected":
		return "查看详情"
	case "application_withdrawn":
		return "查看投递"
	case "application_hired":
		return "查看详情"
	default:
		return ""
	}
}
