//go:build !e2e && !load && !rampup && !integration

package repository

import "testing"

func TestVerifiesEmail(t *testing.T) {
	tests := []struct {
		name     string
		existing bool
		update   *bool
		want     bool
	}{
		{name: "unverified user verified by login", existing: false, update: BoolPtr(true), want: true},
		{name: "unverified user, login does not verify", existing: false, update: BoolPtr(false), want: false},
		{name: "unverified user, no verification in update", existing: false, update: nil, want: false},
		{name: "already verified user", existing: true, update: BoolPtr(true), want: false},
		{name: "verified user, login does not verify", existing: true, update: BoolPtr(false), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := verifiesEmail(tt.existing, &UpdateUserOpts{EmailVerified: tt.update})
			if got != tt.want {
				t.Fatalf("verifiesEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}
