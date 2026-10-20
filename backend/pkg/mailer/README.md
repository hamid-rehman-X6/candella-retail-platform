# Transactional email

Account verification, password reset, and staff invitation use one HTML shell and a matching plain-text message. The code cards and buttons work without external images or fonts. Verification digits are shown in six boxes, with a continuous code that can be selected and copied. Some email clients do not support CSS animation; the code remains visible and usable when motion is stripped. Email clients do not offer reliable JavaScript clipboard access, so the email does not show a fake Copy button.

## Local preview without sending

From `backend/`:

```powershell
New-Item -ItemType Directory -Force tmp | Out-Null
go run ./cmd/email-preview verification > tmp/verification-preview.html
go run ./cmd/email-preview reset > tmp/reset-preview.html
go run ./cmd/email-preview invitation > tmp/invitation-preview.html
```

Open the generated HTML files in a browser. `backend/tmp/` is Git-ignored. The preview uses example addresses and tokens; it does not call Resend or the database. For a real inbox review, use a test account and the application's normal signup, reset, and invitation flows.

## Resend setup

1. Verify a sending domain in [Resend](https://resend.com/docs/dashboard/domains/introduction) and create a sending API key.
2. Set these values in the private backend environment (for local use, `backend/.env`). Never put the key in Git, an issue, or a PR:

```dotenv
MAILER=resend
RESEND_API_KEY=<private sending key>
EMAIL_FROM=Candella <hello@your-verified-domain.example>
APP_BASE_URL=https://your-public-frontend.example
```

The sender address must use the verified domain. Password-reset and staff-invitation links use `APP_BASE_URL`; use the public HTTPS frontend origin in production. The configured code/reset TTLs appear in both HTML and text.

With `MAILER=console`, local development logs the plain-text email instead of sending it. Production refuses console mode. Selecting `MAILER=resend` without an API key or valid `EMAIL_FROM` fails at startup. Resend API errors are reported as delivery failures; the sender does not silently switch to console.

No database migration is required. The application uses the existing [Resend Send Email API](https://resend.com/docs/api-reference/emails/send-email) with HTML and text bodies. Check a test inbox in the target email clients before live rollout.
