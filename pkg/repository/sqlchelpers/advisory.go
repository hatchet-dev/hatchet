package sqlchelpers

import "hash/fnv"

func AdvisoryLockKey(name string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	return int64(h.Sum64()) // nolint:gosec // used only for stable advisory-lock keys, not for security
}
