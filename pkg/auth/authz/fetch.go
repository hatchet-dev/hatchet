package authz

import (
	"context"
	"sync"
)

// Key must be unique within a request, so callers fold in anything that distinguishes rows (such as a tenant).
type Key struct {
	Kind string
	ID   string
}

type requestMemo struct {
	mu   sync.Mutex
	rows map[Key]any
}

type requestMemoCtxKey struct{}

func WithRequestCache(ctx context.Context) context.Context {
	if _, ok := ctx.Value(requestMemoCtxKey{}).(*requestMemo); ok {
		return ctx
	}

	return context.WithValue(ctx, requestMemoCtxKey{}, &requestMemo{rows: map[Key]any{}})
}

func Fetch[T any](ctx context.Context, keys []Key, load func(ctx context.Context, missing []Key) (map[Key]T, error)) (map[Key]T, error) {
	memo, _ := ctx.Value(requestMemoCtxKey{}).(*requestMemo)

	out := make(map[Key]T, len(keys))
	missing := make([]Key, 0, len(keys))

	if memo != nil {
		memo.mu.Lock()
		for _, k := range keys {
			if row, ok := memo.rows[k].(T); ok {
				out[k] = row
			} else {
				missing = append(missing, k)
			}
		}
		memo.mu.Unlock()
	} else {
		missing = append(missing, keys...)
	}

	if len(missing) == 0 {
		return out, nil
	}

	loaded, err := load(ctx, missing)
	if err != nil {
		return nil, err
	}

	if memo != nil {
		memo.mu.Lock()
		for k, row := range loaded {
			memo.rows[k] = row
		}
		memo.mu.Unlock()
	}

	for k, row := range loaded {
		out[k] = row
	}

	return out, nil
}
