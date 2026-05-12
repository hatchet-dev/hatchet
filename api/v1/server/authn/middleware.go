package authn

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"go.opentelemetry.io/otel/trace"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/redirect"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type AuthN struct {
	config *server.ServerConfig

	helpers *SessionHelpers

	l *zerolog.Logger
}

func NewAuthN(config *server.ServerConfig) *AuthN {
	return &AuthN{
		config:  config,
		helpers: NewSessionHelpers(config.SessionStore),
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

	var cookieErr error

	if r.Security.CookieAuth() {
		cookieErr = a.handleCookieAuth(c)

		c.Set("auth_strategy", "cookie")

		if cookieErr == nil {
			return nil
		}
	}

	if cookieErr != nil && !r.Security.BearerAuth() && !r.Security.CustomAuth() {
		return cookieErr
	}

	var bearerErr error

	if r.Security.BearerAuth() {
		bearerErr = a.handleBearerAuth(c)

		c.Set("auth_strategy", "bearer")

		if bearerErr == nil {
			return nil
		}
	}

	if bearerErr != nil && !r.Security.CustomAuth() {
		return bearerErr
	}

	var customErr error

	if r.Security.CustomAuth() {
		customErr = a.handleCustomAuth(c, r)

		c.Set("auth_strategy", "custom")

		if customErr == nil {
			return nil
		}
	}

	if customErr != nil {
		return customErr
	}

	return fmt.Errorf("no auth strategy found")
}

func (a *AuthN) handleNoAuth(c echo.Context) error {
	store := a.config.SessionStore

	ctx := c.Request().Context()

	session, err := store.Get(c.Request(), store.GetName())

	if err != nil {
		a.l.Debug().Ctx(ctx).Err(err).Msg("error getting session")

		return redirect.GetRedirectWithError(c, a.l, err, "Could not log in. Please try again and make sure cookies are enabled.")
	}

	if auth, ok := session.Values["authenticated"].(bool); ok && auth {
		a.l.Debug().Ctx(ctx).Msgf("user was authenticated when no security schemes permit auth")

		return redirect.GetRedirectNoError(c, a.config.Runtime.ServerURL)
	}

	// set unauthenticated session in context
	c.Set("session", session)

	return nil
}

func (a *AuthN) handleCookieAuth(c echo.Context) error {
	forbidden := echo.NewHTTPError(http.StatusForbidden, "Please provide valid credentials")

	store := a.config.SessionStore

	session, err := store.Get(c.Request(), store.GetName())
	ctx := c.Request().Context()
	if err != nil {
		err = a.helpers.SaveUnauthenticated(c)

		if err != nil {
			a.l.Error().Ctx(ctx).Err(err).Msg("error saving unauthenticated session")
			return fmt.Errorf("error saving unauthenticated session")
		}

		return forbidden
	}

	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		// if the session is new, make sure we write a Set-Cookie header to the response
		if session.IsNew {
			if saveErr := a.helpers.SaveNewSession(c, session); saveErr != nil {
				a.l.Error().Ctx(ctx).Err(saveErr).Msg("error saving unauthenticated session")
				return fmt.Errorf("error saving unauthenticated session")
			}

			c.Set("session", session)
		}

		return forbidden
	}

	// read the user id in the token
	userID, ok := session.Values["user_id"].(string)

	if !ok {
		a.l.Debug().Ctx(ctx).Msgf("could not cast user_id to string")

		return forbidden
	}

	userIdUUID, err := uuid.Parse(userID)

	if err != nil {
		a.l.Debug().Ctx(ctx).Err(err).Msg("error parsing user id uuid from session")

		return forbidden
	}

	user, err := a.config.V1.User().GetUserByID(c.Request().Context(), userIdUUID)
	if err != nil {
		a.l.Debug().Ctx(ctx).Err(err).Msg("error getting user by id")

		if errors.Is(err, pgx.ErrNoRows) {
			return forbidden
		}

		return fmt.Errorf("error getting user by id: %w", err)
	}

	c.Set("user", user)
	c.Set("session", session)

	ctx = context.WithValue(ctx, analytics.UserIDKey, userIdUUID)
	ctx = context.WithValue(ctx, analytics.SourceKey, analytics.SourceUI)

	span := trace.SpanFromContext(ctx)
	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "user.id", Value: userIdUUID})

	c.SetRequest(c.Request().WithContext(ctx))

	return nil
}

func (a *AuthN) handleBearerAuth(c echo.Context) error {
	forbidden := echo.NewHTTPError(http.StatusForbidden, "Please provide valid credentials")

	ctx := c.Request().Context()

	token, isExchangeToken, err := getBearerTokenFromRequest(c.Request())

	if err != nil {
		a.l.Debug().Ctx(ctx).Err(err).Msg("error getting bearer token from request")

		return forbidden
	}

	queriedTenant, isTenantScoped := c.Get("tenant").(*sqlcv1.Tenant)

	if isExchangeToken {
		if a.config.Auth.ExchangeTokenClient == nil {
			a.l.Error().Msgf("exchange token client is not configured")

			return forbidden
		}

		tenantId, userId, validationErr := a.config.Auth.ExchangeTokenClient.ValidateExchangeToken(c.Request().Context(), token)

		if validationErr != nil {
			a.l.Error().Err(validationErr).Msg("error validating exchange token")

			return forbidden
		}

		// we permit exchange token auth if the token is valid and represents a user if the endpoint is not tenant-scoped, because
		// this is effectively a PAT without the tenant scoping
		if isTenantScoped && *tenantId != queriedTenant.ID {
			a.l.Error().Msgf("tenant id in token does not match tenant id in context")

			return forbidden
		}

		user, getUserErr := a.config.V1.User().GetUserByID(c.Request().Context(), *userId)

		if getUserErr != nil {
			a.l.Error().Err(getUserErr).Msg("error getting user by id from exchange token")

			if errors.Is(getUserErr, pgx.ErrNoRows) {
				return forbidden
			}

			return fmt.Errorf("error getting user by id from exchange token: %w", getUserErr)
		}

		// important: user is validated later in the authz step
		c.Set("user", user)
		c.Set(middleware.IsExchangeTokenContextKey, true)

		ctx = context.WithValue(ctx, analytics.UserIDKey, *userId)
		ctx = context.WithValue(ctx, analytics.TenantIDKey, *tenantId)
		ctx = context.WithValue(ctx, analytics.SourceKey, analytics.SourceAPI)

		span := trace.SpanFromContext(ctx)
		telemetry.WithAttributes(span,
			telemetry.AttributeKV{Key: "tenant.id", Value: *tenantId},
			telemetry.AttributeKV{Key: "user.id", Value: *userId},
		)

		c.SetRequest(c.Request().WithContext(ctx))

		return nil
	}

	// If we've reached this point, and it's not tenant-scoped, this endpoint is forbidden
	if !isTenantScoped {
		a.l.Debug().Ctx(ctx).Msgf("bearer token auth attempted on non-tenant-scoped endpoint")

		return forbidden
	}

	// Validate the token.
	tenantId, tokenUUID, err := a.config.Auth.JWTManager.ValidateTenantToken(c.Request().Context(), token)

	if err != nil {
		a.l.Debug().Ctx(ctx).Err(err).Msg("error validating tenant token")

		return forbidden
	}

	// Verify that the tenant id which exists in the context is the same as the tenant id
	// in the token.
	if queriedTenant.ID != tenantId {
		a.l.Debug().Ctx(ctx).Msgf("tenant id in token does not match tenant id in context")

		return forbidden
	}

	c.Set(string(analytics.APITokenIDKey), tokenUUID)

	ctx = context.WithValue(ctx, analytics.APITokenIDKey, tokenUUID)
	ctx = context.WithValue(ctx, analytics.TenantIDKey, tenantId)
	ctx = context.WithValue(ctx, analytics.SourceKey, analytics.SourceAPI)

	span := trace.SpanFromContext(ctx)
	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant.id", Value: tenantId},
	)

	c.SetRequest(c.Request().WithContext(ctx))

	return nil
}

func (a *AuthN) handleCustomAuth(c echo.Context, r *middleware.RouteInfo) error {
	if a.config.Auth.CustomAuthenticator == nil {
		return fmt.Errorf("custom auth handler is not set")
	}

	return a.config.Auth.CustomAuthenticator.Authenticate(c, r)
}

const exchangeTokenHeader = "X-Exchange-Token"

var errInvalidAuthHeader = fmt.Errorf("invalid authorization header in request")

func getBearerTokenFromRequest(r *http.Request) (token string, isExchangeToken bool, err error) {
	reqToken := r.Header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer")

	if len(splitToken) != 2 {
		return "", false, errInvalidAuthHeader
	}

	reqToken = strings.TrimSpace(splitToken[1])

	// if there's also an X-Exchange-Token header, then this is an exchange token request
	if r.Header.Get(exchangeTokenHeader) != "" {
		return reqToken, true, nil
	}

	return reqToken, false, nil
}
