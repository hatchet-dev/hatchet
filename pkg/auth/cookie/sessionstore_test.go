//go:build integration

package cookie_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/auth/cookie"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/random"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

func TestSessionStoreSave(t *testing.T) {
	t.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")

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

func TestSessionStoreLogoutClearsCookieWhenSessionRowMissing(t *testing.T) {
	t.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")
	testutils.RunTestWithDatabase(t, func(conf *database.Layer) error {
		const cookieName = "hatchet"

		ss := newSessionStore(t, conf, cookieName)

		req, err := http.NewRequest("GET", "https://hatchet.run", nil)
		if err != nil {
			return err
		}

		session, err := ss.Get(req, cookieName)
		if err != nil {
			return err
		}

		session.Values["authenticated"] = true

		rr := httptest.NewRecorder()
		if err := ss.Save(req, rr, session); err != nil {
			return err
		}

		sessID := session.ID
		_, err = conf.V1.UserSession().Delete(context.Background(), uuid.MustParse(sessID))
		if err != nil {
			return err
		}

		req2, err := http.NewRequest("GET", "https://hatchet.run", nil)
		if err != nil {
			return err
		}

		req2.AddCookie(rr.Result().Cookies()[0])

		session2, err := ss.Get(req2, cookieName)
		if err != nil {
			return err
		}

		session2.Values = make(map[interface{}]interface{})
		session2.Values["authenticated"] = false
		session2.Options.MaxAge = -1

		rr2 := httptest.NewRecorder()
		if err := ss.Save(req2, rr2, session2); err != nil {
			return err
		}

		setCookie := rr2.Result().Header.Get("Set-Cookie")
		assert.Contains(t, setCookie, "Max-Age=0")

		return nil
	})
}

func TestSessionStoreGet(t *testing.T) {
	t.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")
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

func TestSessionStoreRevokedIDIsNeverRevived(t *testing.T) {
	t.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")
	testutils.RunTestWithDatabase(t, func(conf *database.Layer) error {
		ss := newSessionStore(t, conf, "hatchet")
		httpCookie, _ := generateHTTPCookie(t, ss, "hatchet")

		_, session := requestWithCookie(t, ss, httpCookie)
		oldID := uuid.MustParse(session.ID)

		_, err := conf.V1.UserSession().Delete(context.Background(), oldID)
		assert.NoError(t, err)

		req, session := requestWithCookie(t, ss, httpCookie)
		assert.Empty(t, session.ID, "a revoked ID is dropped")

		rr := httptest.NewRecorder()
		assert.NoError(t, ss.Save(req, rr, session))
		assert.NotEqual(t, oldID.String(), session.ID)

		_, err = conf.V1.UserSession().GetById(context.Background(), oldID)
		assert.ErrorIs(t, err, pgx.ErrNoRows)

		return nil
	})
}

func TestSessionStoreSaveRacingRevokeFails(t *testing.T) {
	t.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")
	testutils.RunTestWithDatabase(t, func(conf *database.Layer) error {
		ss := newSessionStore(t, conf, "hatchet")
		httpCookie, _ := generateHTTPCookie(t, ss, "hatchet")

		req, session := requestWithCookie(t, ss, httpCookie)
		oldID := uuid.MustParse(session.ID)

		_, err := conf.V1.UserSession().Delete(context.Background(), oldID)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		assert.ErrorIs(t, ss.Save(req, rr, session), cookie.ErrSessionRevoked)
		assert.Contains(t, rr.Header().Get("Set-Cookie"), "Max-Age=0")

		_, err = conf.V1.UserSession().GetById(context.Background(), oldID)
		assert.ErrorIs(t, err, pgx.ErrNoRows)

		return nil
	})
}

func TestSessionStoreRotateKeepsValuesUnderNewID(t *testing.T) {
	t.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")
	testutils.RunTestWithDatabase(t, func(conf *database.Layer) error {
		ss := newSessionStore(t, conf, "hatchet")
		httpCookie, _ := generateHTTPCookie(t, ss, "hatchet")

		req, session := requestWithCookie(t, ss, httpCookie)
		oldID := uuid.MustParse(session.ID)

		rr := httptest.NewRecorder()
		assert.NoError(t, ss.Save(req, rr, session))
		assert.NoError(t, ss.Rotate(req, rr))

		assert.NotEqual(t, oldID.String(), session.ID)
		assert.Len(t, rr.Result().Cookies(), 1, "exactly one Set-Cookie")

		_, err := conf.V1.UserSession().GetById(context.Background(), oldID)
		assert.ErrorIs(t, err, pgx.ErrNoRows)

		_, rotated := requestWithCookie(t, ss, rr.Result().Cookies()[0])
		assert.Equal(t, session.ID, rotated.ID)
		assert.Equal(t, "mycustomdata", rotated.Values["custom_data"])

		return nil
	})
}

func TestSessionStorePasswordChangeRevokesAllThenRotates(t *testing.T) {
	t.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")
	testutils.RunTestWithDatabase(t, func(conf *database.Layer) error {
		ctx := context.Background()
		ss := newSessionStore(t, conf, "hatchet")

		pass := "old-hash"
		user, err := conf.V1.User().CreateUser(ctx, &v1.CreateUserOpts{
			Email:    uuid.NewString() + "@example.com",
			Password: &pass,
		})
		assert.NoError(t, err)

		current := savedSessionCookie(t, ss, user.ID, map[string]any{"email_verified": "false"})
		other := savedSessionCookie(t, ss, user.ID, nil)

		req, session := requestWithCookie(t, ss, current)
		oldID := uuid.MustParse(session.ID)

		newPass := "new-hash"
		_, err = conf.V1.User().UpdateUser(ctx, user.ID, &v1.UpdateUserOpts{Password: &newPass, RevokeSessions: true})
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		assert.NoError(t, ss.Rotate(req, rr))

		_, err = conf.V1.UserSession().GetById(ctx, oldID)
		assert.ErrorIs(t, err, pgx.ErrNoRows)

		_, otherSession := requestWithCookie(t, ss, other)
		assert.Empty(t, otherSession.ID)

		_, rotated := requestWithCookie(t, ss, rr.Result().Cookies()[0])
		assert.Equal(t, user.ID.String(), rotated.Values["user_id"])
		assert.Equal(t, "false", rotated.Values["email_verified"])

		_, stale := requestWithCookie(t, ss, current)
		assert.Empty(t, stale.ID)

		return nil
	})
}

func requestWithCookie(t *testing.T, ss *cookie.UserSessionStore, c *http.Cookie) (*http.Request, *sessions.Session) {
	req, err := http.NewRequest("GET", "https://hatchet.run", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.AddCookie(c)

	session, err := ss.Get(req, ss.GetName())
	if err != nil {
		t.Fatal(err)
	}

	return req, session
}

func savedSessionCookie(t *testing.T, ss *cookie.UserSessionStore, userID uuid.UUID, values map[string]any) *http.Cookie {
	req, err := http.NewRequest("GET", "https://hatchet.run", nil)
	if err != nil {
		t.Fatal(err)
	}

	session, err := ss.Get(req, ss.GetName())
	if err != nil {
		t.Fatal(err)
	}

	session.Values["user_id"] = userID.String()
	for k, v := range values {
		session.Values[k] = v
	}

	rr := httptest.NewRecorder()
	if err := ss.Save(req, rr, session); err != nil {
		t.Fatal(err)
	}

	return rr.Result().Cookies()[0]
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
		cookie.WithSessionRepository(conf.V1.UserSession()),
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
