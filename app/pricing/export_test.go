package pricing

// RegisteredKinds lists every kind in the registry, for the external test package.
func RegisteredKinds() []string {
	kinds := make([]string, 0, len(registry))
	for kind := range registry {
		kinds = append(kinds, kind)
	}
	return kinds
}
