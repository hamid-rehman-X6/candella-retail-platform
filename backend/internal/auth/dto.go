package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"candella-ecosystem/backend/pkg/validate"
)

// decodeJSON reads and unmarshals a JSON request body into dst. It returns the
// domain ErrInvalidInput on any decoding problem so handlers stay uniform.
func decodeJSON(r *http.Request, dst any) error {
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(dst); err != nil {
		return ErrInvalidInput
	}
	return nil
}

// ---- Requests -------------------------------------------------------------

type registerRequest struct {
	FullName string `json:"fullName"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r registerRequest) validate() error {
	if strings.TrimSpace(r.FullName) == "" {
		return ErrInvalidInput
	}
	if !validate.Email(r.Email) {
		return ErrInvalidEmail
	}
	if !validate.PasswordStrong(r.Password) {
		return ErrWeakPassword
	}
	return nil
}

type emailCodeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type emailOnlyRequest struct {
	Email string `json:"email"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type mfaCodeRequest struct {
	MFAToken string `json:"mfaToken"`
	Code     string `json:"code"`
}

type mfaBackupRequest struct {
	MFAToken   string `json:"mfaToken"`
	BackupCode string `json:"backupCode"`
}

type codeRequest struct {
	Code string `json:"code"`
}

type passwordRequest struct {
	Password string `json:"password"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// ---- Responses ------------------------------------------------------------

type userResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	FullName      string `json:"fullName"`
	EmailVerified bool   `json:"emailVerified"`
	MFAEnabled    bool   `json:"mfaEnabled"`
}

func toUserResponse(u *User) userResponse {
	return userResponse{
		ID:            u.ID,
		Email:         u.Email,
		FullName:      u.FullName,
		EmailVerified: u.IsEmailVerified(),
		MFAEnabled:    u.MFAEnabled,
	}
}

// loginResult is returned by the service's Login: either a session was created,
// or 2FA is required and the caller must complete the second step with mfaToken.
type loginResult struct {
	MFARequired bool
	MFAToken    string // set only when MFARequired
	RawToken    string // session token to set as a cookie, when not MFARequired
	Expires     time.Time
	User        *User
}
