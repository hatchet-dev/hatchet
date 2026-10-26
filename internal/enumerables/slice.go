package enumerables

import (
	"cmp"
	"slices"
)

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
func UniqBy[T any, K comparable](xs []T, fn func(x T) K) []T {
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

// checks if two slices are strictly equal, meaning they have the same elements in the same order
func AreStrictlyEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// checks if two slices are equal, meaning they have the same elements regardless of order
func AreUnorderedEqual[T cmp.Ordered](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	ac := slices.Clone(a)
	bc := slices.Clone(b)
	slices.Sort(ac)
	slices.Sort(bc)

	return slices.Equal(ac, bc)
}

// checks if two slices are equal as sets, meaning they have the same unique elements regardless of order and duplicates
func AreSetEqual[T comparable](a, b []T) bool {
	aSeen := make(map[T]struct{})
	bSeen := make(map[T]struct{})

	for _, x := range a {
		aSeen[x] = struct{}{}
	}

	for _, x := range b {
		bSeen[x] = struct{}{}
	}

	for x := range aSeen {
		if _, ok := bSeen[x]; !ok {
			return false
		}
	}

	for x := range bSeen {
		if _, ok := aSeen[x]; !ok {
			return false
		}
	}

	return true
}

// inspiration: map/2 https://elixir.hexdocs.pm/Enum.html#map/2
func Map[T any, R any](xs []T, fn func(x T) R) []R {
	result := make([]R, 0, len(xs))

	for _, x := range xs {
		result = append(result, fn(x))
	}

	return result
}

func ToMap[T any, K comparable, V any](xs []T, keyFn func(x T) K, valueFn func(x T) V) map[K]V {
	result := make(map[K]V, len(xs))

	for _, x := range xs {
		result[keyFn(x)] = valueFn(x)
	}

	return result
}

func ToMapWithItemsAsValues[T any, K comparable](xs []T, keyFn func(x T) K) map[K]T {
	result := make(map[K]T, len(xs))

	for _, x := range xs {
		result[keyFn(x)] = x
	}

	return result
}

// inspiration: group_by/3 https://elixir.hexdocs.pm/Enum.html#group_by/3
func GroupBy[T any, K comparable](xs []T, keyFn func(x T) K) map[K][]T {
	result := make(map[K][]T)

	for _, x := range xs {
		key := keyFn(x)
		result[key] = append(result[key], x)
	}

	return result
}

// inspiration: MapSet.new/2 https://elixir.hexdocs.pm/MapSet.html#new/2
func ToSet[T any, K comparable](xs []T, keyFn func(x T) K) map[K]struct{} {
	result := make(map[K]struct{}, len(xs))

	for _, x := range xs {
		result[keyFn(x)] = struct{}{}
	}

	return result
}

// inspiration: filter/2 https://elixir.hexdocs.pm/Enum.html#filter/2
func Filter[T any](xs []T, predicate func(x T) bool) []T {
	result := make([]T, 0)

	for _, x := range xs {
		if predicate(x) {
			result = append(result, x)
		}
	}

	return result
}

// inspiration: flat_map/2 https://elixir.hexdocs.pm/Enum.html#flat_map/2
func FlatMap[T any, R any](xs []T, fn func(x T) []R) []R {
	result := make([]R, 0)

	for _, x := range xs {
		result = append(result, fn(x)...)
	}

	return result
}
