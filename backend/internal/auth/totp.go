package auth

import (
	"fmt"

	"github.com/pquerna/otp/totp"
)

const totpIssuer = "Candella"

// generateTOTPSecret creates a new TOTP secret for a user and returns both the
// base32 secret and the otpauth:// URL the frontend renders as a QR code.
func generateTOTPSecret(accountEmail string) (secret, otpauthURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: accountEmail,
	})
	if err != nil {
		return "", "", fmt.Errorf("totp: generate: %w", err)
	}
	return key.Secret(), key.URL(), nil
}

// validateTOTP reports whether a 6-digit code is currently valid for the secret.
func validateTOTP(code, secret string) bool {
	return totp.Validate(code, secret)
}
