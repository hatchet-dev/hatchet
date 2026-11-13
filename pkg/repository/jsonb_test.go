package repository

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateJSONB_InvalidJSON(t *testing.T) {
	if _, err := ValidateJSONB([]byte("{"), "field"); err == nil {
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
		if _, err := ValidateJSONB(c, "field"); err != nil {
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

func TestValidateJSONB_ReplacesUnpairedSurrogateEscapes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "lone high", in: `{"label":"a\ud83d..."}`, want: `{"label":"a\ufffd..."}`},
		{name: "lone low", in: `{"label":"a\ude00b"}`, want: `{"label":"a\ufffdb"}`},
		{name: "valid pair", in: `{"label":"a\ud83d\ude00b"}`, want: `{"label":"a\ud83d\ude00b"}`},
		{name: "valid pair upper case", in: `["\uD83D\uDE00"]`, want: `["\uD83D\uDE00"]`},
		{name: "escaped backslash before u", in: `{"label":"\\ud83d"}`, want: `{"label":"\\ud83d"}`},
		{name: "escaped backslash then lone high", in: `{"label":"\\\ud83d"}`, want: `{"label":"\\\ufffd"}`},
		{name: "high at end of string", in: `"\ud83d"`, want: `"\ufffd"`},
		{name: "high before non surrogate escape", in: `"\ud83d\u0041"`, want: `"\ufffd\u0041"`},
		{name: "two highs then low", in: `"\ud83d\ud83d\ude00"`, want: `"\ufffd\ud83d\ude00"`},
		{name: "reversed pair", in: `"\ude00\ud83d"`, want: `"\ufffd\ufffd"`},
		{name: "lone high in key", in: `{"\ud83d":1}`, want: `{"\ufffd":1}`},
		{name: "no escapes", in: `{"label":"plain text"}`, want: `{"label":"plain text"}`},
		{name: "non surrogate escapes", in: `{"label":"\u00e9\n\"\\"}`, want: `{"label":"\u00e9\n\"\\"}`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ValidateJSONB(c.in, "field")
			require.NoError(t, err)
			require.Equal(t, c.want, got)

			gotBytes, err := ValidateJSONB([]byte(c.in), "field")
			require.NoError(t, err)
			require.Equal(t, c.want, string(gotBytes))

			var fromInput, fromOutput any
			require.NoError(t, json.Unmarshal([]byte(c.in), &fromInput))
			require.NoError(t, json.Unmarshal([]byte(got), &fromOutput))
			require.Equal(t, fromInput, fromOutput, "the output must decode to what Go decodes the input to")
		})
	}
}

func TestValidateJSONB_ReturnsInputWhenNothingToReplace(t *testing.T) {
	for _, in := range [][]byte{
		[]byte(`{"label":"plain text"}`),
		[]byte(`{"label":"a\u00e9\ud83d\ude00"}`),
	} {
		got, err := ValidateJSONB(in, "field")
		require.NoError(t, err)
		require.Same(t, &in[0], &got[0])
	}
}

func TestValidateJSONB_DoesNotModifyInput(t *testing.T) {
	in := []byte(`{"label":"a\ud83d"}`)

	got, err := ValidateJSONB(in, "field")
	require.NoError(t, err)
	require.Equal(t, `{"label":"a\ufffd"}`, string(got))
	require.Equal(t, `{"label":"a\ud83d"}`, string(in))
}
