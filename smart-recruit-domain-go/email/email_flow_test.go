package email

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type testMySQLConfig struct {
	DSN string `yaml:"dsn"`
}

type fullTestConfig struct {
	MySQL           testMySQLConfig `yaml:"mysql"`
	SMTP            testSMTPConfig  `yaml:"smtp"`
	FrontendBaseURL string          `yaml:"frontend_base_url"`
}

type testUser struct {
	ID       int64  `gorm:"column:id"`
	Username string `gorm:"column:username"`
	Email    string `gorm:"column:email"`
}

func (testUser) TableName() string { return "users" }

// TestInterviewEmailFlow simulates the email consumer's handling of an
// interview_scheduled event for a real candidate. It reads the candidate's
// email from the database, renders the template, and sends a test email.
//
// Usage:
//
//	SEND_INTERVIEW_EMAIL_TO=候选人邮箱地址 \
//	go test ./email/ -run TestInterviewEmailFlow -v -count=1
func TestInterviewEmailFlow(t *testing.T) {
	to := os.Getenv("SEND_INTERVIEW_EMAIL_TO")
	if to == "" {
		t.Skip("set SEND_INTERVIEW_EMAIL_TO env var to enable this test")
	}

	// ── Load config ────────────────────────────────────────────────────
	raw, err := os.ReadFile("../config/config.example.yaml")
	if err != nil {
		t.Fatalf("read config.example.yaml: %v", err)
	}
	var cfg fullTestConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse config.example.yaml: %v", err)
	}

	// ── DB: lookup candidate ───────────────────────────────────────────
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect to mysql: %v", err)
	}

	var user testUser
	if err := db.Where("email = ?", to).First(&user).Error; err != nil {
		t.Fatalf("lookup user by email %q: %v", to, err)
	}
	t.Logf("found user: id=%d username=%q email=%q", user.ID, user.Username, user.Email)

	if user.Email == "" {
		t.Fatal("user has no email — email consumer would skip this user")
	}

	// ── Render template ────────────────────────────────────────────────
	baseURL := cfg.FrontendBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}
	r, err := NewRenderer(baseURL)
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}

	payload := struct {
		JobTitle      string
		Content       string
		Link          string
		RecipientName string
		InterviewDate string
		InterviewMode string
		InterviewLink string
		InterviewLoc  string
	}{
		JobTitle:      "测试岗位 — 高级工程师",
		Content:       "您好，您的「测试岗位 — 高级工程师」面试已安排好，请查看详情。",
		Link:          "/applications",
		RecipientName: user.Username,
		InterviewDate: time.Now().Add(48 * time.Hour).Format("2006年01月02日 15:04"),
		InterviewMode: "视频面试",
		InterviewLink: "https://meeting.example.com/test123",
		InterviewLoc:  "线上",
	}

	td := TemplateData{
		RecipientName: payload.RecipientName,
		JobTitle:      payload.JobTitle,
		Content:       payload.Content,
		ActionURL:     payload.Link,
		ActionText:    "查看面试详情",
		InterviewDate: payload.InterviewDate,
		InterviewMode: payload.InterviewMode,
		InterviewLink: payload.InterviewLink,
		InterviewLoc:  payload.InterviewLoc,
	}

	subject, htmlBody, err := r.Render("interview_scheduled", td)
	if err != nil {
		t.Fatalf("render template: %v", err)
	}
	t.Logf("rendered: subject=%q html_len=%d", subject, len(htmlBody))

	// ── Send email ─────────────────────────────────────────────────────
	sender := NewSMTPSender(SMTPConfig{
		Host:        cfg.SMTP.Host,
		Port:        cfg.SMTP.Port,
		Username:    cfg.SMTP.Username,
		Password:    cfg.SMTP.Password,
		FromAddress: cfg.SMTP.FromAddress,
		FromName:    cfg.SMTP.FromName,
	})
	defer sender.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Logf("sending interview_scheduled email to %s ...", user.Email)
	if err := sender.Send(ctx, user.Email, "[SMART RECRUIT TEST] "+subject, htmlBody); err != nil {
		t.Fatalf("send email failed: %v", err)
	}

	fmt.Println()
	fmt.Println("=== 面试通知邮件发送链路测试通过 ===")
	fmt.Printf("  候选人: %s (%s)\n", user.Username, user.Email)
	fmt.Printf("  岗位:   %s\n", payload.JobTitle)
	fmt.Printf("  面试时间: %s\n", payload.InterviewDate)
	fmt.Printf("  面试方式: %s\n", payload.InterviewMode)
	fmt.Printf("  邮件标题: %s\n", subject)
	fmt.Printf("  HTML 长度: %d bytes\n", len(htmlBody))
	fmt.Println()
}
