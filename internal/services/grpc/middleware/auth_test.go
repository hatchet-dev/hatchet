package middleware

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	authtoken "github.com/hatchet-dev/hatchet/pkg/auth/token"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
)

func TestCouldNotCheckTokenKeepsLimitErrorRetryable(t *testing.T) {
	l := zerolog.Nop()
	a := &GRPCAuthN{l: &l}

	err := a.couldNotCheckToken(context.Background(), fmt.Errorf("%w: %w", authtoken.ErrCouldNotReadTokenFromDatabase, &fairpool.LimitError{Key: "shared", Limit: 2}))

	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
}
