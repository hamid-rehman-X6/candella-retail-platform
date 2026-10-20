package mailer

import (
	"html"
	"strings"
	"testing"
	"time"
)

func TestVerificationEmailHasReadableCodeAndEscapedName(t *testing.T) {
	message, err := VerificationEmail("owner@example.com", "<script>alert(1)</script>", "032157", 12*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if message.To != "owner@example.com" || message.HTML == "" || message.Text == "" {
		t.Fatalf("message is incomplete: %+v", message)
	}
	if strings.Contains(message.HTML, "<script>alert(1)</script>") ||
		!strings.Contains(message.HTML, html.EscapeString("<script>alert(1)</script>")) {
		t.Fatal("untrusted name was not escaped in HTML")
	}
	for _, digit := range []string{">0</span>", ">3</span>", ">2</span>", ">1</span>", ">5</span>", ">7</span>"} {
		if !strings.Contains(message.HTML, digit) {
			t.Fatalf("missing code tile %q", digit)
		}
	}
	if !strings.Contains(message.HTML, "032157</strong>") ||
		!strings.Contains(message.Text, "032157") ||
		!strings.Contains(message.Text, "12 minutes") ||
		!strings.Contains(message.HTML, "prefers-reduced-motion") {
		t.Fatal("verification code, expiry, or reduced-motion fallback is missing")
	}
	if strings.Contains(message.HTML, "Copy code</a>") {
		t.Fatal("email must not pretend it can copy to the clipboard")
	}
}

func TestActionEmailsEscapeContentAndKeepSafeLinks(t *testing.T) {
	resetURL := "https://candella.example/reset-password?token=abc&next=home"
	reset, err := PasswordResetEmail("owner@example.com", "A&B", resetURL, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reset.HTML, "Hi A&amp;B,") ||
		!strings.Contains(reset.HTML, "token=abc&amp;next=home") ||
		!strings.Contains(reset.Text, resetURL) ||
		!strings.Contains(reset.Text, "30 minutes") {
		t.Fatal("reset email lost escaping, link, or expiry")
	}

	invite, err := StaffInvitationEmail(
		"staff@example.com", "<owner>", "https://candella.example/invitations/accept?token=demo", 7*24*time.Hour-time.Second,
	)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(invite.HTML, "<owner>") ||
		!strings.Contains(invite.HTML, "&lt;owner&gt;") ||
		!strings.Contains(invite.Text, "7 days") {
		t.Fatal("invitation role escaping or expiry is wrong")
	}
}

func TestTemplatesRejectUnsafeActionsAndCodes(t *testing.T) {
	if _, err := VerificationEmail("a@example.com", "A", "123456", 0); err == nil {
		t.Fatal("accepted a nonpositive verification lifetime")
	}
	for _, code := range []string{"12345", "1234567", "12a456"} {
		if _, err := VerificationEmail("a@example.com", "A", code, time.Minute); err == nil {
			t.Fatalf("accepted bad code %q", code)
		}
	}
	for _, raw := range []string{"javascript:alert(1)", "/relative", "https://user:pass@example.com/path"} {
		if _, err := PasswordResetEmail("a@example.com", "A", raw, time.Minute); err == nil {
			t.Fatalf("accepted unsafe URL %q", raw)
		}
	}
}

func TestValidateBaseURL(t *testing.T) {
	if err := ValidateBaseURL("http://localhost:3000", false); err != nil {
		t.Fatalf("local development URL rejected: %v", err)
	}
	if err := ValidateBaseURL("https://app.example.com", true); err != nil {
		t.Fatalf("public production URL rejected: %v", err)
	}
	for _, raw := range []string{"http://app.example.com", "https://localhost:3000", "https://app.example.com/path", "javascript:alert(1)"} {
		if err := ValidateBaseURL(raw, true); err == nil {
			t.Fatalf("unsafe production frontend URL accepted: %q", raw)
		}
	}
}
