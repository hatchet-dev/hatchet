package grpc

import (
	"net/http"
	"path"
	"time"

	connectcors "connectrpc.com/cors"
	"github.com/rs/cors"
)

// corsMaxAge is how long browsers may cache a preflight response.
const corsMaxAge = 2 * time.Hour

// withCORS lets browser SDKs call the Connect and gRPC-Web protocols from other origins. Origins
// follow the REST API's rules: SERVER_ALLOWED_ORIGINS glob patterns, or any origin when unset.
func withCORS(allowedOrigins []string, next http.Handler) http.Handler {
	opts := cors.Options{
		AllowedMethods: connectcors.AllowedMethods(),
		AllowedHeaders: append(connectcors.AllowedHeaders(), "Authorization"),
		ExposedHeaders: connectcors.ExposedHeaders(),
		MaxAge:         int(corsMaxAge.Seconds()),
	}

	if len(allowedOrigins) == 0 {
		opts.AllowedOrigins = []string{"*"}
	} else {
		opts.AllowOriginFunc = func(origin string) bool {
			for _, pattern := range allowedOrigins {
				if matched, err := path.Match(pattern, origin); err == nil && matched {
					return true
				}
			}

			return false
		}
	}

	return cors.New(opts).Handler(next)
}
