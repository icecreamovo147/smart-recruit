package email

import (
	"context"

	sharedemail "smart-recruit-domain-go/email"
	"smart-recruit-notification-service/internal/application/port"
)

type Renderer struct {
	renderer *sharedemail.Renderer
}

func NewRenderer(renderer *sharedemail.Renderer) *Renderer {
	return &Renderer{renderer: renderer}
}

func (r *Renderer) HasTemplate(notificationType string) bool {
	return r != nil && r.renderer != nil && r.renderer.HasTemplate(notificationType)
}

func (r *Renderer) Render(notificationType string, data port.TemplateData) (string, string, error) {
	return r.renderer.Render(notificationType, sharedemail.TemplateData{
		RecipientName: data.RecipientName,
		JobTitle:      data.JobTitle,
		Content:       data.Content,
		ActionURL:     data.ActionURL,
		ActionText:    data.ActionText,
		InterviewDate: data.InterviewDate,
		InterviewMode: data.InterviewMode,
		InterviewLink: data.InterviewLink,
		InterviewLoc:  data.InterviewLoc,
		OfferTitle:    data.OfferTitle,
		ExpiryDate:    data.ExpiryDate,
	})
}

type Sender struct {
	sender sharedemail.Sender
}

func NewSender(sender sharedemail.Sender) *Sender {
	return &Sender{sender: sender}
}

func (s *Sender) Send(ctx context.Context, to, subject, htmlBody string) error {
	if s == nil || s.sender == nil {
		return nil
	}
	return s.sender.Send(ctx, to, subject, htmlBody)
}
