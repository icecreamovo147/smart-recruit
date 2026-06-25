package email

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

type testSMTPConfig struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	FromAddress string `yaml:"from_address"`
	FromName    string `yaml:"from_name"`
	TLS         bool   `yaml:"tls"`
}

type testConfig struct {
	SMTP            testSMTPConfig `yaml:"smtp"`
	FrontendBaseURL string         `yaml:"frontend_base_url"`
}

func loadTestConfig(t *testing.T) (SMTPConfig, string) {
	t.Helper()

	data, err := os.ReadFile("../config/config.yaml")
	if err != nil {
		t.Fatalf("read config.yaml: %v", err)
	}

	var cfg testConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse config.yaml: %v", err)
	}

	if cfg.SMTP.Host == "" {
		t.Skip("SMTP_HOST is empty, skipping email send test")
	}

	return SMTPConfig{
		Host:        cfg.SMTP.Host,
		Port:        cfg.SMTP.Port,
		Username:    cfg.SMTP.Username,
		Password:    cfg.SMTP.Password,
		FromAddress: cfg.SMTP.FromAddress,
		FromName:    cfg.SMTP.FromName,
		TLS:         cfg.SMTP.TLS,
	}, cfg.FrontendBaseURL
}

// TestSMTPConnection verifies the SMTP server is reachable.
func TestSMTPConnection(t *testing.T) {
	smtpCfg, _ := loadTestConfig(t)

	sender := NewSender(smtpCfg)
	defer sender.Close()

	// If NewSender fell back to LogSender, skip the real send test.
	if _, ok := sender.(*LogSender); ok {
		t.Skip("SMTP fallback to LogSender (host unreachable or not configured)")
	}

	t.Logf("SMTP sender ready: %s:%d", smtpCfg.Host, smtpCfg.Port)
}

// TestRenderAllTemplates verifies all email templates render without error.
func TestRenderAllTemplates(t *testing.T) {
	_, baseURL := loadTestConfig(t)
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}

	r, err := NewRenderer(baseURL)
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}

	types := []string{
		"interview_scheduled",
		"interview_updated",
		"interview_cancelled",
		"offer_sent",
		"offer_withdrawn",
	}

	data := TemplateData{
		RecipientName: "测试用户",
		JobTitle:      "高级工程师",
		Content:       "这是一封测试邮件，用于验证邮件模板渲染功能。",
		ActionURL:     "/applications",
		InterviewDate: time.Now().Add(48 * time.Hour).Format("2006年01月02日 15:04"),
		InterviewMode: "视频面试",
		InterviewLink: "https://meeting.example.com/12345",
		InterviewLoc:  "线上",
		OfferTitle:    "高级工程师 Offer",
		ExpiryDate:    time.Now().Add(7 * 24 * time.Hour).Format("2006-01-02"),
	}

	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			subject, html, err := r.Render(typ, data)
			if err != nil {
				t.Fatalf("render %s: %v", typ, err)
			}
			if subject == "" {
				t.Errorf("empty subject for %s", typ)
			}
			if len(html) < 100 {
				t.Errorf("HTML body too short for %s: %d bytes", typ, len(html))
			}
			t.Logf("[%s] subject=%q html_len=%d", typ, subject, len(html))
		})
	}
}

// TestSendEmail actually sends a test email to verify the full pipeline.
// Set SEND_TEST_EMAIL env var to the recipient address to enable.
func TestSendEmail(t *testing.T) {
	to := os.Getenv("SEND_TEST_EMAIL")
	if to == "" {
		t.Skip("set SEND_TEST_EMAIL env var to a recipient address to enable this test")
	}

	smtpCfg, baseURL := loadTestConfig(t)
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}

	// Bypass NewSender's reachability check — we already tested connection.
	if smtpCfg.Host == "" {
		t.Skip("SMTP host not configured")
	}

	sender := NewSMTPSender(smtpCfg)
	defer sender.Close()

	r, err := NewRenderer(baseURL)
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}

	data := TemplateData{
		RecipientName: "测试用户",
		JobTitle:      "高级工程师",
		Content:       "这是一封自动化测试邮件，用于验证 Smart Recruit 邮件发送链路是否正常。",
		ActionURL:     "/applications",
		InterviewDate: time.Now().Add(48 * time.Hour).Format("2006年01月02日 15:04"),
		InterviewMode: "视频面试",
		InterviewLink: "https://meeting.example.com/12345",
		InterviewLoc:  "线上",
	}

	subject, htmlBody, err := r.Render("interview_scheduled", data)
	if err != nil {
		t.Fatalf("render template: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Logf("sending test email to %s ...", to)
	if err := sender.Send(ctx, to, "[TEST] "+subject, htmlBody); err != nil {
		t.Fatalf("send email failed: %v", err)
	}

	fmt.Printf("\n=== 邮件发送成功 ===\n")
	fmt.Printf("收件人: %s\n", to)
	fmt.Printf("标题: %s\n", subject)
	fmt.Printf("HTML 长度: %d bytes\n", len(htmlBody))
}
