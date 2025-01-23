package middleware

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/redirect"
)

type SecurityRequirement interface {
	IsOptional() bool
	NoAuth() bool
	CookieAuth() bool
	BearerAuth() bool
}

type RouteInfo struct {
	OperationID string
	Security    SecurityRequirement
	Resources   []string
}

type securityRequirement struct {
	requirements []openapi3.SecurityRequirement

	xSecurityOptional bool
}

func (s *securityRequirement) IsOptional() bool {
	return s.xSecurityOptional
}

func (s *securityRequirement) NoAuth() bool {
	return len(s.requirements) == 0
}

func (s *securityRequirement) CookieAuth() bool {
	if s.NoAuth() {
		return false
	}

	for _, requirement := range s.requirements {
		if _, ok := requirement["cookieAuth"]; ok {
			return true
		}
	}

	return false
}

func (s *securityRequirement) BearerAuth() bool {
	if s.NoAuth() {
		return false
	}

	for _, requirement := range s.requirements {
		if _, ok := requirement["bearerAuth"]; ok {
			return true
		}
	}

	return false
}

type MiddlewareFunc func(r *RouteInfo) echo.HandlerFunc

type MiddlewareHandler struct {
	// cache for route info, since we don't want to parse the openapi spec on every request
	cache *lru.Cache[string, *RouteInfo]

	// openapi spec
	swagger *openapi3.T

	// registered middlewares
	mws []MiddlewareFunc
}

func NewMiddlewareHandler(swagger *openapi3.T) (*MiddlewareHandler, error) {
	cache, err := lru.New[string, *RouteInfo](1000)

	if err != nil {
		return nil, err
	}

	return &MiddlewareHandler{
		cache:   cache,
		swagger: swagger,
		mws:     make([]MiddlewareFunc, 0),
	}, nil
}

func (m *MiddlewareHandler) Use(mw MiddlewareFunc) {
	m.mws = append(m.mws, mw)
}

func (m *MiddlewareHandler) Middleware() (echo.MiddlewareFunc, error) {

	router, err := gorillamux.NewRouter(m.swagger)

	if err != nil {
		return nil, err
	}

	f := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()

			var routeInfo *RouteInfo
			var ok bool

			// check the cache for a match, otherwise parse the openapi spec
			if routeInfo, ok = m.cache.Get(getCacheKey(req)); !ok {
				route, _, err := router.FindRoute(req)

				// We failed to find a matching route for the request.
				if err != nil {
					switch e := err.(type) {
					case *routers.RouteError:
						// We've got a bad request, the path requested doesn't match
						// either server, or path, or something.
						return echo.NewHTTPError(http.StatusBadRequest, e.Reason)
					default:
						// This should never happen today, but if our upstream code changes,
						// we don't want to crash the server, so handle the unexpected error.
						return echo.NewHTTPError(http.StatusInternalServerError,
							fmt.Sprintf("error validating route: %s", err.Error()))
					}
				}

				security := route.Operation.Security

				// If there aren't any security requirements for the operation, use the global security requirements
				if security == nil {
					// Use the global security requirements.
					security = &route.Spec.Security
				}

				var isOptional bool

				// read x-security-optional from x-resources
				xSecurityOptional := route.Operation.Extensions["x-security-optional"]

				if xSecurityOptional != nil {
					isOptional = xSecurityOptional.(bool)
				}

				// read resources from x-resources
				var resources []string
				resourcesInt := route.Operation.Extensions["x-resources"]

				if resourcesInt != nil {
					resourcesIntArr := resourcesInt.([]interface{})

					resources = make([]string, len(resourcesIntArr))

					for i, v := range resourcesIntArr {
						resources[i] = v.(string)
					}
				}

				routeInfo = &RouteInfo{
					OperationID: route.Operation.OperationID,
					Security: &securityRequirement{
						requirements:      *security,
						xSecurityOptional: isOptional,
					},
					Resources: resources,
				}

				m.cache.Add(getCacheKey(req), routeInfo)
			}

			for _, middlewareFunc := range m.mws {
				if err := middlewareFunc(routeInfo)(c); err != nil {
					// in the case of a redirect, we don't want to return an error but we want to stop the
					// middleware chain
					if errors.Is(err, redirect.ErrRedirect) {
						return nil
					}

					return err
				}
			}

			return next(c)
		}
	}

	return f, nil
}

func getCacheKey(req *http.Request) string {
	return req.Method + ":" + req.URL.Path
}
