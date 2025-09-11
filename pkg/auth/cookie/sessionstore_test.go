//go:build integration

package cookie_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/auth/cookie"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/random"
)

func TestSessionStoreSave(t *testing.T) {
	time.Sleep(10 * time.Second) // TODO temp hack for tenant non-upsert issue
	testutils.RunTestWithDatabase(t, func(conf *database.Layer) error {
		const cookieName = "hatchet"

		ss := newSessionStore(t, conf, cookieName)

		httpCookie, _ := generateHTTPCookie(t, ss, cookieName)

		assert.Equal(t, cookieName, httpCookie.Name, "name is hatchet")
		assert.Equal(t, 2592000, httpCookie.MaxAge, "max age is 30 days")
		assert.Equal(t, "/", httpCookie.Path, "path is index")
		assert.Equal(t, true, httpCookie.Secure, "cookie is secure")
		assert.Equal(t, "hatchet.run", httpCookie.Domain, "domain is hatchet.run")

		return nil
	})
}

func TestSessionStoreGet(t *testing.T) {
	testutils.RunTestWithDatabase(t, func(conf *database.Layer) error {
		const cookieName = "hatchet"

		ss := newSessionStore(t, conf, cookieName)

		httpCookie, _ := generateHTTPCookie(t, ss, cookieName)

		req, err := http.NewRequest("GET", "http://www.example.com", nil)

		if err != nil {
			t.Fatal(err.Error())
		}

		req.AddCookie(httpCookie)

		sess, err := ss.Get(req, cookieName)

		if err != nil {
			t.Fatal(err.Error())
		}

		// ensure that we can recover data successfully
		assert.Equal(t, "mycustomdata", sess.Values["custom_data"].(string), "custom data should be recovered")

		return nil
	})
}

func newSessionStore(t *testing.T, conf *database.Layer, cookieName string) *cookie.UserSessionStore {
	hashKey, err := random.Generate(16)

	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	blockKey, err := random.Generate(16)

	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	ss, err := cookie.NewUserSessionStore(
		cookie.WithCookieSecrets(hashKey, blockKey),
		cookie.WithCookieDomain("hatchet.run"),
		cookie.WithCookieName(cookieName),
		cookie.WithCookieAllowInsecure(false),
		cookie.WithSessionRepository(conf.APIRepository.UserSession()),
	)

	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	return ss
}

func generateHTTPCookie(t *testing.T, ss *cookie.UserSessionStore, cookieName string) (*http.Cookie, string) {
	// construct a new mock request for the domain
	req, err := http.NewRequest("GET", "https://hatchet.run", nil)

	if err != nil {
		t.Fatal("failed to create request", err)
	}

	session, err := ss.Get(req, cookieName)

	if err != nil {
		t.Fatal("failed to get session", err.Error())
	}

	session.Values["custom_data"] = "mycustomdata"

	rr := httptest.NewRecorder()

	if err = ss.Save(req, rr, session); err != nil {
		t.Fatal("Failed to save session:", err.Error())
	}

	setCookieHeader := rr.Result().Header.Get("Set-Cookie")

	httpCookie := getHTTPCookieFromRaw(setCookieHeader)

	return httpCookie, setCookieHeader
}

func getHTTPCookieFromRaw(rawCookie string) *http.Cookie {
	header := http.Header{}
	header.Add("Set-Cookie", rawCookie)
	req := http.Response{Header: header}
	return req.Cookies()[0]
}
