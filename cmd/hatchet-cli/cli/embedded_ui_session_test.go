package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeEmbeddedAPI mimics the embedded API's cookie auth: a login endpoint
// that checks credentials and sets a session cookie, and API endpoints that
// return 403 without a valid session (matching the real authn middleware).
type fakeEmbeddedAPI struct {
	t *testing.T

	sessionValue string
	logins       atomic.Int64

	// alwaysDeny makes the API endpoints return 403 even with a valid
	// session, modeling a genuine permission denial.
	alwaysDeny bool

	// extraLoginCookie is an additional cookie the login response sets.
	extraLoginCookie *http.Cookie

	lastAPIRequest atomic.Pointer[http.Request]
}

func (f *fakeEmbeddedAPI) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/users/login", func(w http.ResponseWriter, r *http.Request) {
		f.logins.Add(1)

		var body struct{ Email, Password string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if body.Email != "admin@example.com" || body.Password != "Admin123!!" {
			http.Error(w, "invalid credentials", http.StatusBadRequest)
			return
		}

		http.SetCookie(w, &http.Cookie{Name: "hatchet", Value: f.sessionValue, Path: "/", HttpOnly: true})
		if f.extraLoginCookie != nil {
			http.SetCookie(w, f.extraLoginCookie)
		}
		_, _ = w.Write([]byte(`{"email":"admin@example.com"}`))
	})

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		clone := r.Clone(r.Context())
		body, _ := io.ReadAll(r.Body)
		clone.Body = io.NopCloser(strings.NewReader(string(body)))
		f.lastAPIRequest.Store(clone)

		c, err := r.Cookie("hatchet")
		if err != nil || c.Value != f.sessionValue || f.alwaysDeny {
			http.Error(w, "Please provide valid credentials", http.StatusForbidden)
			return
		}

		echo, _ := json.Marshal(string(body))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"echo":` + string(echo) + `}`))
	})

	return mux
}

func newTestSessionTransport(t *testing.T, api *fakeEmbeddedAPI, creds adminCredentials) (*sessionTransport, *httptest.Server) {
	t.Helper()

	srv := httptest.NewServer(api.handler())
	t.Cleanup(srv.Close)

	target, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	return newSessionTransport(nil, target, creds, t.Logf), srv
}

func defaultTestCreds() adminCredentials {
	return adminCredentials{email: "admin@example.com", password: "Admin123!!", isDefault: true}
}

func do(t *testing.T, rt http.RoundTripper, req *http.Request) *http.Response {
	t.Helper()

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("round trip: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	return resp
}

func TestSessionTransportEstablishesSession(t *testing.T) {
	api := &fakeEmbeddedAPI{t: t, sessionValue: "session-1"}
	rt, srv := newTestSessionTransport(t, api, defaultTestCreds())

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/users/current", nil)
	resp := do(t, rt, req)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}

	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "hatchet" {
			sessionCookie = c
		}
	}
	if sessionCookie == nil || sessionCookie.Value != "session-1" {
		t.Fatalf("expected a relayed hatchet session cookie, got %v", resp.Cookies())
	}

	if got := api.logins.Load(); got != 1 {
		t.Errorf("expected 1 login, got %d", got)
	}
}

func TestSessionTransportReplacesExpiredSession(t *testing.T) {
	api := &fakeEmbeddedAPI{t: t, sessionValue: "session-2"}
	rt, srv := newTestSessionTransport(t, api, defaultTestCreds())

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/users/current", nil)
	req.AddCookie(&http.Cookie{Name: "hatchet", Value: "expired"})
	resp := do(t, rt, req)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}

	sent := api.lastAPIRequest.Load()
	if c, err := sent.Cookie("hatchet"); err != nil || c.Value != "session-2" {
		t.Fatalf("expected the retry to carry the fresh session, got %v", sent.Header.Get("Cookie"))
	}
}

func TestSessionTransportReusesSessionAcrossRequests(t *testing.T) {
	api := &fakeEmbeddedAPI{t: t, sessionValue: "session-3"}
	rt, srv := newTestSessionTransport(t, api, defaultTestCreds())

	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest("GET", srv.URL+"/api/v1/tenants/x/workflows", nil)
		resp := do(t, rt, req)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200", i, resp.StatusCode)
		}
	}

	if got := api.logins.Load(); got != 1 {
		t.Errorf("expected the session to be reused after one login, got %d logins", got)
	}
}

func TestSessionTransportReplaysRequestBody(t *testing.T) {
	api := &fakeEmbeddedAPI{t: t, sessionValue: "session-4"}
	rt, srv := newTestSessionTransport(t, api, defaultTestCreds())

	req, _ := http.NewRequest("POST", srv.URL+"/api/v1/tenants/x/workflows/trigger", strings.NewReader(`{"input":1}`))
	resp := do(t, rt, req)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}

	sent := api.lastAPIRequest.Load()
	body, _ := io.ReadAll(sent.Body)
	if string(body) != `{"input":1}` {
		t.Fatalf("expected the retry to replay the body, got %q", body)
	}
}

func TestSessionTransportPassesThroughFailedLogin(t *testing.T) {
	api := &fakeEmbeddedAPI{t: t, sessionValue: "session-5"}
	rt, srv := newTestSessionTransport(t, api, adminCredentials{email: "someone@else.com", password: "Wrong123!!"})

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/users/current", nil)
	resp := do(t, rt, req)

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("got %d, want the original 403 when auto-login fails", resp.StatusCode)
	}

	// A second failure within the backoff window must not log in again.
	req2, _ := http.NewRequest("GET", srv.URL+"/api/v1/users/current", nil)
	_ = do(t, rt, req2)

	if got := api.logins.Load(); got != 1 {
		t.Errorf("expected failed logins to back off, got %d login attempts", got)
	}
}

func TestSessionTransportSkipsBearerAndSessionEndpoints(t *testing.T) {
	api := &fakeEmbeddedAPI{t: t, sessionValue: "session-6"}
	rt, srv := newTestSessionTransport(t, api, defaultTestCreds())

	bearer, _ := http.NewRequest("GET", srv.URL+"/api/v1/tenants/x/workflows", nil)
	bearer.Header.Set("Authorization", "Bearer some-token")
	if resp := do(t, rt, bearer); resp.StatusCode != http.StatusForbidden {
		t.Errorf("bearer request: got %d, want the API's own 403", resp.StatusCode)
	}

	logout, _ := http.NewRequest("POST", srv.URL+"/api/v1/users/logout", nil)
	if resp := do(t, rt, logout); resp.StatusCode != http.StatusForbidden {
		t.Errorf("logout: got %d, want pass-through 403", resp.StatusCode)
	}

	if got := api.logins.Load(); got != 0 {
		t.Errorf("expected no login attempts, got %d", got)
	}
}

func TestSessionTransportDoesNotRetryGenuineForbidden(t *testing.T) {
	api := &fakeEmbeddedAPI{t: t, sessionValue: "session-7"}
	rt, srv := newTestSessionTransport(t, api, defaultTestCreds())

	// Establish the session.
	first, _ := http.NewRequest("GET", srv.URL+"/api/v1/users/current", nil)
	if resp := do(t, rt, first); resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}

	// Simulate a genuine permission 403 by revoking the session server-side
	// while the browser still carries the fresh cookie.
	api.sessionValue = "rotated"

	second, _ := http.NewRequest("GET", srv.URL+"/api/v1/users/current", nil)
	second.AddCookie(&http.Cookie{Name: "hatchet", Value: "session-7"})
	if resp := do(t, rt, second); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("got %d, want 403 (no immediate re-login for a just-established session)", resp.StatusCode)
	}

	if got := api.logins.Load(); got != 1 {
		t.Errorf("expected re-login to be rate-limited, got %d logins", got)
	}
}

func TestSessionTransportRefreshesStaleCachedSession(t *testing.T) {
	api := &fakeEmbeddedAPI{t: t, sessionValue: "session-8"}
	rt, srv := newTestSessionTransport(t, api, defaultTestCreds())

	// Establish and cache a session.
	first, _ := http.NewRequest("GET", srv.URL+"/api/v1/users/current", nil)
	if resp := do(t, rt, first); resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}

	// Simulate an API restart invalidating the cached session.
	api.sessionValue = "session-9"

	// Every login attempt is rate-limited by loginBackoff; move the transport
	// outside the window so the refresh below is permitted.
	rt.mu.Lock()
	rt.attemptedAt = rt.attemptedAt.Add(-loginBackoff)
	rt.mu.Unlock()

	// A fresh browser (no cookies) must still get a working session in one
	// request: cached session fails, transport refreshes and retries.
	second, _ := http.NewRequest("GET", srv.URL+"/api/v1/users/current", nil)
	resp := do(t, rt, second)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200 after refreshing the stale cached session", resp.StatusCode)
	}

	var relayed string
	for _, c := range resp.Cookies() {
		if c.Name == "hatchet" {
			relayed = c.Value
		}
	}
	if relayed != "session-9" {
		t.Fatalf("expected the refreshed session to be relayed, got %q", relayed)
	}

	if got := api.logins.Load(); got != 2 {
		t.Errorf("expected exactly 2 logins, got %d", got)
	}
}

func TestSessionTransportGenuineForbiddenLimitsRefreshLogins(t *testing.T) {
	// The API accepts logins but denies the operation itself, so every request
	// without the cached cookie takes the reuse-then-refresh path. The refresh
	// must be rate-limited: genuine permission denials must not perform a
	// fresh admin login per request.
	api := &fakeEmbeddedAPI{t: t, sessionValue: "session-10", alwaysDeny: true}
	rt, srv := newTestSessionTransport(t, api, defaultTestCreds())

	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", srv.URL+"/api/v1/tenants/x/workflows", nil)
		resp := do(t, rt, req)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("request %d: got %d, want the genuine 403", i, resp.StatusCode)
		}
	}

	if got := api.logins.Load(); got != 1 {
		t.Errorf("expected at most 1 login for rapid genuine 403s, got %d", got)
	}
}

func TestSessionTransportLoginKeepsTargetPathPrefix(t *testing.T) {
	// The embedded API lives under a path prefix; the login must go to the
	// validated target's prefixed login endpoint, never to the origin root.
	api := &fakeEmbeddedAPI{t: t, sessionValue: "session-11"}

	var rootLogins atomic.Int64
	mux := http.NewServeMux()
	mux.Handle("/embedded/", http.StripPrefix("/embedded", api.handler()))
	mux.HandleFunc("/api/v1/users/login", func(w http.ResponseWriter, r *http.Request) {
		rootLogins.Add(1)
		http.Error(w, "not the embedded API", http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	target, err := url.Parse(srv.URL + "/embedded")
	if err != nil {
		t.Fatal(err)
	}

	rt := newSessionTransport(nil, target, defaultTestCreds(), t.Logf)

	req, _ := http.NewRequest("GET", srv.URL+"/embedded/api/v1/users/current", nil)
	resp := do(t, rt, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200 via the prefixed login endpoint", resp.StatusCode)
	}

	if got := api.logins.Load(); got != 1 {
		t.Errorf("expected 1 login on the prefixed endpoint, got %d", got)
	}
	if got := rootLogins.Load(); got != 0 {
		t.Fatalf("credentials were sent outside the validated prefix (%d root logins)", got)
	}

	// The prefixed session endpoints themselves still pass through untouched.
	logout, _ := http.NewRequest("POST", srv.URL+"/embedded/api/v1/users/logout", nil)
	if resp := do(t, rt, logout); resp.StatusCode != http.StatusForbidden {
		t.Errorf("prefixed logout: got %d, want pass-through 403", resp.StatusCode)
	}
	if got := api.logins.Load(); got != 1 {
		t.Errorf("expected no login for the prefixed logout, got %d", got)
	}
}

func TestSessionTransportLoginTimeout(t *testing.T) {
	// The login runs while the session mutex is held: a login response that
	// never arrives must fail within loginTimeout and unblock other requests.
	release := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/users/login", func(w http.ResponseWriter, r *http.Request) {
		<-release
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Please provide valid credentials", http.StatusForbidden)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) }) // runs before srv.Close, releasing the stalled handler

	target, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	rt := newSessionTransport(nil, target, defaultTestCreds(), t.Logf)
	rt.loginTimeout = 100 * time.Millisecond

	start := time.Now()

	done := make(chan int, 1)
	go func() {
		req, _ := http.NewRequest("GET", srv.URL+"/api/v1/tenants/x/workflows", nil)
		resp, err := rt.RoundTrip(req)
		if err != nil {
			done <- -1
			return
		}
		defer resp.Body.Close()
		done <- resp.StatusCode
	}()

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/users/current", nil)
	if resp := do(t, rt, req); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("got %d, want the original 403 when the login stalls", resp.StatusCode)
	}

	select {
	case status := <-done:
		if status != http.StatusForbidden {
			t.Errorf("concurrent request: got %d, want 403", status)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent request stayed blocked past the login timeout")
	}

	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("login did not fail within the timeout, took %s", elapsed)
	}
}

func TestSessionTransportRejectsReservedLoginCookie(t *testing.T) {
	// A login response that also sets the proxy's own ui-token cookie must not
	// have that cookie cached, sent on retries, or relayed to the browser.
	api := &fakeEmbeddedAPI{
		t:                t,
		sessionValue:     "session-12",
		extraLoginCookie: &http.Cookie{Name: uiTokenCookie, Value: "engine-forged", Path: "/"},
	}
	rt, srv := newTestSessionTransport(t, api, defaultTestCreds())

	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/users/current", nil)
	resp := do(t, rt, req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}

	for _, c := range resp.Cookies() {
		if c.Name == uiTokenCookie {
			t.Errorf("the reserved ui-token cookie was relayed to the browser: %v", c)
		}
	}

	sent := api.lastAPIRequest.Load()
	if _, err := sent.Cookie(uiTokenCookie); err == nil {
		t.Errorf("the reserved ui-token cookie was sent on the retry: %q", sent.Header.Get("Cookie"))
	}

	rt.mu.Lock()
	for _, c := range rt.cookies {
		if c.Name == uiTokenCookie {
			t.Errorf("the reserved ui-token cookie was cached")
		}
	}
	rt.mu.Unlock()
}

func TestStripRequestCookie(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com/api/v1/meta", nil)
	req.AddCookie(&http.Cookie{Name: uiTokenCookie, Value: "secret"})
	req.AddCookie(&http.Cookie{Name: "hatchet", Value: "session"})

	stripRequestCookie(req, uiTokenCookie)

	if _, err := req.Cookie(uiTokenCookie); err == nil {
		t.Errorf("expected the ui-token cookie to be stripped")
	}
	if c, err := req.Cookie("hatchet"); err != nil || c.Value != "session" {
		t.Errorf("expected the session cookie to survive, got %v", req.Header.Get("Cookie"))
	}
}
