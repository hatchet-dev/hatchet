//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateJSONBAgreesWithPostgres(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	cases := []struct {
		name string
		json string
	}{
		{name: "object", json: `{"a":1}`},
		{name: "string", json: `"a string is valid json"`},
		{name: "number", json: `123`},
		{name: "boolean", json: `true`},
		{name: "null", json: `null`},
		{name: "empty array", json: `[]`},
		{name: "non surrogate escapes", json: `{"label":"\u00e9\n\"\\"}`},
		{name: "duplicate keys", json: `{"a":1,"a":2}`},

		{name: "unterminated object", json: `{`},
		{name: "trailing comma", json: `[1,]`},
		{name: "single quoted string", json: `{'a':1}`},
		{name: "bare word", json: `nope`},

		{name: "null escape in value", json: `{"a":"\u0000"}`},
		{name: "null escape in key", json: `{"f\u0000oo":"bar"}`},
		{name: "null escape in nested value", json: `[{"a":"A","b":"B","c":"C\u0000"}]`},
		{name: "escaped backslash then null escape", json: `{"a":"\\\u0000"}`},
		{name: "escaped backslash before u0000", json: `{"a":"\\u0000"}`},
		{name: "two escaped backslashes before u0000", json: `{"a":"\\\\u0000"}`},

		{name: "surrogate pair", json: `{"label":"a\ud83d\ude00b"}`},
		{name: "surrogate pair upper case", json: `["\uD83D\uDE00"]`},
		{name: "surrogate pair in key", json: `{"\ud83d\ude00":1}`},
		{name: "two surrogate pairs", json: `"\ud83d\ude00\ud83d\ude00"`},
		{name: "first high then last low", json: `"\ud800\udfff"`},
		{name: "last high then first low", json: `"\udbff\udc00"`},
		{name: "last code point below the surrogates", json: `"\ud7ff"`},
		{name: "first code point above the surrogates", json: `"\ue000"`},
		{name: "escaped backslash before ud83d", json: `{"label":"\\ud83d"}`},

		{name: "lone high", json: `{"label":"a\ud83d..."}`},
		{name: "lone low", json: `{"label":"a\ude00b"}`},
		{name: "lone first high", json: `"\ud800"`},
		{name: "lone last high", json: `"\udbff"`},
		{name: "lone first low", json: `"\udc00"`},
		{name: "lone last low", json: `"\udfff"`},
		{name: "high at end of string", json: `"\ud83d"`},
		{name: "high at end of key", json: `{"\ud83d":1}`},
		{name: "high before short escape", json: `"\ud83d\n"`},
		{name: "high before code point below the surrogates", json: `"\ud83d\ud7ff"`},
		{name: "high before code point above the surrogates", json: `"\ud83d\ue000"`},
		{name: "two highs then low", json: `"\ud83d\ud83d\ude00"`},
		{name: "pair then lone low", json: `"\ud83d\ude00\ude00"`},
		{name: "reversed pair", json: `"\ude00\ud83d"`},
		{name: "two first lows", json: `"\udc00\udc00"`},
		{name: "high before text that reads as a low", json: `"\ud83d..de00"`},
		{name: "high then escaped backslash then low", json: `"\ud83d\\\ude00"`},
		{name: "escaped backslash then lone high", json: `{"label":"\\\ud83d"}`},
		{name: "high and low in adjacent strings", json: `["\ud83d","\ude00"]`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stored string
			postgresErr := pool.QueryRow(context.Background(), `SELECT ($1::text)::jsonb::text`, c.json).Scan(&stored)
			validateErr := ValidateJSONB([]byte(c.json), "field")

			require.Equal(t, postgresErr == nil, validateErr == nil, "%s: postgres said %v, ValidateJSONB said %v", c.json, postgresErr, validateErr)
		})
	}
}
