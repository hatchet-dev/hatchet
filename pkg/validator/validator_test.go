//go:build !e2e && !load && !rampup && !integration

package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type nameResource struct {
	DisplayName string `validate:"hatchetName"`
}

func TestValidatorName(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErrTag string
	}{
		{
			name:  "valid name",
			input: "test-name",
		},
		{
			name:       "invalid name with special characters",
			input:      "&&!!",
			wantErrTag: "hatchetName",
		},
	}

	v := newValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(&nameResource{DisplayName: tt.input})
			if tt.wantErrTag != "" {
				assert.ErrorContains(t, err, "validation for 'DisplayName' failed on the '"+tt.wantErrTag+"' tag")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type cronResource struct {
	Cron string `validate:"cron"`
}

func TestValidatorCron(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErrTag string
	}{
		{
			name:  "valid cron expression",
			input: "*/5 * * * *",
		},
		{
			name:       "invalid cron expression (missing field)",
			input:      "*/5 * * *",
			wantErrTag: "cron",
		},
	}

	v := newValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(&cronResource{Cron: tt.input})
			if tt.wantErrTag != "" {
				assert.ErrorContains(t, err, "validation for 'Cron' failed on the '"+tt.wantErrTag+"' tag")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type durationResource struct {
	Duration string `validate:"duration"`
}

func TestValidatorDuration(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErrTag string
	}{
		{
			name:  "valid duration",
			input: "5s",
		},
		{
			name:  "minutes and seconds",
			input: "42m30s",
		},
		{
			name:  "hours minutes seconds",
			input: "1h30m5s",
		},
		{
			name:  "decimal hour",
			input: "1.5h",
		},
		{
			name:  "milliseconds",
			input: "1500ms",
		},
		{
			name:  "zero with unit",
			input: "0s",
		},
		{
			name:       "invalid duration (missing unit)",
			input:      "5",
			wantErrTag: "duration",
		},
		{
			name:       "invalid duration (bare zero)",
			input:      "0",
			wantErrTag: "duration",
		},
		{
			name:       "invalid duration (trailing garbage)",
			input:      "42m30sX",
			wantErrTag: "duration",
		},
		{
			name:       "invalid duration (negative)",
			input:      "-1.5h",
			wantErrTag: "duration",
		},
		{
			name:       "invalid duration (leading plus sign)",
			input:      "+1h",
			wantErrTag: "duration",
		},
		{
			name:       "invalid duration (nanoseconds)",
			input:      "100ns",
			wantErrTag: "duration",
		},
		{
			name:       "invalid duration (microseconds)",
			input:      "100us",
			wantErrTag: "duration",
		},
		{
			name:       "invalid duration (days)",
			input:      "1d",
			wantErrTag: "duration",
		},
		{
			name:       "invalid duration (overflow)",
			input:      "9999999999999h",
			wantErrTag: "duration",
		},
	}

	v := newValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(&durationResource{Duration: tt.input})
			if tt.wantErrTag != "" {
				assert.ErrorContains(t, err, "validation for 'Duration' failed on the '"+tt.wantErrTag+"' tag")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type passwordResource struct {
	Password string `validate:"password"`
}

func TestValidatorPassword(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErrTag string
	}{
		{
			name:  "valid password",
			input: "ValidPass1",
		},
		{
			name:       "too short (under 8 characters)",
			input:      "Short1",
			wantErrTag: "password",
		},
		{
			name:  "valid password between 32 and 64 characters",
			input: "ThisPasswordIsLongerThan32CharsButValid1A123456789012",
		},
		{
			name:  "valid password at exactly 64 characters",
			input: "ValidPassword1AValidPassword1AValidPassword1AValidPassword1A1234",
		},
		{
			name:       "too long (over 64 characters)",
			input:      "ValidPassword1AValidPassword1AValidPassword1AValidPassword1A12345",
			wantErrTag: "password",
		},
	}

	v := newValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(&passwordResource{Password: tt.input})
			if tt.wantErrTag != "" {
				assert.ErrorContains(t, err, "validation for 'Password' failed on the '"+tt.wantErrTag+"' tag")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
