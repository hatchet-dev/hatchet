package listutils

func Uniq[T comparable](statuses []T) []T {
	seen := make(map[T]struct{})
	result := make([]T, 0)

	for _, status := range statuses {
		if _, ok := seen[status]; !ok {
			seen[status] = struct{}{}
			result = append(result, status)
		}
	}

	return result
}

func Any[T comparable](statuses []T, target T) bool {
	for _, status := range statuses {
		if status == target {
			return true
		}
	}

	return false
}

func All[T comparable](statuses []T, target T) bool {
	for _, status := range statuses {
		if status != target {
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
