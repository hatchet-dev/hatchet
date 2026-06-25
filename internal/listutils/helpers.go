package listutils

import "slices"

func Uniq[T comparable](xs []T) []T {
	seen := make(map[T]struct{})
	result := make([]T, 0)

	for _, x := range xs {
		if _, ok := seen[x]; !ok {
			seen[x] = struct{}{}
			result = append(result, x)
		}
	}

	return result
}

func Any[T comparable](xs []T, target T) bool {
	return slices.Contains(xs, target)
}

func All[T comparable](xs []T, target T) bool {
	for _, x := range xs {
		if x != target {
			return false
		}
	}
	return true
}

func MaxOf[T any](xs []T, fn func(T) int) int {
	m := 0
	for _, x := range xs {
		if c := fn(x); c > m {
			m = c
		}
	}
	return m
}
