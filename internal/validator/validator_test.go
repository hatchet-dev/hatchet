package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type nameResource struct {
	DisplayName string `form:"hatchet-name"`
}

func TestValidatorInvalidName(t *testing.T) {
	v := newValidator()

	err := v.Struct(&nameResource{
		DisplayName: "&&!!",
	})

	assert.ErrorContains(t, err, "validation for 'DisplayName' failed on the 'hatchet-name' tag", "should throw error on invalid name")
}

func TestValidatorValidName(t *testing.T) {
	v := newValidator()

	err := v.Struct(&nameResource{
		DisplayName: "test-name",
	})

	assert.NoError(t, err, "no error")
}

type cronResource struct {
	Cron string `form:"cron"`
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
