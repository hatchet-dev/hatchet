package repository

import "testing"

func TestValidateJSONB_InvalidJSON(t *testing.T) {
	if err := ValidateJSONB([]byte("{"), "field"); err == nil {
		t.Fatalf("expected error for invalid json, got nil")
	}
}

func TestValidateJSONB_ValidJSON(t *testing.T) {
	cases := [][]byte{
		[]byte(`{"a":1}`),
		[]byte(`"a string is valid json"`),
		[]byte(`123`),
		[]byte(`true`),
		[]byte(`null`),
		[]byte(`[]`),
	}

	for _, c := range cases {
		if err := ValidateJSONB(c, "field"); err != nil {
			t.Fatalf("expected nil error for valid json %q, got %v", string(c), err)
		}
	}
}

func TestValidateJSONB_RejectsEncodedNull(t *testing.T) {
	// This byte slice contains the literal substring `\u0000`.
	b := []byte("{\"a\":\"\\u0000\"}")

	if err := ValidateJSONB(b, "field"); err == nil {
		t.Fatalf("expected error for encoded null, got nil")
	}
}
