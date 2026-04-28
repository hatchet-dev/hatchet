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
			name:       "invalid duration (missing unit)",
			input:      "5",
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
