//go:build integration

package authn_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/auth/cookie"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/random"
)

// TestValidateOAuthState_EmptyStateBypass reproduces the login-CSRF bypass:
// after a victim completes one OAuth flow, session.Values["oauth_state_github"]
// is left as "" instead of being deleted.  An attacker can then replay
// ?state= (empty string) against that session and ValidateOAuthState incorrectly
// returns true, allowing the attacker's OAuth code to be exchanged within the
// victim's browser session.
func TestValidateOAuthState_EmptyStateBypass(t *testing.T) {
	t.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")

	testutils.RunTestWithDatabase(t, func(conf *database.Layer) error {
		const cookieName = "hatchet-test"

		hashKey, err := random.Generate(16)
		require.NoError(t, err)
		blockKey, err := random.Generate(16)
		require.NoError(t, err)

		ss, err := cookie.NewUserSessionStore(
			cookie.WithCookieSecrets(hashKey, blockKey),
			cookie.WithCookieDomain("localhost"),
			cookie.WithCookieName(cookieName),
			cookie.WithCookieAllowInsecure(true),
			cookie.WithSessionRepository(conf.V1.UserSession()),
		)
		require.NoError(t, err)

		e := echo.New()
		helpers := authn.NewSessionHelpers(ss)

		// ── Step 1: prime the session ──────────────────────────────────────────
		// Simulate what the server does AFTER a successful OAuth round-trip:
		// it calls session.Values[stateKey] = "" instead of deleting the key.
		// We reproduce this by calling SaveKV directly.
		primeReq := httptest.NewRequest(http.MethodGet, "/", nil)
		primeRec := httptest.NewRecorder()
		primeCtx := e.NewContext(primeReq, primeRec)

		err = helpers.SaveKV(primeCtx, "oauth_state_github", "")
		require.NoError(t, err, "priming session should succeed")

		// Extract the Set-Cookie header so we can attach it to the attack request.
		setCookieRaw := primeRec.Header().Get("Set-Cookie")
		require.NotEmpty(t, setCookieRaw, "response must contain a Set-Cookie header")

		var sessionCookie *http.Cookie
		for _, c := range primeRec.Result().Cookies() {
			if c.Name == cookieName {
				sessionCookie = c
				break
			}
		}
		require.NotNil(t, sessionCookie, "session cookie must be present in response")

		// ── Step 2: attacker fires the exploit ────────────────────────────────
		// The attack URL carries an empty state parameter (?state=).
		// The victim's primed session cookie is attached.
		attackURL := fmt.Sprintf("/api/v1/users/github/callback?code=ATTACKER_CODE&state=")
		attackReq := httptest.NewRequest(http.MethodGet, attackURL, nil)
		attackReq.AddCookie(sessionCookie)
		attackRec := httptest.NewRecorder()
		attackCtx := e.NewContext(attackReq, attackRec)

		isValid, _, err := helpers.ValidateOAuthState(attackCtx, "github")

		// ── Step 3: assert the expected (fixed) behaviour ─────────────────────
		// Before the fix, isValid == true and err == nil, proving the bypass.
		// After the fix, the call must return false (and a non-nil error).
		assert.False(t, isValid,
			"empty ?state= must NOT validate; if this assertion fails the bypass is active")
		assert.Error(t, err,
			"ValidateOAuthState must return an error when state is empty")

		return nil
	})
}

// TestValidateOAuthState_LegitimateFlow verifies that the normal OAuth path is
// unaffected by the fix: a correctly generated state round-trips successfully.
func TestValidateOAuthState_LegitimateFlow(t *testing.T) {
	t.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")

	testutils.RunTestWithDatabase(t, func(conf *database.Layer) error {
		const cookieName = "hatchet-test-legit"

		hashKey, err := random.Generate(16)
		require.NoError(t, err)
		blockKey, err := random.Generate(16)
		require.NoError(t, err)

		ss, err := cookie.NewUserSessionStore(
			cookie.WithCookieSecrets(hashKey, blockKey),
			cookie.WithCookieDomain("localhost"),
			cookie.WithCookieName(cookieName),
			cookie.WithCookieAllowInsecure(true),
			cookie.WithSessionRepository(conf.V1.UserSession()),
		)
		require.NoError(t, err)

		e := echo.New()
		helpers := authn.NewSessionHelpers(ss)

		// Start the OAuth flow: SaveOAuthState generates a random state and stores it.
		startReq := httptest.NewRequest(http.MethodGet, "/api/v1/users/github", nil)
		startRec := httptest.NewRecorder()
		startCtx := e.NewContext(startReq, startRec)

		state, err := helpers.SaveOAuthState(startCtx, "github")
		require.NoError(t, err)
		require.NotEmpty(t, state)

		var sessionCookie *http.Cookie
		for _, c := range startRec.Result().Cookies() {
			if c.Name == cookieName {
				sessionCookie = c
				break
			}
		}
		require.NotNil(t, sessionCookie)

		// Callback with the correct state value.
		callbackURL := fmt.Sprintf("/api/v1/users/github/callback?code=REAL_CODE&state=%s", state)
		callbackReq := httptest.NewRequest(http.MethodGet, callbackURL, nil)
		callbackReq.AddCookie(sessionCookie)
		callbackRec := httptest.NewRecorder()
		callbackCtx := e.NewContext(callbackReq, callbackRec)

		isValid, _, err := helpers.ValidateOAuthState(callbackCtx, "github")

		assert.True(t, isValid, "correct state must validate successfully")
		assert.NoError(t, err)

		return nil
	})
}
