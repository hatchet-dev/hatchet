package cli

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveProfileWithoutForm(t *testing.T) {
	tests := []struct {
		name           string
		names          []string
		defaultProfile string
		useDefault     bool
		wantName       string
		wantOK         bool
	}{
		{
			name:           "default set and valid",
			names:          []string{"local", "prod"},
			defaultProfile: "prod",
			useDefault:     true,
			wantName:       "prod",
			wantOK:         true,
		},
		{
			name:           "single profile, no default",
			names:          []string{"local"},
			defaultProfile: "",
			useDefault:     true,
			wantName:       "local",
			wantOK:         true,
		},
		{
			name:           "single profile, stale default",
			names:          []string{"local"},
			defaultProfile: "removed-profile",
			useDefault:     true,
			wantName:       "local",
			wantOK:         true,
		},
		{
			name:           "multiple profiles, no default",
			names:          []string{"local", "prod"},
			defaultProfile: "",
			useDefault:     true,
			wantName:       "",
			wantOK:         false,
		},
		{
			name:           "single profile but management command",
			names:          []string{"local"},
			defaultProfile: "",
			useDefault:     false,
			wantName:       "",
			wantOK:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, ok := resolveProfileWithoutForm(tt.names, tt.defaultProfile, tt.useDefault)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

// The fixture mirrors huh's no-TTY error. The message once misreported this failure as a
// "profile remove form" error, hence the exclusion.
func TestProfileSelectionFailureMessage(t *testing.T) {
	formErr := errors.New("huh: could not open a new TTY: open /dev/tty: no such device or address")

	msg := profileSelectionFailureMessage(formErr, []string{"local", "prod"})

	assert.Contains(t, msg, "profile selection form")
	assert.Contains(t, msg, "--profile")
	assert.Contains(t, msg, "hatchet profile set-default")
	assert.Contains(t, msg, "local, prod")
	assert.Contains(t, msg, "/dev/tty")
	assert.NotContains(t, msg, "remove")
}
