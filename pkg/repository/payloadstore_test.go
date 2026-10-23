package repository

import "testing"

func TestPayloadIsEmpty(t *testing.T) {
	cases := []struct {
		name    string
		payload []byte
		want    bool
	}{
		{name: "nil", payload: nil, want: true},
		{name: "empty", payload: []byte{}, want: true},
		{name: "whitespace only", payload: []byte("  \n"), want: true},
		{name: "empty object", payload: []byte("{}"), want: true},
		{name: "empty object with whitespace", payload: []byte(" {}\n"), want: true},
		{name: "non-empty object", payload: []byte(`{"a":1}`), want: false},
		{name: "empty array", payload: []byte("[]"), want: false},
		{name: "null literal", payload: []byte("null"), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NewPayload[TaskInputShape](tc.payload).IsEmpty(); got != tc.want {
				t.Fatalf("Payload(%q).IsEmpty() = %v, want %v", tc.payload, got, tc.want)
			}
		})
	}
}
