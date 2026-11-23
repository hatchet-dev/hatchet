package authz

type DenyReason string

const (
	NotMember         DenyReason = "not-member"
	RoleInsufficient  DenyReason = "role-insufficient"
	EmailUnverified   DenyReason = "email-unverified"
	TokenRestricted   DenyReason = "token-restricted" // unavailable to API tokens whatever their capabilities, and gRPC for read-only tokens
	MissingCapability DenyReason = "missing-capability"
)

type Decision struct {
	Allowed bool
	Reason  DenyReason // empty when Allowed
}

func Allow() Decision {
	return Decision{Allowed: true}
}

func Deny(reason DenyReason) Decision {
	return Decision{Reason: reason}
}
