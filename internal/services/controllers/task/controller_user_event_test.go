//go:build !e2e && !load && !rampup && !integration

package task

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
)

func TestUserEventCandidateMatchesPreserveScope(t *testing.T) {
	scope := "scope-a"
	seenAt := time.Now().UTC()
	eventID := uuid.New()

	candidates := userEventCandidateMatches([]*tasktypes.UserEventTaskPayload{
		{
			EventExternalId: eventID,
			EventSeenAt:     seenAt,
			EventKey:        "shared-user-event-key",
			EventData:       []byte(`{"scope":"a"}`),
			EventScope:      &scope,
		},
		{
			EventExternalId: uuid.New(),
			EventSeenAt:     seenAt,
			EventKey:        "unscoped-user-event-key",
			EventData:       []byte(`{}`),
		},
	})

	require.Len(t, candidates, 2)
	require.Equal(t, eventID, candidates[0].ID)
	require.False(t, candidates[0].EventTimestamp.IsZero())
	require.Equal(t, &scope, candidates[0].ResourceHint)
	require.Nil(t, candidates[1].ResourceHint)
}
