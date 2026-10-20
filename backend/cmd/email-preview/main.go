// email-preview renders sample transactional messages without a database,
// API key, network call, or real verification token.
package main

import (
	"fmt"
	"os"
	"time"

	"candella-ecosystem/backend/pkg/mailer"
)

func main() {
	if len(os.Args) != 2 {
		usage()
	}
	var (
		message mailer.Email
		err     error
	)
	switch os.Args[1] {
	case "verification":
		message, err = mailer.VerificationEmail("preview@example.com", "Alex", "032157", 10*time.Minute)
	case "reset":
		message, err = mailer.PasswordResetEmail("preview@example.com", "Alex", "https://example.com/reset-password?token=preview-only", 30*time.Minute)
	case "invitation":
		message, err = mailer.StaffInvitationEmail("preview@example.com", "member", "https://example.com/invitations/accept?token=preview-only", 7*24*time.Hour)
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(message.HTML)
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: go run ./cmd/email-preview <verification|reset|invitation>")
	os.Exit(2)
}
