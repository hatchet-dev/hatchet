package authn

import (
	"net/http"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/internal/config/server"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

type AuthN struct {
	config *server.ServerConfig

	helpers *SessionHelpers

	l *zerolog.Logger
}

func NewAuthN(config *server.ServerConfig) *AuthN {
	return &AuthN{
		config:  config,
		helpers: NewSessionHelpers(config),
		l:       config.Logger,
	}
}

func (a *AuthN) Middleware(r *middleware.RouteInfo) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := a.authenticate(c, r)
		if err != nil {
			return err
		}

		return nil
	}
}

func (a *AuthN) authenticate(c echo.Context, r *middleware.RouteInfo) error {
	// if security is optional, continue
	if r.Security.IsOptional() {
		return nil
	}

	if r.Security.NoAuth() {
		return a.handleNoAuth(c)
	}

	var err error

	if r.Security.CookieAuth() {
		err = a.handleCookieAuth(c)
	}

	if err != nil {
		return err
	}

	if err != nil && r.Security.BearerAuth() {
		err = a.handleBearerAuth(c)
	}

	return err
}

func (a *AuthN) handleNoAuth(c echo.Context) error {
	forbidden := echo.NewHTTPError(http.StatusForbidden, "Please provide valid credentials")

	store := a.config.SessionStore

	session, err := store.Get(c.Request(), store.GetName())

	if err != nil {
		a.l.Debug().Err(err).Msg("error getting session")

		return forbidden
	}

	if auth, ok := session.Values["authenticated"].(bool); ok && auth {
		a.l.Debug().Msgf("user was authenticated when no security schemes permit auth")

		return forbidden
	}

	// set unauthenticated session in context
	c.Set("session", session)

	return nil
}

func (a *AuthN) handleCookieAuth(c echo.Context) error {
	forbidden := echo.NewHTTPError(http.StatusForbidden, "Please provide valid credentials")

	store := a.config.SessionStore

	session, err := store.Get(c.Request(), store.GetName())
	if err != nil {
		err = a.helpers.SaveUnauthenticated(c)

		if err != nil {
			a.l.Error().Err(err).Msg("error saving unauthenticated session")
		}

		return forbidden
	}

	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		// if the session is new, make sure we write a Set-Cookie header to the response
		if session.IsNew {
			saveNewSession(c, session)

			c.Set("session", session)
		}

		return forbidden
	}

	// read the user id in the token
	userID, ok := session.Values["user_id"].(string)

	if !ok {
		a.l.Debug().Msgf("could not cast user_id to string")

		return forbidden
	}

	user, err := a.config.Repository.User().GetUserByID(userID)

	// set the user and session in context
	c.Set("user", user)
	c.Set("session", session)

	return nil
}

func (a *AuthN) handleBearerAuth(c echo.Context) error {
	panic("implement me")
}
