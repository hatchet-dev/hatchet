package loaderutils

import (
	"fmt"
	"strings"
)

func ParseHeaders(v string) (map[string]string, error) {
	if v == "" {
		return nil, nil
	}
	out := make(map[string]string)
	for _, pair := range strings.Split(v, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		idx := strings.Index(pair, "=")
		if idx < 0 {
			return nil, fmt.Errorf("invalid header %q: missing '=' separator", pair)
		}
		key := strings.TrimSpace(pair[:idx])
		val := strings.TrimSpace(pair[idx+1:])
		if key == "" {
			return nil, fmt.Errorf("invalid header %q: empty key", pair)
		}
		out[key] = val
	}
	return out, nil
}
