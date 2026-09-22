//go:build !e2e && !load && !rampup && !integration

package hostgrpc

import "github.com/rs/zerolog"

func newNopLogger() *zerolog.Logger {
	l := zerolog.Nop()
	return &l
}
