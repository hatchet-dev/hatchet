package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/cel"
)

func TestResolveDisplayName(t *testing.T) {
	parser := cel.NewCELParser()
	logger := zerolog.Nop()
	runID := uuid.New()
	const fallback = "step1-1700000000000"

	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name     string
		expr     *string
		runInput map[string]interface{}
		meta     map[string]interface{}
		want     string
	}{
		{
			name:     "nil expression falls back",
			expr:     nil,
			runInput: map[string]interface{}{"name": "Acme"},
			want:     fallback,
		},
		{
			name:     "empty expression falls back",
			expr:     strPtr("   "),
			runInput: map[string]interface{}{"name": "Acme"},
			want:     fallback,
		},
		{
			name:     "reads a top-level input key",
			expr:     strPtr("input.name"),
			runInput: map[string]interface{}{"name": "Acme Corp"},
			want:     "Acme Corp",
		},
		{
			name:     "quoted string literal",
			expr:     strPtr("'hello'"),
			runInput: map[string]interface{}{},
			want:     "hello",
		},
		{
			name:     "missing key falls back",
			expr:     strPtr("input.missing"),
			runInput: map[string]interface{}{"name": "Acme"},
			want:     fallback,
		},
		{
			name:     "has() guard on missing key falls back to CEL literal",
			expr:     strPtr("has(input.name) ? input.name : 'unnamed'"),
			runInput: map[string]interface{}{},
			want:     "unnamed",
		},
		{
			name:     "non-string result falls back",
			expr:     strPtr("input.count"),
			runInput: map[string]interface{}{"count": 42},
			want:     fallback,
		},
		{
			name:     "empty-string result falls back",
			expr:     strPtr("''"),
			runInput: map[string]interface{}{},
			want:     fallback,
		},
		{
			name:     "reads additional_metadata",
			expr:     strPtr("additional_metadata.tenant"),
			runInput: map[string]interface{}{},
			meta:     map[string]interface{}{"tenant": "t-123"},
			want:     "t-123",
		},
		{
			name:     "result is trimmed",
			expr:     strPtr("'  padded  '"),
			runInput: map[string]interface{}{},
			want:     "padded",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveDisplayName(parser, &logger, tc.expr, tc.runInput, tc.meta, runID, fallback)
			if got != tc.want {
				t.Errorf("resolveDisplayName() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveDisplayName_TruncatesTo255Runes(t *testing.T) {
	parser := cel.NewCELParser()
	logger := zerolog.Nop()

	long := make([]rune, 300)
	for i := range long {
		long[i] = 'a'
	}
	expr := "input.name"
	got := resolveDisplayName(parser, &logger, &expr, map[string]interface{}{"name": string(long)}, nil, uuid.New(), "fallback")

	if len([]rune(got)) != maxDisplayNameRunes {
		t.Errorf("expected result truncated to %d runes, got %d", maxDisplayNameRunes, len([]rune(got)))
	}
}
