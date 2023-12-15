package authn

import (
	"github.com/gorilla/sessions"
	"github.com/hatchet-dev/hatchet/internal/config/server"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/labstack/echo/v4"
)

type SessionHelpers struct {
	config *server.ServerConfig
}

func NewSessionHelpers(config *server.ServerConfig) *SessionHelpers {
	return &SessionHelpers{
		config: config,
	}
}

func (s *SessionHelpers) SaveAuthenticated(c echo.Context, user *db.UserModel) error {
	session, err := s.config.SessionStore.Get(c.Request(), s.config.SessionStore.GetName())

	if err != nil {
		return err
	}

	session.Values["authenticated"] = true
	session.Values["user_id"] = user.ID

	return session.Save(c.Request(), c.Response())
}

func (s *SessionHelpers) SaveUnauthenticated(c echo.Context) error {
	session, err := s.config.SessionStore.Get(c.Request(), s.config.SessionStore.GetName())

	if err != nil {
		return err
	}

	// unset all values
	session.Values = make(map[interface{}]interface{})
	session.Values["authenticated"] = false

	// we set the maxage of the session so that the session gets deleted. This avoids cases
	// where the same cookie can get re-authed to a different user, which would be problematic
	// if the session values weren't properly cleared on logout.
	session.Options.MaxAge = -1

	return session.Save(c.Request(), c.Response())
}

func saveNewSession(c echo.Context, session *sessions.Session) error {
	session.Values["authenticated"] = false

	return session.Save(c.Request(), c.Response())
}
