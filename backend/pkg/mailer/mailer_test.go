package mailer

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewFailsClosedForMailConfiguration(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, tc := range []struct {
		kind, key, from string
		production      bool
	}{
		{kind: "console", production: true},
		{kind: "resend", from: "Candella <hello@example.com>"},
		{kind: "resend", key: "test-key", from: "not a sender"},
		{kind: "unknown"},
	} {
		if _, err := New(tc.kind, tc.key, tc.from, tc.production, log); err == nil {
			t.Fatalf("accepted bad mail configuration: %+v", tc)
		}
	}
	if _, err := New("console", "", "", false, log); err != nil {
		t.Fatalf("development console sender rejected: %v", err)
	}
	if _, err := New("resend", "test-key", "Candella <hello@example.com>", true, log); err != nil {
		t.Fatalf("valid Resend settings rejected: %v", err)
	}
}

func TestResendSendsBothHTMLAndPlainText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/emails" ||
			r.Header.Get("Authorization") != "Bearer test-key" ||
			r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected Resend request: %s %s", r.Method, r.URL.Path)
		}
		var payload struct {
			From, Subject, Text, HTML string
			To                        []string
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if payload.From != "Candella <hello@example.com>" ||
			len(payload.To) != 1 || payload.To[0] != "customer@example.com" ||
			payload.Subject == "" || !strings.Contains(payload.Text, "032157") ||
			!strings.Contains(payload.HTML, "code-tile") {
			t.Errorf("incomplete Resend payload: %+v", payload)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	message, err := VerificationEmail("customer@example.com", "Customer", "032157", 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	sender := NewResendMailer("test-key", "Candella <hello@example.com>")
	sender.endpoint = server.URL + "/emails"
	if err := sender.Send(context.Background(), message); err != nil {
		t.Fatal(err)
	}
}

func TestResendReportsProviderFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer server.Close()
	sender := NewResendMailer("test-key", "hello@example.com")
	sender.endpoint = server.URL
	if err := sender.Send(context.Background(), Email{To: "a@example.com", Subject: "A", Text: "B"}); err == nil ||
		!strings.Contains(err.Error(), "422") {
		t.Fatalf("expected provider status, got %v", err)
	}
}
