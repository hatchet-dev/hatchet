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

type passwordResource struct {
	Password string `validate:"password"`
}

func TestValidatorValidPassword(t *testing.T) {
	v := newValidator()

	err := v.Struct(&passwordResource{
		Password: "ValidPass1",
	})

	assert.NoError(t, err, "should accept a valid password")
}

func TestValidatorPasswordTooShort(t *testing.T) {
	v := newValidator()

	err := v.Struct(&passwordResource{
		Password: "Short1",
	})

	assert.ErrorContains(t, err, "validation for 'Password' failed on the 'password' tag", "should reject password shorter than 8 characters")
}

func TestValidatorPasswordBetween32And64(t *testing.T) {
	v := newValidator()

	// 50-character password: valid (between 32 and 64)
	err := v.Struct(&passwordResource{
		Password: "ThisPasswordIsLongerThan32CharsButValid1A123456789012",
	})

	assert.NoError(t, err, "should accept passwords between 32 and 64 characters")
}

func TestValidatorPasswordExactly64(t *testing.T) {
	v := newValidator()

	// exactly 64 characters
	err := v.Struct(&passwordResource{
		Password: "ValidPassword1AValidPassword1AValidPassword1AValidPassword1A1234",
	})

	assert.NoError(t, err, "should accept passwords of exactly 64 characters")
}

func TestValidatorPasswordLongerThan64(t *testing.T) {
	v := newValidator()

	// 65 characters
	err := v.Struct(&passwordResource{
		Password: "ValidPassword1AValidPassword1AValidPassword1AValidPassword1A12345",
	})

	assert.ErrorContains(t, err, "validation for 'Password' failed on the 'password' tag", "should reject passwords longer than 64 characters")
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
