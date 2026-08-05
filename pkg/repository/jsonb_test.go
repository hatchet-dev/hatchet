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
		[]byte(`{"a":"\\u0000"}`),
		[]byte(`{"a":"\\\\u0000"}`),
	}

	for _, c := range cases {
		if err := ValidateJSONB(c, "field"); err != nil {
			t.Fatalf("expected nil error for valid json %q, got %v", string(c), err)
		}
	}
}

func TestValidateJSONB_RejectsEncodedNull(t *testing.T) {
	// This byte slice contains the literal substring `\u0000`.
	cases := [][]byte{
		[]byte(`{"a":"\u0000"}`),
		[]byte(`{"a":"\\\u0000"}`),
		[]byte(`{"foo\u0000":"bar"}`),
		[]byte(`{"f\u0000oo":"bar"}`),
		[]byte(`[{"f\u0000oo":"bar"}]`),
		[]byte(`[{"a":"A","b":"B","c":"C\u0000"}]`),
	}
	for _, c := range cases {
		if isValid := isUnicodeValid(c); isValid {
			t.Fatalf("expected invalid unicode for json %q, got valid", string(c))
		}
	}
}

func TestSanitizeJSONB(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"clean object", `{"a":"b"}`, `{"a":"b"}`},
		{"null escape in a value", `{"a":"b\u0000c"}`, `{"a":"bc"}`},
		{"null escape in a key", `{"foo\u0000":"bar"}`, `{"foo":"bar"}`},
		{"null escape in a nested value", `[{"a":"A","b":"B","c":"C\u0000"}]`, `[{"a":"A","b":"B","c":"C"}]`},
		{"only a null escape", `{"a":"\u0000"}`, `{"a":""}`},
		{"top level string", `"\u0000abc"`, `"abc"`},
		{"escaped backslash is not the start of an escape", `{"a":"\\u0000"}`, `{"a":"\\u0000"}`},
		{"escaped backslash followed by a null escape", `{"a":"\\\u0000"}`, `{"a":"\\"}`},
		{"raw NUL byte", "{\"a\":\"b\x00c\"}", `{"a":"bc"}`},
		{"empty", "", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := string(SanitizeJSONB([]byte(c.in)))

			if got != c.want {
				t.Fatalf("SanitizeJSONB(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestSanitizeJSONB_OutputPassesValidation(t *testing.T) {
	cases := []string{
		`{"a":"\u0000"}`,
		`{"a":"\\\u0000"}`,
		`{"foo\u0000":"bar"}`,
		`{"f\u0000oo":"bar"}`,
		`[{"f\u0000oo":"bar"}]`,
		`[{"a":"A","b":"B","c":"C\u0000"}]`,
		"{\"a\":\"b\x00c\"}",
	}

	for _, c := range cases {
		sanitized := SanitizeJSONB([]byte(c))

		if err := ValidateJSONB(sanitized, "field"); err != nil {
			t.Fatalf("expected sanitized %q to validate, got %v", string(sanitized), err)
		}
	}
}

func TestSanitizeJSONB_DoesNotCopyCleanPayloads(t *testing.T) {
	in := []byte(`{"a":"b"}`)
	out := SanitizeJSONB(in)

	if &in[0] != &out[0] {
		t.Fatalf("expected a clean payload to be returned as-is, got a copy")
	}
}
