package transformers

type payloadOptions struct {
	includePayloads bool
}

type PayloadOption func(*payloadOptions)

// WithPayloads includes or omits input, output, and event payload fields.
// Additional metadata is always included. Omitted options default to including payloads.
func WithPayloads(include bool) PayloadOption {
	return func(o *payloadOptions) {
		o.includePayloads = include
	}
}

func applyPayloadOptions(opts []PayloadOption) payloadOptions {
	o := payloadOptions{includePayloads: true}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func boolPtr(v bool) *bool {
	return &v
}

func emptyJSON() map[string]interface{} {
	return map[string]interface{}{}
}
