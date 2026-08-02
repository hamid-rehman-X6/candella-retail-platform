// Package mailer sends transactional emails behind a small interface, so the auth
// service depends on the behavior — not on a concrete provider. Two implementations
// ship today: ConsoleMailer (dev: logs the message so codes/links are visible in the
// server output) and ResendMailer (prod: posts to the Resend HTTP API).
package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Email is a single outbound message. Text is required; HTML is optional.
type Email struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

// Mailer delivers an Email. Implementations must be safe for concurrent use.
type Mailer interface {
	Send(ctx context.Context, email Email) error
}

// ---------------------------------------------------------------------------
// ConsoleMailer — development. Writes the email to the logger instead of sending.
// ---------------------------------------------------------------------------

type ConsoleMailer struct{ log *slog.Logger }

// NewConsoleMailer returns a Mailer that logs messages (so you can read OTP codes
// and reset links straight from the server output during local development).
func NewConsoleMailer(log *slog.Logger) *ConsoleMailer { return &ConsoleMailer{log: log} }

func (m *ConsoleMailer) Send(_ context.Context, email Email) error {
	m.log.Info("email (console mailer — not actually sent)",
		"to", email.To,
		"subject", email.Subject,
		"body", email.Text,
	)
	return nil
}

// ---------------------------------------------------------------------------
// ResendMailer — production. Posts to https://api.resend.com/emails.
// ---------------------------------------------------------------------------

type ResendMailer struct {
	apiKey string
	from   string
	client *http.Client
}

// NewResendMailer returns a Mailer backed by the Resend API.
func NewResendMailer(apiKey, from string) *ResendMailer {
	return &ResendMailer{
		apiKey: apiKey,
		from:   from,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (m *ResendMailer) Send(ctx context.Context, email Email) error {
	payload := map[string]any{
		"from":    m.from,
		"to":      []string{email.To},
		"subject": email.Subject,
		"text":    email.Text,
	}
	if email.HTML != "" {
		payload["html"] = email.HTML
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mailer: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("mailer: request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("mailer: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("mailer: resend returned status %d", resp.StatusCode)
	}
	return nil
}

// New selects the Mailer implementation from configuration ("console" | "resend").
func New(kind, resendAPIKey, from string, log *slog.Logger) Mailer {
	if kind == "resend" && resendAPIKey != "" {
		return NewResendMailer(resendAPIKey, from)
	}
	return NewConsoleMailer(log)
}
