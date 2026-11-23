package authz

type DenyReason string

const (
	NotMember         DenyReason = "not-member"
	RoleInsufficient  DenyReason = "role-insufficient"
	EmailUnverified   DenyReason = "email-unverified"
	TokenRestricted   DenyReason = "token-restricted" // API token deny list, read-only on gRPC
	NotAPIToken       DenyReason = "not-api-token"    // gRPC only accepts tenant API tokens
	MissingCapability DenyReason = "missing-capability"
	OutOfTagScope     DenyReason = "out-of-tag-scope"
	NotEntitled       DenyReason = "not-entitled"
	TenantArchived    DenyReason = "tenant-archived"
	NotOwned          DenyReason = "not-owned" // operator/worker/member belongs to someone else, or doesn't exist
	RoleEscalation    DenyReason = "role-escalation"
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

type Decisions map[Key]Decision

// All picks the lowest denied key so the reported denial is stable across map iteration orders.
func (d Decisions) All() (Key, Decision) {
	var (
		deniedKey Key
		denied    Decision
		found     bool
	)

	for k, dec := range d {
		if dec.Allowed {
			continue
		}

		if !found || k.String() < deniedKey.String() {
			deniedKey, denied, found = k, dec, true
		}
	}

	if !found {
		return Key{}, Allow()
	}

	return deniedKey, denied
}
