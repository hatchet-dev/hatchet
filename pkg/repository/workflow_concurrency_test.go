//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/validator"
)

func TestCreateConcurrencyOpts_MaxRunsValidation(t *testing.T) {
	v := validator.NewDefaultValidator()

	tests := []struct {
		name    string
		maxRuns *int32
		wantErr bool
	}{
		{name: "nil max runs is allowed", maxRuns: nil, wantErr: false},
		{name: "positive max runs is allowed", maxRuns: int32Ptr(5), wantErr: false},
		{name: "zero max runs is rejected", maxRuns: int32Ptr(0), wantErr: true},
		{name: "negative max runs is rejected", maxRuns: int32Ptr(-1), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := CreateConcurrencyOpts{
				MaxRuns:    tt.maxRuns,
				Expression: "'constant'",
			}

			apiErrors, err := v.ValidateAPI(opts)
			assert.NoError(t, err)

			if tt.wantErr {
				assert.NotNil(t, apiErrors)
			} else {
				assert.Nil(t, apiErrors)
			}
		})
	}
}
