package headers

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
)

func Middleware() middleware.MiddlewareFunc {
	return func(r *middleware.RouteInfo) echo.HandlerFunc {
		return func(c echo.Context) error {
			// ensure the Strict-Transport-Security header is set for all
			// endpoints, as it will help ensure protection against TLS protocol downgrade
			// attacks and cookie hijacking. The header also ensures that browsers only serve
			// requests using a secure HTTPS connection.
			c.Response().Header().Set("Strict-Transport-Security", "max-age=776000; includeSubDomains;  preload")

			// Adds a layer of defense against a range of content
			// injection vulnerabilities by allowing the application to inform the client of expected
			// resource sources. This can be used to prevent scripts from external sources from
			// being injected into the application.
			c.Response().Header().Set("Content-Security-Policy", "script-src 'self'; frame-ancestors 'self'; object-src 'none'; base-uri 'self'")

			// Defines how much information browsers include in the referrer
			// header when users navigate away from the application, which can prevent
			// information disclosure.
			c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Helps prevent clickjacking attacks by preventing the application from being framed in undesirable locations.
			c.Response().Header().Set("X-Frame-Options", "DENY")

			// Used to prevent MIME-sniffing, which is the process in
			// which browsers attempt to determine the content type and encoding of an HTTP
			// response if these properties are incorrect or not present within the HTTP headers.

			c.Response().Header().Set("X-Content-Type-Options", "nosniff")

			// allows you to control which origins can use which browser
			// features, both in the top-level page and in embedded frames. For every feature
			// controlled by Feature Policy, the feature is only enabled in the current document
			// or frame if its origin matches the allowed list of origins.
			c.Response().Header().Set("Permissions-Policy", "geolocation=(), midi=(), notifications=(), push=(), sync-xhr=(), microphone=(), camera=(), magnetometer=(), gyroscope=(), speaker=(), vibrate=(), fullscreen=(self), payment=()")

			// The require-corp directive implies that the document can only access resources that are either from the same origin or have been specifically granted permission otherwise.

			c.Response().Header().Set("Cross-Origin-Embedder-Policy", "require-corp")

			// Cross-Origin-Opener-Policy prevents a document from being opened in a browsing context that has a different opener than its own. This helps prevent attacks where a document is opened in a new tab or window and is able to navigate the opening document to a malicious URL.
			c.Response().Header().Set("Cross-Origin-Opener-Policy", "same-origin")

			// Only requests from the same origin (i.e. scheme + host + port) can read the resource.
			c.Response().Header().Set("Cross-Origin-Resource-Policy", "same-origin")
			return nil
		}
	}
}
