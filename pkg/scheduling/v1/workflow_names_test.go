package v1

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	v1repo "github.com/hatchet-dev/hatchet/pkg/repository"
)

type fakeWorkflowNamesRepo struct {
	v1repo.SchedulerRepository

	names map[uuid.UUID]string
	err   error
	calls int
}

func (f *fakeWorkflowNamesRepo) ListWorkflowNamesByIds(ctx context.Context, workflowIds []uuid.UUID) (map[uuid.UUID]string, error) {
	f.calls++

	if f.err != nil {
		return nil, f.err
	}

	out := make(map[uuid.UUID]string)

	for _, id := range workflowIds {
		if name, ok := f.names[id]; ok {
			out[id] = name
		}
	}

	return out, nil
}

func TestWorkflowNameCacheResolve(t *testing.T) {
	l := zerolog.Nop()
	ctx := context.Background()

	wf1 := uuid.New()
	wf2 := uuid.New()
	unknown := uuid.New()

	t.Run("fetches misses and caches them", func(t *testing.T) {
		repo := &fakeWorkflowNamesRepo{names: map[uuid.UUID]string{wf1: "one", wf2: "two"}}
		c := newWorkflowNameCache(repo, &l)

		names := c.resolve(ctx, []uuid.UUID{wf1, wf2})

		assert.Equal(t, map[uuid.UUID]string{wf1: "one", wf2: "two"}, names)
		assert.Equal(t, 1, repo.calls)

		names = c.resolve(ctx, []uuid.UUID{wf1, wf2})

		assert.Equal(t, map[uuid.UUID]string{wf1: "one", wf2: "two"}, names)
		assert.Equal(t, 1, repo.calls, "cached ids should not be re-fetched")
	})

	t.Run("unresolvable ids are absent from the result", func(t *testing.T) {
		repo := &fakeWorkflowNamesRepo{names: map[uuid.UUID]string{wf1: "one"}}
		c := newWorkflowNameCache(repo, &l)

		names := c.resolve(ctx, []uuid.UUID{wf1, unknown})

		assert.Equal(t, map[uuid.UUID]string{wf1: "one"}, names)
	})

	t.Run("returns cached names when the fetch fails", func(t *testing.T) {
		repo := &fakeWorkflowNamesRepo{names: map[uuid.UUID]string{wf1: "one"}}
		c := newWorkflowNameCache(repo, &l)

		c.resolve(ctx, []uuid.UUID{wf1})

		repo.err = errors.New("db down")

		names := c.resolve(ctx, []uuid.UUID{wf1, wf2})

		assert.Equal(t, map[uuid.UUID]string{wf1: "one"}, names)
	})
}
