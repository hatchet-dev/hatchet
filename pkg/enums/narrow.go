package enums

// Narrow converts a value to an enum with the same value names, reporting false when T's Valid method rejects it.
func Narrow[T interface {
	~string
	Valid() bool
}, S ~string](s S) (T, bool) {
	t := T(s)
	return t, t.Valid()
}
