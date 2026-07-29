package listutils

import "slices"

// inspiration: uniq/1 https://elixir.hexdocs.pm/Enum.html#uniq/1
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

// inspiration: uniq_by/2 https://elixir.hexdocs.pm/Enum.html#uniq_by/2
func UniqBy[T comparable, K comparable](xs []T, fn func(x T) K) []T {
	seen := make(map[K]struct{})
	result := make([]T, 0)

	for _, x := range xs {
		cmp := fn(x)
		if _, ok := seen[cmp]; !ok {
			seen[cmp] = struct{}{}
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

// inspiration: max/3 https://elixir.hexdocs.pm/Enum.html#max/3
func MaxOf[T any](xs []T, fn func(T) int) int {
	m := 0
	for _, x := range xs {
		if c := fn(x); c > m {
			m = c
		}
	}
	return m
}
