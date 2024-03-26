package authn

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/internal/config/server"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
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
		c.Set("auth_strategy", "cookie")
	}

	if err != nil && !r.Security.BearerAuth() {
		return err
	}

	if err != nil && r.Security.BearerAuth() {
		err = a.handleBearerAuth(c)
		c.Set("auth_strategy", "bearer")

		if err == nil {
			return nil
		}
	}

	return err
}

func (a *AuthN) handleNoAuth(c echo.Context) error {
	store := a.config.SessionStore

	session, err := store.Get(c.Request(), store.GetName())

	if err != nil {
		a.l.Debug().Err(err).Msg("error getting session")

		return GetRedirectWithError(c, a.l, err, "Could not log in. Please try again and make sure cookies are enabled.")
	}

	if auth, ok := session.Values["authenticated"].(bool); ok && auth {
		a.l.Debug().Msgf("user was authenticated when no security schemes permit auth")

		return GetRedirectNoError(c, a.config.Runtime.ServerURL)
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
			if err := saveNewSession(c, session); err != nil {
				a.l.Error().Err(err).Msg("error saving unauthenticated session")
				return forbidden
			}

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

	user, err := a.config.APIRepository.User().GetUserByID(userID)
	if err != nil {
		a.l.Debug().Err(err).Msg("error getting user by id")

		return forbidden
	}

	// set the user and session in context
	c.Set("user", user)
	c.Set("session", session)

	return nil
}

func (a *AuthN) handleBearerAuth(c echo.Context) error {
	forbidden := echo.NewHTTPError(http.StatusForbidden, "Please provide valid credentials")

	// a tenant id must exist in the context in order for the bearer auth to succeed, since
	// these tokens are tenant-scoped
	queriedTenant, ok := c.Get("tenant").(*db.TenantModel)

	if !ok {
		a.l.Debug().Msgf("tenant not found in context")

		return forbidden
	}

	token, err := getBearerTokenFromRequest(c.Request())

	if err != nil {
		a.l.Debug().Err(err).Msg("error getting bearer token from request")

		return forbidden
	}

	// Validate the token.
	tenantId, err := a.config.Auth.JWTManager.ValidateTenantToken(token)

	if err != nil {
		a.l.Debug().Err(err).Msg("error validating tenant token")

		return forbidden
	}

	// Verify that the tenant id which exists in the context is the same as the tenant id
	// in the token.
	if queriedTenant.ID != tenantId {
		a.l.Debug().Msgf("tenant id in token does not match tenant id in context")

		return forbidden
	}

	return nil
}

var errInvalidAuthHeader = fmt.Errorf("invalid authorization header in request")

func getBearerTokenFromRequest(r *http.Request) (string, error) {
	reqToken := r.Header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer")

	if len(splitToken) != 2 {
		return "", errInvalidAuthHeader
	}

	reqToken = strings.TrimSpace(splitToken[1])

	return reqToken, nil
}
