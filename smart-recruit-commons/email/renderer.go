package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strings"
)

//go:embed templates/*.html
var templatesFS embed.FS

// TemplateData holds data passed to email templates.
type TemplateData struct {
	RecipientName string
	JobTitle      string
	Content       string
	ActionURL     string
	ActionText    string
	// Interview-specific
	InterviewDate string
	InterviewMode string
	InterviewLink string
	InterviewLoc  string
	// Offer-specific
	OfferTitle string
	ExpiryDate string
}

// Renderer manages email HTML templates.
type Renderer struct {
	templates map[string]*template.Template
	baseURL   string
}

// NewRenderer loads embedded templates and returns a Renderer.
func NewRenderer(baseURL string) (*Renderer, error) {
	r := &Renderer{
		templates: make(map[string]*template.Template),
		baseURL:   strings.TrimRight(baseURL, "/"),
	}

	// Parse the base layout.
	layoutTmpl, err := template.ParseFS(templatesFS, "templates/layout.html")
	if err != nil {
		return nil, fmt.Errorf("parse layout: %w", err)
	}

	// Map notification types to template files.
	typeTemplateMap := map[string]string{
		"interview_scheduled":   "templates/interview_scheduled.html",
		"interview_updated":     "templates/interview_updated.html",
		"interview_cancelled":   "templates/interview_cancelled.html",
		"offer_sent":            "templates/offer_sent.html",
		"offer_withdrawn":       "templates/offer_withdrawn.html",
		"interview_assigned":    "templates/interview_assigned.html",
		"new_application":       "templates/new_application.html",
		"application_approved":  "templates/application_approved.html",
		"application_rejected":  "templates/application_rejected.html",
		"application_withdrawn": "templates/application_withdrawn.html",
		"application_hired":     "templates/application_hired.html",
	}

	for notifType, tmplFile := range typeTemplateMap {
		tmpl, err := layoutTmpl.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone layout for %s: %w", notifType, err)
		}
		_, err = tmpl.ParseFS(templatesFS, tmplFile)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", tmplFile, err)
		}
		r.templates[notifType] = tmpl
	}

	return r, nil
}

// Render generates the subject and HTML body for a given notification type.
func (r *Renderer) Render(notificationType string, data TemplateData) (subject, html string, err error) {
	tmpl, ok := r.templates[notificationType]
	if !ok {
		return "", "", fmt.Errorf("no email template for notification type: %s", notificationType)
	}

	// Build full action URL.
	if data.ActionURL != "" && !strings.HasPrefix(data.ActionURL, "http") {
		data.ActionURL = r.baseURL + data.ActionURL
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout", data); err != nil {
		return "", "", fmt.Errorf("render template %s: %w", notificationType, err)
	}

	subject = subjectForType(notificationType, data)
	return subject, buf.String(), nil
}

// HasTemplate checks if a template exists for the given notification type.
func (r *Renderer) HasTemplate(notificationType string) bool {
	_, ok := r.templates[notificationType]
	return ok
}

func subjectForType(notificationType string, data TemplateData) string {
	switch notificationType {
	case "interview_scheduled":
		return fmt.Sprintf("面试安排通知 — %s", data.JobTitle)
	case "interview_updated":
		return fmt.Sprintf("面试时间变更通知 — %s", data.JobTitle)
	case "interview_cancelled":
		return fmt.Sprintf("面试取消通知 — %s", data.JobTitle)
	case "offer_sent":
		return fmt.Sprintf("您收到了一份 Offer — %s", data.JobTitle)
	case "offer_withdrawn":
		return fmt.Sprintf("Offer 已撤回 — %s", data.JobTitle)
	case "interview_assigned":
		return fmt.Sprintf("新的面试安排 — %s", data.JobTitle)
	case "new_application":
		return fmt.Sprintf("新的投递 — %s", data.JobTitle)
	case "application_approved":
		return fmt.Sprintf("投递通过筛选 — %s", data.JobTitle)
	case "application_rejected":
		return fmt.Sprintf("投递进展 — %s", data.JobTitle)
	case "application_withdrawn":
		return fmt.Sprintf("投递已撤回 — %s", data.JobTitle)
	case "application_hired":
		return fmt.Sprintf("入职确认 — %s", data.JobTitle)
	default:
		return "Smart Recruit 通知"
	}
}
