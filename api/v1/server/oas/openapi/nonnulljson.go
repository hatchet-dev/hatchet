package openapi

import (
	"bytes"
	"encoding/json"
)

type NonNullableJSON map[string]interface{}

func (m *NonNullableJSON) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		*m = NonNullableJSON{}
		return nil
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*m = NonNullableJSON(raw)
	return nil
}

func (m NonNullableJSON) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(map[string]interface{}(m))
}
