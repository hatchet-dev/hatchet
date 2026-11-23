package authz

import (
	"context"
	"sync"
)

type memoKey struct {
	tenantID string
	orgID    string
	key      Key
}

type requestMemo struct {
	mu   sync.Mutex
	rows map[memoKey]any
}

type requestMemoCtxKey struct{}

// WithRequestCache lets every check in one request share loaded rows; without it Fetch always loads.
func WithRequestCache(ctx context.Context) context.Context {
	if _, ok := ctx.Value(requestMemoCtxKey{}).(*requestMemo); ok {
		return ctx
	}

	return context.WithValue(ctx, requestMemoCtxKey{}, &requestMemo{rows: map[memoKey]any{}})
}

// Fetch returns rows for keys, calling load once for the misses; errors and missing rows are never cached.
func Fetch[T any](ctx context.Context, scope Scope, keys []Key, load func(ctx context.Context, missing []Key) (map[Key]T, error)) (map[Key]T, error) {
	memo, _ := ctx.Value(requestMemoCtxKey{}).(*requestMemo)

	out := make(map[Key]T, len(keys))
	missing := make([]Key, 0, len(keys))

	if memo != nil {
		memo.mu.Lock()
		for _, k := range keys {
			if row, ok := memo.rows[newMemoKey(scope, k)].(T); ok {
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
			memo.rows[newMemoKey(scope, k)] = row
		}
		memo.mu.Unlock()
	}

	for k, row := range loaded {
		out[k] = row
	}

	return out, nil
}

func newMemoKey(scope Scope, k Key) memoKey {
	return memoKey{tenantID: optionalID(scope.TenantID), orgID: optionalID(scope.OrgID), key: k}
}
