package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// numericCode returns a cryptographically-random decimal string of n digits
// (zero-padded), e.g. "004215". Used for the 6-digit email codes.
func numericCode(n int) (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	num, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("tokens: numeric code: %w", err)
	}
	return fmt.Sprintf("%0*d", n, num), nil
}

// randomToken returns a URL-safe, base64-encoded random token of nBytes entropy.
// Used for session tokens and password-reset tokens.
func randomToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("tokens: random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// backupCode returns a human-friendly single-use code like "4F9K-2QMX".
func backupCode() (string, error) {
	const alphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ" // no ambiguous 0/O/1/I
	buf := make([]byte, 8)
	for i := range buf {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", fmt.Errorf("tokens: backup code: %w", err)
		}
		buf[i] = alphabet[idx.Int64()]
	}
	return string(buf[:4]) + "-" + string(buf[4:]), nil
}

// ---------------------------------------------------------------------------
// Stateless signed challenges (HMAC). Used for the short-lived "mfaToken" issued
// between the password step and the 2FA step of login, and for the OAuth "state"
// parameter. No database row needed — the signature + expiry are self-contained.
// Format: base64url(subject) "." purpose "." expUnix "." base64url(hmac).
// ---------------------------------------------------------------------------

func signChallenge(key []byte, subject, purpose string, ttl time.Duration) string {
	exp := strconv.FormatInt(time.Now().Add(ttl).Unix(), 10)
	payload := base64.RawURLEncoding.EncodeToString([]byte(subject)) + "." + purpose + "." + exp
	return payload + "." + sign(key, payload)
}

func parseChallenge(key []byte, token, purpose string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 4 {
		return "", errors.New("tokens: malformed challenge")
	}
	payload := parts[0] + "." + parts[1] + "." + parts[2]
	if subtle.ConstantTimeCompare([]byte(parts[3]), []byte(sign(key, payload))) != 1 {
		return "", errors.New("tokens: bad signature")
	}
	if parts[1] != purpose {
		return "", errors.New("tokens: wrong purpose")
	}
	exp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return "", errors.New("tokens: expired")
	}
	subject, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", errors.New("tokens: bad subject")
	}
	return string(subject), nil
}

func sign(key []byte, payload string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
