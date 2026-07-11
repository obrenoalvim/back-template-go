package mail

import (
	"fmt"
	"log/slog"
	"net/smtp"

	"github.com/obrenoalvim/back-template-go/internal/config"
)

type Mailer struct {
	cfg config.Config
}

func New(cfg config.Config) *Mailer {
	return &Mailer{cfg: cfg}
}

func (m *Mailer) Send(to, subject, body string) {
	if m.cfg.MailHost == "" {
		slog.Info("mail.console_fallback", "to", to, "subject", subject, "body", body)
		return
	}

	addr := fmt.Sprintf("%s:%d", m.cfg.MailHost, m.cfg.MailPort)
	var auth smtp.Auth
	if m.cfg.MailUsername != "" {
		auth = smtp.PlainAuth("", m.cfg.MailUsername, m.cfg.MailPassword, m.cfg.MailHost)
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n", m.cfg.MailFrom, to, subject, body)
	if err := smtp.SendMail(addr, auth, m.cfg.MailFrom, []string{to}, []byte(msg)); err != nil {
		slog.Error("mail.send_failed", "to", to, "error", err)
	}
}
