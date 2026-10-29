package middleware

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/redirect"
)

type SecurityRequirement interface {
	IsOptional() bool
	NoAuth() bool
	CookieAuth() bool
	BearerAuth() bool
	CustomAuth() bool
}

type RouteInfo struct {
	OperationID string
	Security    SecurityRequirement
	Resources   []string
	Route       *routers.Route
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

func (s *securityRequirement) CustomAuth() bool {
	if s.NoAuth() {
		return false
	}

	for _, requirement := range s.requirements {
		if _, ok := requirement["customAuth"]; ok {
			return true
		}
	}

	return false
}

type MiddlewareFunc func(r *RouteInfo) echo.HandlerFunc

type MiddlewareHandler struct {
	// openapi spec
	swagger *openapi3.T

	// registered middlewares
	mws []MiddlewareFunc
}

func NewMiddlewareHandler(swagger *openapi3.T) (*MiddlewareHandler, error) {
	return &MiddlewareHandler{
		swagger: swagger,
		mws:     make([]MiddlewareFunc, 0),
	}, nil
}

func (m *MiddlewareHandler) Use(mw MiddlewareFunc) {
	m.mws = append(m.mws, mw)
}

// Middleware must be registered as group or route middleware: c.Path() is only set after echo routes.
func (m *MiddlewareHandler) Middleware() (echo.MiddlewareFunc, error) {
	routes, err := buildRouteInfos(m.swagger)
	if err != nil {
		return nil, err
	}

	f := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			routeInfo, ok := routes[routeKey(c.Request().Method, c.Path())]

			if !ok {
				return echo.ErrNotFound
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

var specPathParam = regexp.MustCompile(`\{([.;?]?)([^{}*]+)(\*?)\}`)

func routeKey(method, echoPath string) string {
	return method + ":" + echoPath
}

func buildRouteInfos(swagger *openapi3.T) (map[string]*RouteInfo, error) {
	routes := make(map[string]*RouteInfo)

	for path, pathItem := range swagger.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			security := operation.Security

			// If there aren't any security requirements for the operation, use the global security requirements
			if security == nil {
				security = &swagger.Security
			}

			var isOptional bool

			if xSecurityOptional := operation.Extensions["x-security-optional"]; xSecurityOptional != nil {
				isOptional, _ = xSecurityOptional.(bool)
			}

			var resources []string

			if resourcesInt := operation.Extensions["x-resources"]; resourcesInt != nil {
				resourcesIntArr, ok := resourcesInt.([]interface{})

				if !ok {
					return nil, fmt.Errorf("x-resources of operation %s must be a list", operation.OperationID)
				}

				resources = make([]string, len(resourcesIntArr))

				for i, v := range resourcesIntArr {
					resource, ok := v.(string)

					if !ok {
						return nil, fmt.Errorf("x-resources of operation %s must be a list of strings", operation.OperationID)
					}

					resources[i] = resource
				}
			}

			key := routeKey(method, specPathParam.ReplaceAllString(path, ":$2"))

			if _, exists := routes[key]; exists {
				return nil, fmt.Errorf("duplicate route %s in openapi spec", key)
			}

			routes[key] = &RouteInfo{
				OperationID: operation.OperationID,
				Security: &securityRequirement{
					requirements:      *security,
					xSecurityOptional: isOptional,
				},
				Resources: resources,
				Route: &routers.Route{
					Spec:      swagger,
					Path:      path,
					PathItem:  pathItem,
					Method:    method,
					Operation: operation,
				},
			}
		}
	}

	return routes, nil
}
