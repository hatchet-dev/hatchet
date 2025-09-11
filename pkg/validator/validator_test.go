//go:build !e2e && !load && !rampup && !integration

package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type nameResource struct {
	DisplayName string `validate:"hatchetName"`
}

func TestValidatorInvalidName(t *testing.T) {
	v := newValidator()

	err := v.Struct(&nameResource{
		DisplayName: "&&!!",
	})

	assert.ErrorContains(t, err, "validation for 'DisplayName' failed on the 'hatchetName' tag", "should throw error on invalid name")
}

func TestValidatorValidName(t *testing.T) {
	v := newValidator()

	err := v.Struct(&nameResource{
		DisplayName: "test-name",
	})

	assert.NoError(t, err, "no error")
}

type cronResource struct {
	Cron string `validate:"cron"`
}

func TestValidatorValidCron(t *testing.T) {
	v := newValidator()

	err := v.Struct(&cronResource{
		Cron: "*/5 * * * *",
	})

	assert.NoError(t, err, "no error")
}

func TestValidatorInvalidCron(t *testing.T) {
	v := newValidator()

	err := v.Struct(&cronResource{
		Cron: "*/5 * * *",
	})

	assert.ErrorContains(t, err, "validation for 'Cron' failed on the 'cron' tag", "should throw error on invalid cron")
}

func TestValidatorValidDuration(t *testing.T) {
	v := newValidator()

	err := v.Struct(&struct {
		Duration string `validate:"duration"`
	}{
		Duration: "5s",
	})

	assert.NoError(t, err, "no error")
}

func TestValidatorInvalidDuration(t *testing.T) {
	v := newValidator()

	err := v.Struct(&struct {
		Duration string `validate:"duration"`
	}{
		Duration: "5",
	})

	assert.ErrorContains(t, err, "validation for 'Duration' failed on the 'duration' tag", "should throw error on invalid duration")
}
