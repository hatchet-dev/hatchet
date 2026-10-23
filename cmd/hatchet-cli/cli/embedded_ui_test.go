package cli

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
)

func TestTokenGate(t *testing.T) {
	gate := tokenGate("secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	gate.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusForbidden {
		t.Errorf("no token: got %d, want 403", rec.Code)
	}

	rec = httptest.NewRecorder()
	gate.ServeHTTP(rec, httptest.NewRequest("GET", "/?ui_token=wrong", nil))
	if rec.Code != http.StatusForbidden {
		t.Errorf("wrong token: got %d, want 403", rec.Code)
	}

	rec = httptest.NewRecorder()
	gate.ServeHTTP(rec, httptest.NewRequest("GET", "/?ui_token=secret", nil))
	if rec.Code != http.StatusFound {
		t.Fatalf("valid token: got %d, want 302", rec.Code)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Value != "secret" {
		t.Fatalf("valid token: expected session cookie, got %v", cookies)
	}

	req := httptest.NewRequest("GET", "/api/v1/meta", nil)
	req.AddCookie(cookies[0])
	rec = httptest.NewRecorder()
	gate.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("with cookie: got %d, want 200", rec.Code)
	}
}

func TestRewriteResponseCookiesDropsReservedName(t *testing.T) {
	// An ordinary upstream response must not be able to set or clear the
	// proxy's own ui-token cookie on the browser.
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Add("Set-Cookie", "hatchet=session; Path=/; Domain=api.example.com; Secure; HttpOnly")
	resp.Header.Add("Set-Cookie", uiTokenCookie+"=forged; Path=/")

	rewriteResponseCookies(resp)

	cookies := resp.Header["Set-Cookie"]
	if len(cookies) != 1 || setCookieName(cookies[0]) != "hatchet" {
		t.Fatalf("expected only the session cookie to survive, got %v", cookies)
	}
	if strings.Contains(cookies[0], "Domain") || strings.Contains(cookies[0], "Secure") {
		t.Errorf("expected Domain and Secure to be stripped, got %q", cookies[0])
	}

	only := &http.Response{Header: http.Header{}}
	only.Header.Add("Set-Cookie", uiTokenCookie+"=forged; Path=/")

	rewriteResponseCookies(only)

	if got := only.Header["Set-Cookie"]; len(got) != 0 {
		t.Errorf("expected the reserved cookie to be dropped entirely, got %v", got)
	}
}

func TestAllowedUIHosts(t *testing.T) {
	loopback := allowedUIHosts("localhost", 8080)
	for _, h := range []string{"localhost:8080", "127.0.0.1:8080", "[::1]:8080", "LOCALHOST:8080"} {
		if !hostAllowed(loopback, h) {
			t.Errorf("loopback bind: expected %q to be allowed", h)
		}
	}
	for _, h := range []string{"evil.example:8080", "127.0.0.1:9999", "127.0.0.1", ""} {
		if hostAllowed(loopback, h) {
			t.Errorf("loopback bind: expected %q to be rejected", h)
		}
	}

	custom := allowedUIHosts("myhost.internal", 9000)
	if !hostAllowed(custom, "myhost.internal:9000") {
		t.Errorf("custom bind: expected the custom host to be allowed")
	}
	if hostAllowed(custom, "localhost:9000") {
		t.Errorf("custom non-loopback bind: expected loopback hosts to be rejected")
	}

	wildcard := allowedUIHosts("0.0.0.0", 9000)
	if !hostAllowed(wildcard, "localhost:9000") || !hostAllowed(wildcard, "127.0.0.1:9000") {
		t.Errorf("wildcard bind: expected loopback hosts to be allowed")
	}
}

func TestOriginGate(t *testing.T) {
	allowed := allowedUIHosts("localhost", 8080)

	tests := []struct {
		name   string
		method string
		host   string
		origin string
		want   int
	}{
		{"get own host", "GET", "127.0.0.1:8080", "", http.StatusOK},
		{"get localhost alias", "GET", "localhost:8080", "", http.StatusOK},
		{"get unknown host", "GET", "evil.example:8080", "", http.StatusForbidden},
		{"get wrong port", "GET", "127.0.0.1:9999", "", http.StatusForbidden},
		{"post no origin", "POST", "127.0.0.1:8080", "", http.StatusOK},
		{"post own origin", "POST", "127.0.0.1:8080", "http://127.0.0.1:8080", http.StatusOK},
		{"post origin host alias", "POST", "127.0.0.1:8080", "http://localhost:8080", http.StatusOK},
		{"post cross-origin port", "POST", "127.0.0.1:8080", "http://127.0.0.1:9999", http.StatusForbidden},
		{"post null origin", "POST", "127.0.0.1:8080", "null", http.StatusForbidden},
		{"post https origin", "POST", "127.0.0.1:8080", "https://127.0.0.1:8080", http.StatusForbidden},
		{"delete cross-origin", "DELETE", "127.0.0.1:8080", "http://evil.example", http.StatusForbidden},
		{"options preflight cross-origin", "OPTIONS", "127.0.0.1:8080", "http://localhost:9999", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			served := false
			gate := originGate(allowed, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				served = true
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(tt.method, "http://"+tt.host+"/api/v1/tenants/x", nil)
			req.Host = tt.host
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			rec := httptest.NewRecorder()
			gate.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Errorf("got %d, want %d", rec.Code, tt.want)
			}
			if served != (tt.want == http.StatusOK) {
				t.Errorf("handler served=%v, want %v", served, tt.want == http.StatusOK)
			}
		})
	}
}

func TestOriginGateRejectsBeforeProxying(t *testing.T) {
	// A cross-origin request with valid cookies must be rejected before any
	// proxying, header rewriting, or automatic login: the upstream API must
	// see nothing.
	api := &fakeEmbeddedAPI{t: t, sessionValue: "session-gate"}

	srv := httptest.NewServer(api.handler())
	t.Cleanup(srv.Close)

	target, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			stripRequestCookie(pr.Out, uiTokenCookie)
		},
	}
	proxy.Transport = newSessionTransport(nil, target, defaultTestCreds(), t.Logf)

	gate := originGate(allowedUIHosts("localhost", 8080), tokenGate("secret", proxy))

	send := func(method, host, origin string) int {
		req := httptest.NewRequest(method, "http://"+host+"/api/v1/tenants/x/workflows/trigger", strings.NewReader(`{}`))
		req.Host = host
		req.AddCookie(&http.Cookie{Name: uiTokenCookie, Value: "secret"})
		if origin != "" {
			req.Header.Set("Origin", origin)
		}

		rec := httptest.NewRecorder()
		gate.ServeHTTP(rec, req)

		return rec.Code
	}

	if got := send("POST", "127.0.0.1:8080", "http://127.0.0.1:9999"); got != http.StatusForbidden {
		t.Fatalf("cross-origin POST: got %d, want 403", got)
	}
	if got := send("GET", "evil.example:8080", ""); got != http.StatusForbidden {
		t.Fatalf("unknown Host with a valid token: got %d, want 403", got)
	}

	if got := api.logins.Load(); got != 0 {
		t.Fatalf("rejected requests triggered %d logins", got)
	}
	if api.lastAPIRequest.Load() != nil {
		t.Fatal("a rejected request reached the upstream API")
	}

	// The UI's own origin passes through and is signed in automatically.
	if got := send("POST", "127.0.0.1:8080", "http://127.0.0.1:8080"); got != http.StatusOK {
		t.Fatalf("same-origin POST: got %d, want 200", got)
	}
}

func TestParseTargetURL(t *testing.T) {
	ok := []string{"http://localhost:8080", "https://hatchet.example.com"}
	for _, s := range ok {
		if _, err := parseTargetURL(s); err != nil {
			t.Errorf("parseTargetURL(%q) unexpected error: %v", s, err)
		}
	}

	bad := []string{"localhost:8080", "ftp://x", "http://", "://nope", ""}
	for _, s := range bad {
		if _, err := parseTargetURL(s); err == nil {
			t.Errorf("parseTargetURL(%q) expected error, got nil", s)
		}
	}
}
