// Package id generates prefixed, URL-safe, collision-resistant identifiers in the
// Stripe style documented in docs/database/00_schema_design.md (§1.2), e.g.
// "usr_4f8K2mQpXrTz". A short type prefix makes IDs self-describing in logs and
// APIs; the 12-char base62 suffix gives ~62^12 (~3.2e21) values per prefix.
package id

import gonanoid "github.com/matoous/go-nanoid/v2"

// alphabet is base62 (no "-"/"_"), so the single underscore after the prefix is an
// unambiguous separator when parsing an ID.
const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

const suffixLength = 12

// Prefix is the per-table identifier prefix. Keep these in sync with the registry
// in docs/database/00_schema_design.md (§1.2).
type Prefix string

const (
	User             Prefix = "usr"
	Session          Prefix = "sess"
	OAuthAccount     Prefix = "oauth"
	VerificationCode Prefix = "vcode"
	PasswordResetTok Prefix = "prt"
	MFABackupCode    Prefix = "bkp"
	AuditLog         Prefix = "alog"
)

// New returns a new prefixed ID, e.g. New(id.User) -> "usr_4f8K2mQpXrTz".
// It panics only if the system CSPRNG is unavailable, which is unrecoverable.
func New(prefix Prefix) string {
	suffix, err := gonanoid.Generate(alphabet, suffixLength)
	if err != nil {
		// crypto/rand failing is a fatal, unrecoverable condition.
		panic("id: failed to generate identifier: " + err.Error())
	}
	return string(prefix) + "_" + suffix
}
