package authn

import (
	"fmt"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/random"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type SessionHelpers struct {
	config *server.ServerConfig
}

func NewSessionHelpers(config *server.ServerConfig) *SessionHelpers {
	return &SessionHelpers{
		config: config,
	}
}

func (s *SessionHelpers) SaveAuthenticated(c echo.Context, user *dbsqlc.User) error {
	session, err := s.config.SessionStore.Get(c.Request(), s.config.SessionStore.GetName())

	if err != nil {
		return err
	}

	session.Values["authenticated"] = true
	session.Values["user_id"] = sqlchelpers.UUIDToStr(user.ID)

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

func (s *SessionHelpers) SaveKV(
	c echo.Context,
	k, v string,
) error {
	session, err := s.config.SessionStore.Get(c.Request(), s.config.SessionStore.GetName())

	if err != nil {
		return err
	}

	session.Values[k] = v

	return session.Save(c.Request(), c.Response())
}

func (s *SessionHelpers) GetKey(
	c echo.Context,
	k string,
) (string, error) {
	session, err := s.config.SessionStore.Get(c.Request(), s.config.SessionStore.GetName())

	if err != nil {
		return "", err
	}

	v, ok := session.Values[k]

	if !ok {
		return "", fmt.Errorf("key not found")
	}

	vStr, ok := v.(string)

	if !ok {
		return "", fmt.Errorf("could not cast value to string")
	}

	return vStr, nil
}

func (s *SessionHelpers) RemoveKey(
	c echo.Context,
	k string,
) error {
	session, err := s.config.SessionStore.Get(c.Request(), s.config.SessionStore.GetName())

	if err != nil {
		return err
	}

	delete(session.Values, k)

	return session.Save(c.Request(), c.Response())
}

func (s *SessionHelpers) SaveOAuthState(
	c echo.Context,
	integration string,
) (string, error) {
	state, err := random.Generate(32)

	if err != nil {
		return "", err
	}

	session, err := s.config.SessionStore.Get(c.Request(), s.config.SessionStore.GetName())

	if err != nil {
		return "", err
	}

	stateKey := fmt.Sprintf("oauth_state_%s", integration)

	// need state parameter to validate when redirected
	session.Values[stateKey] = state

	// need a parameter to indicate that this was triggered through the oauth flow
	session.Values["oauth_triggered"] = true

	if err := session.Save(c.Request(), c.Response()); err != nil {
		return "", err
	}

	return state, nil
}

func (s *SessionHelpers) ValidateOAuthState(
	c echo.Context,
	integration string,
) (isValidated bool, isOAuthTriggered bool, err error) {
	stateKey := fmt.Sprintf("oauth_state_%s", integration)

	session, err := s.config.SessionStore.Get(c.Request(), s.config.SessionStore.GetName())

	if err != nil {
		return false, false, err
	}

	if _, ok := session.Values[stateKey]; !ok {
		return false, false, fmt.Errorf("state parameter not found in session")
	}

	if c.Request().URL.Query().Get("state") != session.Values[stateKey] {
		return false, false, fmt.Errorf("state parameters do not match")
	}

	if isOAuthTriggeredVal, exists := session.Values["oauth_triggered"]; exists {
		var ok bool

		isOAuthTriggered, ok = isOAuthTriggeredVal.(bool)

		isOAuthTriggered = ok && isOAuthTriggered
	}

	// need state parameter to validate when redirected
	session.Values[stateKey] = ""
	session.Values["oauth_triggered"] = false

	if err := session.Save(c.Request(), c.Response()); err != nil {
		return false, false, fmt.Errorf("could not clear session")
	}

	return true, isOAuthTriggered, nil
}

func saveNewSession(c echo.Context, session *sessions.Session) error {
	session.Values["authenticated"] = false

	return session.Save(c.Request(), c.Response())
}
