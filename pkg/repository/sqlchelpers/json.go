package sqlchelpers

import "encoding/json"

func JSONBToStringMap(data []byte) map[string]string {
	if len(data) == 0 {
		return nil
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	result := make(map[string]string, len(raw))
	for k, v := range raw {
		switch val := v.(type) {
		case string:
			result[k] = val
		default:
			b, err := json.Marshal(val)
			if err == nil {
				result[k] = string(b)
			}
		}
	}

	return result
}
