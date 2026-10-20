package auth_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pquerna/otp/totp"
	"github.com/pressly/goose/v3"

	"candella-ecosystem/backend/internal/audit"
	"candella-ecosystem/backend/internal/auth"
	"candella-ecosystem/backend/migrations"
	"candella-ecosystem/backend/pkg/config"
	"candella-ecosystem/backend/pkg/crypto"
	"candella-ecosystem/backend/pkg/database"
	"candella-ecosystem/backend/pkg/mailer"
)

const cookieName = "candella_session"

// TestAuthAccessBoundariesPostgres drives the real auth HTTP routes against a
// disposable database and proves each boundary fails closed: no session is
// issued for a wrong password, an unverified or disabled account, or a missing
// or wrong second factor, and a revoked or expired session stops working.
func TestAuthAccessBoundariesPostgres(t *testing.T) {
	env := newAuthEnv(t)
	password := "Correct-Horse-9!"

	t.Run("verified sign-up holds a working session", func(t *testing.T) {
		b := env.browser()
		env.signUp(b, password)
		if code, _ := b.do("GET", "/auth/me", nil); code != http.StatusOK {
			t.Fatalf("/me after verify = %d, want 200", code)
		}
	})

	t.Run("wrong password and unknown email look identical and issue no session", func(t *testing.T) {
		owner := env.browser()
		email := env.signUp(owner, password)

		b := env.browser()
		code, res := b.do("POST", "/auth/login", map[string]string{"email": email, "password": "Wrong-Pass-1!"})
		expectError(t, "wrong password", code, res, http.StatusUnauthorized, "invalid_credentials")
		code, res = b.do("POST", "/auth/login", map[string]string{"email": env.newEmail(), "password": password})
		expectError(t, "unknown email", code, res, http.StatusUnauthorized, "invalid_credentials")
		expectNoSession(t, b)
	})

	t.Run("unverified account cannot sign in", func(t *testing.T) {
		b := env.browser()
		email := env.newEmail()
		if code, res := b.do("POST", "/auth/register", map[string]string{"fullName": "Unverified", "email": email, "password": password}); code != http.StatusCreated {
			t.Fatalf("register = %d %+v", code, res.Error)
		}
		code, res := b.do("POST", "/auth/login", map[string]string{"email": email, "password": password})
		expectError(t, "unverified login", code, res, http.StatusForbidden, "email_not_verified")
		expectNoSession(t, b)
	})

	t.Run("logout revokes the session server-side", func(t *testing.T) {
		b := env.browser()
		env.signUp(b, password)
		stolen := b.sessionCookie()

		if code, _ := b.do("POST", "/auth/logout", nil); code != http.StatusOK {
			t.Fatalf("logout = %d", code)
		}
		replay := env.browserWithCookie(stolen)
		code, res := replay.do("GET", "/auth/me", nil)
		expectError(t, "replayed cookie after logout", code, res, http.StatusUnauthorized, "unauthenticated")
	})

	t.Run("expired session is refused", func(t *testing.T) {
		b := env.browser()
		env.signUp(b, password)
		env.exec(`UPDATE sessions SET expires_at = now() - interval '1 minute' WHERE token_hash = $1`,
			crypto.HashToken(b.sessionCookie()))
		code, res := b.do("GET", "/auth/me", nil)
		expectError(t, "expired session", code, res, http.StatusUnauthorized, "unauthenticated")
	})

	t.Run("2FA: the password alone never yields a session", func(t *testing.T) {
		b := env.browser()
		email := env.signUp(b, password)
		secret, backupCodes := env.enableTwoFactor(b)
		b.do("POST", "/auth/logout", nil)

		mfaToken := env.passwordStep(b, email, password)
		if code, _ := b.do("GET", "/auth/me", nil); code != http.StatusUnauthorized {
			t.Fatalf("/me between password and 2FA = %d, want 401", code)
		}

		code, res := b.do("POST", "/auth/login/verify-2fa", map[string]string{"mfaToken": mfaToken, "code": invalidTOTP(t, secret)})
		expectError(t, "wrong TOTP", code, res, http.StatusBadRequest, "code_invalid")
		expectNoSession(t, b)

		code, res = b.do("POST", "/auth/login/verify-2fa", map[string]string{"mfaToken": extendExpiry(t, mfaToken), "code": currentTOTP(t, secret)})
		expectError(t, "challenge with a forged expiry", code, res, http.StatusBadRequest, "token_invalid")
		expectNoSession(t, b)

		if code, res := b.do("POST", "/auth/login/verify-2fa", map[string]string{"mfaToken": mfaToken, "code": currentTOTP(t, secret)}); code != http.StatusOK {
			t.Fatalf("valid TOTP = %d %+v", code, res.Error)
		}
		if code, _ := b.do("GET", "/auth/me", nil); code != http.StatusOK {
			t.Fatalf("/me after 2FA = %d, want 200", code)
		}

		// A backup code works exactly once.
		b.do("POST", "/auth/logout", nil)
		mfaToken = env.passwordStep(b, email, password)
		if code, res := b.do("POST", "/auth/login/verify-backup", map[string]string{"mfaToken": mfaToken, "backupCode": backupCodes[0]}); code != http.StatusOK {
			t.Fatalf("first backup code use = %d %+v", code, res.Error)
		}
		b.do("POST", "/auth/logout", nil)
		mfaToken = env.passwordStep(b, email, password)
		code, res = b.do("POST", "/auth/login/verify-backup", map[string]string{"mfaToken": mfaToken, "backupCode": backupCodes[0]})
		expectError(t, "reused backup code", code, res, http.StatusBadRequest, "code_invalid")
		expectNoSession(t, b)
	})

	t.Run("disabled account fails closed everywhere", func(t *testing.T) {
		live := env.browser()
		email := env.signUp(live, password)
		env.exec(`UPDATE users SET status = 'disabled' WHERE email = $1`, email)

		code, res := live.do("GET", "/auth/me", nil)
		expectError(t, "live session of a disabled user", code, res, http.StatusUnauthorized, "unauthenticated")

		b := env.browser()
		code, res = b.do("POST", "/auth/login", map[string]string{"email": email, "password": password})
		expectError(t, "disabled login", code, res, http.StatusForbidden, "account_disabled")
		expectNoSession(t, b)

		// Without the password, a disabled account is indistinguishable from any other.
		code, res = b.do("POST", "/auth/login", map[string]string{"email": email, "password": "Wrong-Pass-1!"})
		expectError(t, "disabled + wrong password", code, res, http.StatusUnauthorized, "invalid_credentials")
	})

	t.Run("account disabled between the password and 2FA steps gets no session", func(t *testing.T) {
		b := env.browser()
		email := env.signUp(b, password)
		secret, _ := env.enableTwoFactor(b)
		b.do("POST", "/auth/logout", nil)

		mfaToken := env.passwordStep(b, email, password)
		env.exec(`UPDATE users SET status = 'disabled' WHERE email = $1`, email)

		code, res := b.do("POST", "/auth/login/verify-2fa", map[string]string{"mfaToken": mfaToken, "code": currentTOTP(t, secret)})
		expectError(t, "2FA for a since-disabled account", code, res, http.StatusForbidden, "account_disabled")
		expectNoSession(t, b)
	})
}

// ---- environment -----------------------------------------------------------

type authEnv struct {
	t      *testing.T
	ctx    context.Context
	pool   *pgxpool.Pool
	server *httptest.Server
	mail   *captureMailer
	run    string
	seq    int
}

func newAuthEnv(t *testing.T) *authEnv {
	t.Helper()
	dsn := disposableDSN(t, "AUTH_TEST_DATABASE_URL", "/candella_auth_test")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	migrate(ctx, t, dsn)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	// Seed as the login; run the code under test as the API role, so RLS and
	// grants apply exactly as they do in the server.
	appPool, err := database.Connect(ctx, dsn, database.AppRole)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(appPool.Close)

	cfg := config.Config{
		AppBaseURL:            "http://localhost:3000",
		SessionCookieName:     cookieName,
		SessionTTL:            time.Hour,
		SessionCookieSameSite: "lax",
		// Cheap hashing keeps the suite fast; strength is not under test here.
		Argon2MemoryKB: 1024,
		Argon2Time:     1,
		Argon2Threads:  1,
		EmailCodeTTL:   10 * time.Minute,
		ResetTokenTTL:  30 * time.Minute,
	}
	enc, err := crypto.NewEncryptor(base64.StdEncoding.EncodeToString(randomBytes(t, 32)))
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	mail := &captureMailer{}
	svc := auth.NewService(auth.NewPostgresRepository(appPool), mail, audit.NewRecorder(appPool, log), cfg, log, enc, randomBytes(t, 32))

	router := chi.NewRouter()
	router.Route("/api/v1", func(api chi.Router) {
		auth.Routes(api, auth.NewHandler(svc, cfg), func(next http.Handler) http.Handler { return next })
	})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return &authEnv{t: t, ctx: ctx, pool: pool, server: server, mail: mail,
		run: fmt.Sprintf("%d", time.Now().UnixNano())}
}

// disposableDSN refuses anything but a local throwaway database, so the suite
// can never run against a developer or production database by accident.
func disposableDSN(t *testing.T, envVar, localPath string) string {
	t.Helper()
	dsn := os.Getenv(envVar)
	ci := false
	if dsn == "" && os.Getenv("GITHUB_ACTIONS") == "true" && os.Getenv("GITHUB_REPOSITORY") == "hamid-rehman-X6/candella-retail-platform" {
		dsn, ci = os.Getenv("DATABASE_URL"), true
	}
	if dsn == "" {
		t.Skipf("set %s for a disposable %s database", envVar, strings.TrimPrefix(localPath, "/"))
	}
	u, err := url.Parse(dsn)
	wantPath := localPath
	if ci {
		wantPath = "/candella"
	}
	if err != nil || u.Path != wantPath || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") || (ci && (u.User == nil || u.User.Username() != "candella")) {
		t.Fatal("auth integration test requires a local disposable PostgreSQL database")
	}
	return dsn
}

func migrate(ctx context.Context, t *testing.T, dsn string) {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	// Packages run in parallel against one CI database; serialise migrations.
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(hashtext('candella_integration_migrations'))`); err != nil {
		t.Fatal(err)
	}
	defer conn.ExecContext(ctx, `SELECT pg_advisory_unlock(hashtext('candella_integration_migrations'))`)
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpContext(ctx, db, "."); err != nil {
		t.Fatalf("disposable DB migrations: %v", err)
	}
}

func (e *authEnv) exec(query string, args ...any) {
	e.t.Helper()
	tag, err := e.pool.Exec(e.ctx, query, args...)
	if err != nil {
		e.t.Fatal(err)
	}
	if tag.RowsAffected() == 0 {
		e.t.Fatalf("no rows affected by %q", query)
	}
}

func (e *authEnv) newEmail() string {
	e.seq++
	return fmt.Sprintf("auth-%s-%d@example.test", e.run, e.seq)
}

// signUp registers and verifies a fresh account in b, leaving b signed in.
func (e *authEnv) signUp(b *browser, password string) string {
	e.t.Helper()
	email := e.newEmail()
	if code, res := b.do("POST", "/auth/register", map[string]string{"fullName": "Test Owner", "email": email, "password": password}); code != http.StatusCreated {
		e.t.Fatalf("register = %d %+v", code, res.Error)
	}
	if code, res := b.do("POST", "/auth/verify-email", map[string]string{"email": email, "code": e.mail.codeFor(e.t, email)}); code != http.StatusOK {
		e.t.Fatalf("verify-email = %d %+v", code, res.Error)
	}
	if b.sessionCookie() == "" {
		e.t.Fatal("verify-email did not issue a session")
	}
	return email
}

// enableTwoFactor turns on TOTP for the signed-in user and returns the secret
// and the one-time backup codes.
func (e *authEnv) enableTwoFactor(b *browser) (string, []string) {
	e.t.Helper()
	code, res := b.do("POST", "/auth/2fa/setup", nil)
	if code != http.StatusOK {
		e.t.Fatalf("2fa/setup = %d %+v", code, res.Error)
	}
	var setup struct{ Secret string }
	mustDecode(e.t, res.Data, &setup)

	code, res = b.do("POST", "/auth/2fa/enable", map[string]string{"code": currentTOTP(e.t, setup.Secret)})
	if code != http.StatusOK {
		e.t.Fatalf("2fa/enable = %d %+v", code, res.Error)
	}
	var enabled struct{ BackupCodes []string }
	mustDecode(e.t, res.Data, &enabled)
	if len(enabled.BackupCodes) == 0 {
		e.t.Fatal("no backup codes returned")
	}
	return setup.Secret, enabled.BackupCodes
}

// passwordStep submits the password for a 2FA account and returns the challenge,
// asserting that no session cookie was issued yet.
func (e *authEnv) passwordStep(b *browser, email, password string) string {
	e.t.Helper()
	code, res := b.do("POST", "/auth/login", map[string]string{"email": email, "password": password})
	if code != http.StatusOK {
		e.t.Fatalf("login = %d %+v", code, res.Error)
	}
	var step struct {
		MFARequired bool   `json:"mfaRequired"`
		MFAToken    string `json:"mfaToken"`
	}
	mustDecode(e.t, res.Data, &step)
	if !step.MFARequired || step.MFAToken == "" {
		e.t.Fatalf("login for a 2FA account did not ask for the second factor: %+v", step)
	}
	expectNoSession(e.t, b)
	return step.MFAToken
}

// ---- browser ---------------------------------------------------------------

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// browser is an HTTP client with its own cookie jar, like one tab's session.
type browser struct {
	t      *testing.T
	base   *url.URL
	client *http.Client
	cookie string // sent explicitly instead of the jar, to replay a captured cookie
}

func (e *authEnv) browser() *browser {
	jar, err := cookiejar.New(nil)
	if err != nil {
		e.t.Fatal(err)
	}
	base, _ := url.Parse(e.server.URL)
	return &browser{t: e.t, base: base, client: &http.Client{Jar: jar}}
}

func (e *authEnv) browserWithCookie(value string) *browser {
	base, _ := url.Parse(e.server.URL)
	return &browser{t: e.t, base: base, client: &http.Client{}, cookie: value}
}

func (b *browser) do(method, path string, body any) (int, envelope) {
	b.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			b.t.Fatal(err)
		}
		reader = strings.NewReader(string(raw))
	}
	req, err := http.NewRequest(method, b.base.String()+"/api/v1"+path, reader)
	if err != nil {
		b.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if b.cookie != "" {
		req.AddCookie(&http.Cookie{Name: cookieName, Value: b.cookie})
	}
	resp, err := b.client.Do(req)
	if err != nil {
		b.t.Fatal(err)
	}
	defer resp.Body.Close()
	var env envelope
	_ = json.NewDecoder(resp.Body).Decode(&env)
	return resp.StatusCode, env
}

func (b *browser) sessionCookie() string {
	if b.client.Jar == nil {
		return b.cookie
	}
	for _, c := range b.client.Jar.Cookies(b.base) {
		if c.Name == cookieName {
			return c.Value
		}
	}
	return ""
}

// ---- assertions & helpers --------------------------------------------------

func expectError(t *testing.T, what string, code int, res envelope, wantStatus int, wantCode string) {
	t.Helper()
	got := ""
	if res.Error != nil {
		got = res.Error.Code
	}
	if code != wantStatus || got != wantCode {
		t.Fatalf("%s = %d %q, want %d %q", what, code, got, wantStatus, wantCode)
	}
}

func expectNoSession(t *testing.T, b *browser) {
	t.Helper()
	if v := b.sessionCookie(); v != "" {
		t.Fatalf("a session cookie was issued when none should be")
	}
	if code, _ := b.do("GET", "/auth/me", nil); code != http.StatusUnauthorized {
		t.Fatalf("/me = %d, want 401", code)
	}
}

func mustDecode(t *testing.T, raw json.RawMessage, dst any) {
	t.Helper()
	if err := json.Unmarshal(raw, dst); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
}

func currentTOTP(t *testing.T, secret string) string {
	t.Helper()
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return code
}

// invalidTOTP returns a code outside the validator's ±1 step window, so it is
// wrong with certainty rather than with high probability.
func invalidTOTP(t *testing.T, secret string) string {
	t.Helper()
	valid := map[string]bool{}
	for _, offset := range []time.Duration{-30 * time.Second, 0, 30 * time.Second} {
		code, err := totp.GenerateCode(secret, time.Now().Add(offset))
		if err != nil {
			t.Fatal(err)
		}
		valid[code] = true
	}
	for n := 0; ; n++ {
		if candidate := fmt.Sprintf("%06d", n); !valid[candidate] {
			return candidate
		}
	}
}

// extendExpiry forges a later expiry into a subject.purpose.expiry.signature
// challenge, as an attacker trying to keep a stolen challenge alive would.
func extendExpiry(t *testing.T, token string) string {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 4 {
		t.Fatalf("unexpected challenge format: %d parts", len(parts))
	}
	parts[2] += "0"
	return strings.Join(parts, ".")
}

func randomBytes(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return b
}

// captureMailer records outgoing email so the test can read verification codes.
type captureMailer struct {
	mu   sync.Mutex
	sent []mailer.Email
}

func (m *captureMailer) Send(_ context.Context, e mailer.Email) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, e)
	return nil
}

var sixDigits = regexp.MustCompile(`\b(\d{6})\b`)

func (m *captureMailer) codeFor(t *testing.T, to string) string {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.sent) - 1; i >= 0; i-- {
		if m.sent[i].To == to {
			if match := sixDigits.FindStringSubmatch(m.sent[i].Text); match != nil {
				if !strings.Contains(m.sent[i].HTML, match[1]) {
					t.Fatal("verification email HTML does not contain its plain-text code")
				}
				return match[1]
			}
		}
	}
	t.Fatalf("no verification code sent to %s", to)
	return ""
}
