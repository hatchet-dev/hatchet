package cli

import (
	"net/http"
	"net/http/httptest"
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
