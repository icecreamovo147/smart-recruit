package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"go.uber.org/zap"

	"smart-recruit-platform-go/logger"
)

// Sender is the interface for sending emails.
type Sender interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
	Close() error
}

// SMTPConfig holds SMTP server configuration.
type SMTPConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	FromAddress string
	FromName    string
	TLS         bool
	Required    bool
}

// SMTPSender sends emails via SMTP with STARTTLS support.
type SMTPSender struct {
	cfg SMTPConfig
}

// NewSMTPSender creates a new SMTP sender.
func NewSMTPSender(cfg SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) Send(_ context.Context, to, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	msg := buildMIMEMessage(s.cfg.FromAddress, s.cfg.FromName, to, subject, htmlBody)

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	// Port 465 uses implicit TLS (direct TLS handshake).
	// Other ports (e.g. 587) use smtp.SendMail, which auto-negotiates STARTTLS.
	if s.cfg.Port == 465 {
		return sendWithTLS(addr, s.cfg.Host, auth, s.cfg.FromAddress, to, msg)
	}
	return smtp.SendMail(addr, auth, s.cfg.FromAddress, []string{to}, msg)
}

func (s *SMTPSender) Close() error {
	return nil
}

func sendWithTLS(addr, host string, auth smtp.Auth, from, to string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Quit()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}
	return nil
}

// LogSender logs emails instead of sending them (for local development).
type LogSender struct{}

// NewLogSender creates a sender that logs email content.
func NewLogSender() *LogSender {
	return &LogSender{}
}

func (s *LogSender) Send(_ context.Context, to, subject, htmlBody string) error {
	logger.L().Info("email (log mode)",
		zap.String("to", MaskEmail(to)),
		zap.String("subject", subject),
		zap.Int("html_len", len(htmlBody)),
	)
	return nil
}

func (s *LogSender) Close() error {
	return nil
}

func buildMIMEMessage(from, fromName, to, subject, htmlBody string) []byte {
	var b strings.Builder
	if fromName != "" {
		fmt.Fprintf(&b, "From: %s <%s>\r\n", fromName, from)
	} else {
		fmt.Fprintf(&b, "From: %s\r\n", from)
	}
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	b.WriteString("\r\n")
	b.WriteString(htmlBody)
	return []byte(b.String())
}

// NewSender creates the appropriate Sender based on configuration.
// Returns LogSender when SMTP host is empty (development mode).
func NewSender(cfg SMTPConfig) (Sender, error) {
	if cfg.Host == "" {
		logger.L().Info("SMTP not configured, using log-only email sender")
		return NewLogSender(), nil
	}
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	if cfg.FromAddress == "" {
		cfg.FromAddress = "noreply@smart-recruit.local"
	}
	if cfg.FromName == "" {
		cfg.FromName = "Smart Recruit"
	}
	// Validate host:port is reachable (best-effort).
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port), 5e9)
	if err != nil {
		if cfg.Required {
			return nil, fmt.Errorf("SMTP host %s:%d unreachable: %w", cfg.Host, cfg.Port, err)
		}
		logger.L().Warn("SMTP host unreachable, falling back to log sender",
			zap.String("host", cfg.Host), zap.Int("port", cfg.Port), zap.Error(err))
		return NewLogSender(), nil
	}
	conn.Close()
	logger.L().Info("SMTP sender initialized",
		zap.String("host", cfg.Host), zap.Int("port", cfg.Port))
	return NewSMTPSender(cfg), nil
}

// MaskEmail masks the local part of an email address for PII protection.
// Returns "***" if the address cannot be parsed.
func MaskEmail(addr string) string {
	if addr == "" {
		return ""
	}
	idx := strings.LastIndex(addr, "@")
	if idx < 0 {
		return "***"
	}
	if idx <= 1 {
		return "***" + addr[idx:]
	}
	return addr[:1] + "***" + addr[idx:]
}
