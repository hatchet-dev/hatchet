package serverlessoperator

import (
	"errors"

	"github.com/hatchet-dev/hatchet/pkg/encryption"
)

// ErrFIPSOnlyMode is returned when the process runs with GODEBUG=fips140=only. The durable
// relay dials endpoints over RFC 6455 websockets, whose upgrade handshake is specified over
// SHA-1, and Go refuses SHA-1 in that mode; fips140=on allows it.
var ErrFIPSOnlyMode = errors.New("the serverless operator cannot run under GODEBUG=fips140=only: the websocket upgrade handshake needs SHA-1, which that mode refuses; run with fips140=on or disable the operator")

// CheckRuntimeSupport reports whether this process can run the operator at all. It runs once
// at startup so an unsupported mode fails the boot instead of the first durable delivery.
func CheckRuntimeSupport() error {
	if !encryption.SHA1Allowed() {
		return ErrFIPSOnlyMode
	}

	return nil
}
