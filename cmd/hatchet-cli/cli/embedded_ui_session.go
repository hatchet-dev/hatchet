package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// The embedded engine always seeds an admin user (adminseed.SeedDatabase runs
// on every embedded startup) with these defaults from the server config
// (pkg/config/database.SeedConfigFile). WithAdminUser / ADMIN_EMAIL and
// ADMIN_PASSWORD on the embedded process override them, in which case the same
// variables can be set for 'hatchet embedded-ui'.
const (
	defaultSeedAdminEmail    = "admin@example.com"
	defaultSeedAdminPassword = "Admin123!!"
)

// adminCredentials is the seeded admin identity the proxy signs in with.
type adminCredentials struct {
	email     string
	password  string
	isDefault bool
}

// resolveAdminCredentials returns the credentials for the transparent
// dashboard sign-in: the embedded seed defaults, unless ADMIN_EMAIL /
// ADMIN_PASSWORD are set (matching the variables the embedded engine itself
// reads when seeding).
func resolveAdminCredentials() adminCredentials {
	email := os.Getenv("ADMIN_EMAIL")
	password := os.Getenv("ADMIN_PASSWORD")

	if email == "" && password == "" {
		return adminCredentials{email: defaultSeedAdminEmail, password: defaultSeedAdminPassword, isDefault: true}
	}

	if email == "" {
		email = defaultSeedAdminEmail
	}
	if password == "" {
		password = defaultSeedAdminPassword
	}

	return adminCredentials{email: email, password: password}
}

const (
	sessionLoginPath       = "/api/v1/users/login"
	sessionLogoutPath      = "/api/v1/users/logout"
	sessionCurrentUserPath = "/api/v1/users/current"

	// loginBackoff is the minimum interval between login attempts, whatever
	// the previous attempt's outcome: a page load fires many API calls at
	// once, and every one of them can race through the auth-failure path
	// before the first Set-Cookie reaches the browser; likewise a genuine
	// permission 403 must not turn every denied request into a fresh login.
	loginBackoff = 2 * time.Second

	// defaultLoginTimeout bounds the whole login operation (request and
	// response drain). The login runs while t.mu is held, so a stalled engine
	// must not block every other request needing session state.
	defaultLoginTimeout = 10 * time.Second

	// maxReplayBody is the largest request body the transport buffers so the
	// request can be retried after a login. Dashboard requests are small;
	// anything larger is passed through without auto-login support.
	maxReplayBody = 8 << 20
)

// sessionTransport establishes the dashboard's API session transparently. The
// ui-token gate already proves the browser was handed the URL printed by
// 'hatchet embedded-ui', so the user should never see a login screen: when a
// proxied API request fails authentication, the transport logs in to the
// embedded API as the seeded admin, retries the request with the fresh
// session cookie, and relays the session's Set-Cookie headers to the browser
// so subsequent requests (and session expiry mid-session) keep working.
type sessionTransport struct {
	base  http.RoundTripper
	creds adminCredentials

	// loginURL is the embedded API's login endpoint on the proxy target,
	// including the target's path prefix. loginPath and logoutPath are the
	// corresponding request paths on the target used to recognize the session
	// lifecycle endpoints. currentUserURL is the current-user endpoint used to
	// probe whether a session the transport did not establish is still valid.
	loginURL       string
	loginPath      string
	logoutPath     string
	currentUserURL string

	// loginTimeout bounds each login operation; defaults to
	// defaultLoginTimeout (overridable in tests).
	loginTimeout time.Duration

	// logf reports login failures (rate-limited by loginBackoff).
	logf func(format string, args ...interface{})

	mu          sync.Mutex
	cookies     []*http.Cookie // session cookies from the last successful login
	obtainedAt  time.Time
	lastErr     error
	attemptedAt time.Time // start of the last login attempt, successful or not
}

func newSessionTransport(base http.RoundTripper, target *url.URL, creds adminCredentials, logf func(format string, args ...interface{})) *sessionTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	if logf == nil {
		logf = func(string, ...interface{}) {}
	}

	// Join the session endpoints onto the full target URL so a target under a
	// path prefix keeps its prefix, matching the metadata check and the
	// reverse proxy's own URL rewriting. Credentials must only ever be sent
	// to the validated target.
	loginTarget := target.JoinPath(sessionLoginPath)
	logoutTarget := target.JoinPath(sessionLogoutPath)
	currentUserTarget := target.JoinPath(sessionCurrentUserPath)

	// JoinPath leaves the leading slash off when the target has no path;
	// proxied request paths always start with one.
	if !strings.HasPrefix(loginTarget.Path, "/") {
		loginTarget.Path = "/" + loginTarget.Path
	}
	if !strings.HasPrefix(logoutTarget.Path, "/") {
		logoutTarget.Path = "/" + logoutTarget.Path
	}
	if !strings.HasPrefix(currentUserTarget.Path, "/") {
		currentUserTarget.Path = "/" + currentUserTarget.Path
	}

	return &sessionTransport{
		base:           base,
		creds:          creds,
		loginURL:       loginTarget.String(),
		loginPath:      loginTarget.Path,
		logoutPath:     logoutTarget.Path,
		currentUserURL: currentUserTarget.String(),
		loginTimeout:   defaultLoginTimeout,
		logf:           logf,
	}
}

func (t *sessionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !t.eligible(req) {
		return t.base.RoundTrip(req)
	}

	body, replayable, err := bufferRequestBody(req)
	if err != nil {
		return nil, err
	}
	if !replayable {
		return t.base.RoundTrip(req)
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil || !isAuthFailure(resp.StatusCode) {
		return resp, err
	}

	// A user can sign in manually as a different account (the printed
	// credentials, or any user created in the dashboard). If the failed
	// request carries a session the transport did not establish and that
	// session is still valid, this is a genuine authorization denial for that
	// account: pass it through instead of replaying the request as the seeded
	// admin.
	if t.carriesValidForeignSession(req) {
		return resp, nil
	}

	cookies, fromCache, sessionErr := t.sessionCookies(req)
	if sessionErr != nil {
		// Leave the original auth failure untouched: the dashboard falls back
		// to its login screen and the printed credentials still work.
		return resp, nil
	}

	discardResponse(resp)

	retryResp, err := t.retryWithCookies(req, body, cookies)
	if err != nil {
		return nil, err
	}

	// A cached session can itself be stale (for example after the API
	// restarted): refresh it with a real login and retry once more.
	if isAuthFailure(retryResp.StatusCode) && fromCache {
		if fresh, refreshErr := t.refreshSession(req, cookies); refreshErr == nil {
			discardResponse(retryResp)

			cookies = fresh

			retryResp, err = t.retryWithCookies(req, body, cookies)
			if err != nil {
				return nil, err
			}
		}
	}

	// Persist the session on the browser regardless of the retry's outcome so
	// later requests carry it directly.
	attachSessionCookies(retryResp, cookies)

	return retryResp, nil
}

// retryWithCookies re-sends the (buffered) request with the session cookies.
func (t *sessionTransport) retryWithCookies(req *http.Request, body []byte, cookies []*http.Cookie) (*http.Response, error) {
	retry := req.Clone(req.Context())
	if body != nil {
		retry.Body = io.NopCloser(bytes.NewReader(body))
	}

	replaceRequestCookies(retry, cookies)

	return t.base.RoundTrip(retry)
}

// eligible reports whether the transport manages authentication for this
// request. Requests that authenticate another way, and the session lifecycle
// endpoints themselves, pass through untouched.
func (t *sessionTransport) eligible(req *http.Request) bool {
	if req.Header.Get("Authorization") != "" {
		return false
	}

	switch req.URL.Path {
	case t.loginPath, t.logoutPath:
		return false
	}

	return true
}

func isAuthFailure(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden
}

// carriesValidForeignSession reports whether req carries session cookies that
// are not the transport's own cached session and that the API still accepts.
// The API answers 403 both to requests with no valid session and to denied
// requests from a valid one, so validity is probed with a current-user lookup
// using exactly the carried cookies. A dead session (expired, revoked, or left
// over from a previous engine) is not treated as foreign and is replaced by
// the automatic sign-in as usual.
func (t *sessionTransport) carriesValidForeignSession(req *http.Request) bool {
	carried := req.Cookies()
	if len(carried) == 0 {
		return false
	}

	t.mu.Lock()
	cached := t.cookies
	t.mu.Unlock()

	if len(cached) > 0 && requestCarriesCookies(req, cached) {
		return false
	}

	ctx, cancel := context.WithTimeout(req.Context(), t.loginTimeout)
	defer cancel()

	probe, err := http.NewRequestWithContext(ctx, http.MethodGet, t.currentUserURL, nil)
	if err != nil {
		return false
	}

	for _, c := range carried {
		probe.AddCookie(&http.Cookie{Name: c.Name, Value: c.Value}) // nolint:gosec // outgoing request cookie; Secure/HttpOnly/SameSite are response attributes
	}

	resp, err := t.base.RoundTrip(probe)
	if err != nil {
		return false
	}
	defer discardResponse(resp)

	return resp.StatusCode == http.StatusOK
}

// sessionCookies returns session cookies expected to satisfy a request that
// just failed authentication, logging in as the seeded admin when the cached
// session is missing or was the one that just failed. fromCache reports
// whether the cookies were reused rather than freshly obtained.
func (t *sessionTransport) sessionCookies(failed *http.Request) (cookies []*http.Cookie, fromCache bool, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	carried := requestCarriesCookies(failed, t.cookies)

	// The cached session was obtained by another request and the failed
	// request did not carry it yet: reuse it.
	if len(t.cookies) > 0 && !carried {
		return t.cookies, true, nil
	}

	// The request already carried a session we just established, so this is
	// most likely a genuine 401/403 (or a revoked session): rate-limit
	// re-login so permission errors do not turn into login storms.
	if carried && time.Since(t.obtainedAt) < loginBackoff {
		return nil, false, fmt.Errorf("the freshly established session was rejected")
	}

	cookies, err = t.loginLocked(failed)
	if err != nil {
		return nil, false, err
	}

	return cookies, false, nil
}

// refreshSession replaces a cached session that just failed with a fresh
// login. If another request already refreshed the cache, the newer session is
// returned instead of logging in again.
func (t *sessionTransport) refreshSession(failed *http.Request, stale []*http.Cookie) ([]*http.Cookie, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.cookies) > 0 && !cookiesEqual(t.cookies, stale) {
		return t.cookies, nil
	}

	return t.loginLocked(failed)
}

// loginLocked performs a login (t.mu must be held), updating the cache and
// applying the attempt backoff.
func (t *sessionTransport) loginLocked(req *http.Request) ([]*http.Cookie, error) {
	// Apply loginBackoff to every login attempt, successful or not: a genuine
	// permission 403 on the cached-session path must not perform a fresh
	// admin login for each denied request.
	if !t.attemptedAt.IsZero() && time.Since(t.attemptedAt) < loginBackoff {
		if t.lastErr != nil {
			return nil, t.lastErr
		}

		return nil, fmt.Errorf("a login was already attempted in the last %s", loginBackoff)
	}

	t.attemptedAt = time.Now()

	cookies, err := t.login(req)
	if err != nil {
		t.lastErr = err
		t.logf("could not sign the dashboard in automatically (sign in manually with the printed credentials): %v", err)

		return nil, err
	}

	t.cookies = cookies
	t.obtainedAt = time.Now()
	t.lastErr = nil

	return cookies, nil
}

// cookiesEqual reports whether the two cookie sets have identical name/value
// pairs.
func cookiesEqual(a, b []*http.Cookie) bool {
	if len(a) != len(b) {
		return false
	}

	values := map[string]string{}
	for _, c := range a {
		values[c.Name] = c.Value
	}

	for _, c := range b {
		if values[c.Name] != c.Value {
			return false
		}
	}

	return true
}

// login performs the email/password login against the embedded API and
// returns the session cookies it set.
func (t *sessionTransport) login(orig *http.Request) ([]*http.Cookie, error) {
	payload, err := json.Marshal(map[string]string{
		"email":    t.creds.email,
		"password": t.creds.password,
	})
	if err != nil {
		return nil, err
	}

	// The login runs while t.mu is held: bound the whole operation (request
	// and response drain) so a stalled engine cannot block other requests on
	// the mutex indefinitely. cancel is deferred before discardResponse so
	// the drain still runs inside the timed scope.
	ctx, cancel := context.WithTimeout(orig.Context(), t.loginTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.loginURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach the login endpoint: %w", err)
	}
	defer discardResponse(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login as %s returned status %d", t.creds.email, resp.StatusCode)
	}

	var cookies []*http.Cookie
	for _, c := range resp.Cookies() {
		// Never collect the proxy's own auth cookie: caching and relaying it
		// would let an engine response overwrite the browser's ui-token. The
		// engine's session cookie name is config-driven (SERVER_AUTH_COOKIE_NAME,
		// default "hatchet"), so the reserved name is denylisted rather than
		// allowlisting a fixed session cookie name.
		if c.Name == uiTokenCookie {
			continue
		}

		if c.Value != "" {
			cookies = append(cookies, c)
		}
	}

	if len(cookies) == 0 {
		return nil, fmt.Errorf("login as %s set no session cookie", t.creds.email)
	}

	return cookies, nil
}

// bufferRequestBody reads the request body into memory so the request can be
// sent twice. It returns the buffered bytes (nil when there is no body) and
// whether the request is replayable; bodies over maxReplayBody are restored
// as a stream and reported as not replayable.
func bufferRequestBody(req *http.Request) ([]byte, bool, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, true, nil
	}

	buf, err := io.ReadAll(io.LimitReader(req.Body, maxReplayBody+1))
	if err != nil {
		_ = req.Body.Close()
		return nil, false, err
	}

	if len(buf) > maxReplayBody {
		req.Body = struct {
			io.Reader
			io.Closer
		}{io.MultiReader(bytes.NewReader(buf), req.Body), req.Body}

		return nil, false, nil
	}

	_ = req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(buf))

	return buf, true, nil
}

// requestCarriesCookies reports whether req already carries every cookie in
// cookies with the same value.
func requestCarriesCookies(req *http.Request, cookies []*http.Cookie) bool {
	if len(cookies) == 0 {
		return false
	}

	sent := map[string]string{}
	for _, c := range req.Cookies() {
		sent[c.Name] = c.Value
	}

	for _, c := range cookies {
		if sent[c.Name] != c.Value {
			return false
		}
	}

	return true
}

// replaceRequestCookies swaps any cookies named like the session cookies for
// the fresh values, keeping unrelated cookies intact.
func replaceRequestCookies(req *http.Request, cookies []*http.Cookie) {
	fresh := map[string]bool{}
	for _, c := range cookies {
		fresh[c.Name] = true
	}

	existing := req.Cookies()
	req.Header.Del("Cookie")

	for _, c := range existing {
		if !fresh[c.Name] {
			req.AddCookie(c)
		}
	}

	for _, c := range cookies {
		req.AddCookie(&http.Cookie{Name: c.Name, Value: c.Value}) // nolint:gosec // outgoing request cookie; Secure/HttpOnly/SameSite are response attributes
	}
}

// attachSessionCookies adds the session's Set-Cookie headers to the response
// so the browser persists the session, unless the response already sets a
// cookie with the same name (for example a logout clearing it).
func attachSessionCookies(resp *http.Response, cookies []*http.Cookie) {
	already := map[string]bool{}
	for _, c := range resp.Cookies() {
		already[c.Name] = true
	}

	for _, c := range cookies {
		if !already[c.Name] {
			resp.Header.Add("Set-Cookie", c.String())
		}
	}
}

func discardResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}

	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	_ = resp.Body.Close()
}
