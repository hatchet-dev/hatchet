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
		if isValid := IsUnicodeValid(c); isValid {
			t.Fatalf("expected invalid unicode for json %q, got valid", string(c))
		}
	}
}

func TestValidateJSONB_UnpairedSurrogateEscapes(t *testing.T) {
	for _, c := range surrogateEscapeCases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateJSONB([]byte(c.json), "field")

			if c.unpaired && err == nil {
				t.Fatalf("expected an error for %s, got nil", c.json)
			}

			if !c.unpaired && err != nil {
				t.Fatalf("expected no error for %s, got %v", c.json, err)
			}
		})
	}
}

var surrogateEscapeCases = []struct {
	name     string
	json     string
	unpaired bool
}{
	{name: "lone high", json: `{"label":"a\ud83d..."}`, unpaired: true},
	{name: "lone low", json: `{"label":"a\ude00b"}`, unpaired: true},
	{name: "high at end of string", json: `"\ud83d"`, unpaired: true},
	{name: "high before non surrogate escape", json: `"\ud83d\u0041"`, unpaired: true},
	{name: "two highs then low", json: `"\ud83d\ud83d\ude00"`, unpaired: true},
	{name: "reversed pair", json: `"\ude00\ud83d"`, unpaired: true},
	{name: "high then escaped backslash then low", json: `"\ud83d\\\ude00"`, unpaired: true},
	{name: "escaped backslash then lone high", json: `{"label":"\\\ud83d"}`, unpaired: true},
	{name: "lone high in key", json: `{"\ud83d":1}`, unpaired: true},
	{name: "valid pair", json: `{"label":"a\ud83d\ude00b"}`},
	{name: "valid pair upper case", json: `["\uD83D\uDE00"]`},
	{name: "escaped backslash before u", json: `{"label":"\\ud83d"}`},
	{name: "non surrogate escapes", json: `{"label":"\u00e9\n\"\\"}`},
	{name: "no escapes", json: `{"label":"plain text"}`},
}
