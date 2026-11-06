package authn

import "github.com/hatchet-dev/hatchet/api/v1/server/middleware"

// Control-plane-managed shards must not mint independent browser identities:
// the control plane owns user authentication and its MFA policy.
func isLocalLoginRoute(r *middleware.RouteInfo) bool {
	if r.Route == nil || r.Route.Operation == nil {
		return false
	}
	switch r.Route.Operation.OperationID {
	case "UserCreate", "UserUpdateLogin", "UserUpdateGoogleOauthStart", "UserUpdateGoogleOauthCallback", "UserUpdateGithubOauthStart", "UserUpdateGithubOauthCallback":
		return true
	default:
		return false
	}
}
