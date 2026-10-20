package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"
)

// The HTML shell is shared by every transactional message. Inline styles keep
// the important content readable in clients that strip <style> or animations.
var messageHTML = template.Must(template.New("candella-email").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="color-scheme" content="light">
  <title>{{.Subject}}</title>
  <style>
    @keyframes code-arrive { from { transform: translateY(5px); opacity: .55; } to { transform: translateY(0); opacity: 1; } }
    .code-tile { animation: code-arrive .45s ease-out both; }
    @media (prefers-reduced-motion: reduce) { .code-tile { animation: none !important; } }
    @media screen and (max-width: 620px) { .email-pad { padding: 28px 22px !important; } }
  </style>
</head>
<body style="margin:0;padding:0;background:#f6f4ef;color:#1d1b18;font-family:Arial,Helvetica,sans-serif;">
  <span style="display:none;max-height:0;overflow:hidden;opacity:0;color:transparent;">{{.Preheader}}</span>
  <table role="presentation" cellpadding="0" cellspacing="0" width="100%" style="border-collapse:collapse;background:#f6f4ef;">
    <tr><td align="center" style="padding:36px 12px;">
      <table role="presentation" cellpadding="0" cellspacing="0" width="100%" style="max-width:560px;border-collapse:collapse;">
        <tr><td style="padding:0 6px 18px;">
          <span style="display:inline-block;width:34px;height:34px;border-radius:11px;background:#f5a800;color:#241600;font-size:20px;font-weight:800;line-height:34px;text-align:center;vertical-align:middle;">C</span>
          <span style="padding-left:9px;color:#171511;font-size:19px;font-weight:700;letter-spacing:-.4px;vertical-align:middle;">Candella</span>
        </td></tr>
        <tr><td class="email-pad" style="border:1px solid #e8e1d3;border-top:4px solid #f5a800;border-radius:16px;background:#ffffff;padding:40px 42px;">
          <p style="margin:0 0 13px;color:#9b5b00;font-size:11px;font-weight:700;letter-spacing:2px;text-transform:uppercase;">{{.Eyebrow}}</p>
          <h1 style="margin:0 0 20px;color:#171511;font-size:27px;line-height:1.2;letter-spacing:-.7px;">{{.Title}}</h1>
          <p style="margin:0 0 12px;color:#45413b;font-size:15px;line-height:1.65;">Hi {{.Name}},</p>
          <p style="margin:0 0 24px;color:#45413b;font-size:15px;line-height:1.65;">{{.Intro}}</p>
          {{if .Code}}
          <table role="presentation" cellpadding="0" cellspacing="0" width="100%" style="border-collapse:collapse;border-radius:12px;background:#f2f5f9;">
            <tr><td style="padding:22px 18px;">
              <p style="margin:0 0 14px;color:#4a5d71;font-size:12px;font-weight:700;letter-spacing:1px;text-transform:uppercase;">Code requested</p>
              <table role="presentation" cellpadding="0" cellspacing="0" aria-label="Verification code {{.Code}}" style="border-collapse:collapse;">
                <tr>{{range .Digits}}<td style="padding:0 6px 0 0;"><span class="code-tile" style="display:inline-block;width:39px;height:47px;border:1px solid #e1e7ee;border-radius:6px;background:#ffffff;color:#172230;font-size:25px;font-weight:600;line-height:47px;text-align:center;animation-delay:{{.Delay}};">{{.Value}}</span></td>{{end}}</tr>
              </table>
              <p style="margin:14px 0 0;color:#4a5d71;font-size:12px;line-height:1.5;">To copy it easily, select this code: <strong style="font-family:Consolas,Monaco,monospace;letter-spacing:2px;color:#172230;">{{.Code}}</strong></p>
            </td></tr>
          </table>
          {{end}}
          {{if .ActionURL}}
          <table role="presentation" cellpadding="0" cellspacing="0" style="border-collapse:collapse;"><tr><td style="border-radius:9px;background:#f5a800;">
            <a href="{{.ActionURL}}" style="display:inline-block;padding:14px 22px;color:#241600;font-size:14px;font-weight:700;text-decoration:none;">{{.ActionLabel}} &rarr;</a>
          </td></tr></table>
          <p style="margin:17px 0 0;color:#706a62;font-size:12px;line-height:1.6;">If the button does not work, copy this link into your browser:<br><a href="{{.ActionURL}}" style="color:#865200;word-break:break-all;">{{.ActionURL}}</a></p>
          {{end}}
          <p style="margin:25px 0 0;color:#45413b;font-size:13px;line-height:1.6;">{{.Expiry}}</p>
          <p style="margin:12px 0 0;color:#706a62;font-size:13px;line-height:1.6;">{{.Safety}}</p>
        </td></tr>
        <tr><td style="padding:20px 7px;color:#827b70;font-size:12px;line-height:1.6;">
          Sent by Candella. This is an automated account message; please do not reply.
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`))

type codeDigit struct {
	Value string
	Delay string
}

type messageData struct {
	Subject, Preheader, Eyebrow, Title, Name, Intro string
	Code, ActionURL, ActionLabel, Expiry, Safety    string
	Digits                                          []codeDigit
}

func VerificationEmail(to, name, code string, ttl time.Duration) (Email, error) {
	if ttl <= 0 {
		return Email{}, fmt.Errorf("mailer: verification code lifetime must be positive")
	}
	if len(code) != 6 {
		return Email{}, fmt.Errorf("mailer: verification code must have six digits")
	}
	for _, digit := range code {
		if digit < '0' || digit > '9' {
			return Email{}, fmt.Errorf("mailer: verification code must be numeric")
		}
	}
	digits := make([]codeDigit, 0, 6)
	for i, digit := range code {
		digits = append(digits, codeDigit{Value: string(digit), Delay: fmt.Sprintf("%dms", i*55)})
	}
	return renderMessage(to, messageData{
		Subject:   "Your Candella verification code",
		Preheader: "Use your six-digit code to verify your Candella account.",
		Eyebrow:   "Account security",
		Title:     "Verify your email",
		Name:      displayName(name),
		Intro:     "Enter this one-time code in Candella to finish setting up your account.",
		Code:      code,
		Digits:    digits,
		Expiry:    "This code expires in " + durationLabel(ttl) + ".",
		Safety:    "If you did not create a Candella account, you can ignore this email. Never share this code.",
	})
}

func PasswordResetEmail(to, name, resetURL string, ttl time.Duration) (Email, error) {
	if ttl <= 0 {
		return Email{}, fmt.Errorf("mailer: password reset lifetime must be positive")
	}
	if err := validateActionURL(resetURL); err != nil {
		return Email{}, err
	}
	return renderMessage(to, messageData{
		Subject:     "Reset your Candella password",
		Preheader:   "A secure link to reset your Candella password.",
		Eyebrow:     "Account security",
		Title:       "Reset your password",
		Name:        displayName(name),
		Intro:       "We received a request to reset your password. Use the button below to choose a new one.",
		ActionURL:   resetURL,
		ActionLabel: "Reset password",
		Expiry:      "This link expires in " + durationLabel(ttl) + ".",
		Safety:      "If you did not request this change, you can ignore this email. Your password will stay the same.",
	})
}

func StaffInvitationEmail(to, role, inviteURL string, ttl time.Duration) (Email, error) {
	if ttl <= 0 {
		return Email{}, fmt.Errorf("mailer: staff invitation lifetime must be positive")
	}
	if err := validateActionURL(inviteURL); err != nil {
		return Email{}, err
	}
	return renderMessage(to, messageData{
		Subject:     "Join your Candella workspace",
		Preheader:   "You have been invited to a Candella workspace.",
		Eyebrow:     "Workspace invitation",
		Title:       "You are invited",
		Name:        "there",
		Intro:       "You have been invited to join a Candella workspace as " + strings.TrimSpace(role) + ". Sign in with the email address that received this invitation to accept.",
		ActionURL:   inviteURL,
		ActionLabel: "Accept invitation",
		Expiry:      "This invitation expires in " + durationLabel(ttl) + ".",
		Safety:      "If you were not expecting this invitation, you can ignore this email.",
	})
}

func renderMessage(to string, data messageData) (Email, error) {
	var html bytes.Buffer
	if err := messageHTML.Execute(&html, data); err != nil {
		return Email{}, fmt.Errorf("mailer: render HTML: %w", err)
	}
	var plain strings.Builder
	fmt.Fprintf(&plain, "CANDELLA\n\n%s\n\nHi %s,\n\n%s\n\n", data.Title, data.Name, data.Intro)
	if data.Code != "" {
		fmt.Fprintf(&plain, "Your verification code: %s\n\n", data.Code)
	}
	if data.ActionURL != "" {
		fmt.Fprintf(&plain, "%s: %s\n\n", data.ActionLabel, data.ActionURL)
	}
	fmt.Fprintf(&plain, "%s\n\n%s\n\nSent by Candella. This is an automated account message; please do not reply.\n", data.Expiry, data.Safety)
	return Email{To: to, Subject: data.Subject, Text: plain.String(), HTML: html.String()}, nil
}

func displayName(name string) string {
	name = strings.TrimSpace(strings.NewReplacer("\r", " ", "\n", " ").Replace(name))
	if name == "" {
		return "there"
	}
	return name
}

func durationLabel(ttl time.Duration) string {
	if ttl >= 24*time.Hour {
		days := int((ttl + 24*time.Hour - 1) / (24 * time.Hour))
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
	minutes := int((ttl + time.Minute - 1) / time.Minute)
	if minutes <= 1 {
		return "1 minute"
	}
	return fmt.Sprintf("%d minutes", minutes)
}

func validateActionURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return fmt.Errorf("mailer: action URL must be an absolute HTTP(S) URL")
	}
	return nil
}

// ValidateBaseURL prevents production emails from pointing at a local or
// non-HTTPS frontend. Development may use localhost over HTTP.
func ValidateBaseURL(raw string, production bool) error {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil ||
		(u.Scheme != "http" && u.Scheme != "https") ||
		(u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("mailer: APP_BASE_URL must be a frontend HTTP(S) origin")
	}
	if production && (u.Scheme != "https" || strings.EqualFold(u.Hostname(), "localhost") || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1") {
		return fmt.Errorf("mailer: production APP_BASE_URL must be a public HTTPS origin")
	}
	return nil
}
