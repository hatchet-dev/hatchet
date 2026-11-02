package enumerables

// inspiration: Map.has_key?/2 https://elixir.hexdocs.pm/Map.html#has_key?/2
func HasKey[K comparable, V any](m map[K]V, key K) bool {
	_, ok := m[key]
	return ok
}
