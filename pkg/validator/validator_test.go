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

func TestValidatorValidCronWithSeconds(t *testing.T) {
	v := newValidator()

	// Test 6-field cron expressions (with seconds)
	testCases := []string{
		"*/30 * * * * *",     // Every 30 seconds
		"0 */2 * * * *",      // Every 2 minutes
		"15 30 10 * * *",     // Every day at 10:30:15
		"0 0 0 1 * *",        // First day of every month at midnight
		"*/15 * * * * MON-FRI", // Every 15 seconds on weekdays
	}

	for _, cronExpr := range testCases {
		err := v.Struct(&cronResource{
			Cron: cronExpr,
		})
		assert.NoError(t, err, "should accept valid 6-field cron: %s", cronExpr)
	}
}

func TestValidatorInvalidCronWithSeconds(t *testing.T) {
	v := newValidator()

	// Test invalid cron expressions (both 5 and 6-field)
	testCases := []string{
		"* * * *",            // Too few fields
		"* * * * * * *",      // Too many fields
		"invalid * * * *",    // Invalid field value
		"60 * * * * *",       // Invalid seconds (60)
		"* 60 * * * *",       // Invalid minutes (60)
		"* * 24 * * *",       // Invalid hours (24)
	}

	for _, cronExpr := range testCases {
		err := v.Struct(&cronResource{
			Cron: cronExpr,
		})
		assert.ErrorContains(t, err, "validation for 'Cron' failed on the 'cron' tag", "should reject invalid cron: %s", cronExpr)
	}
}

func TestCronHasSecondsDetection(t *testing.T) {
	testCases := []struct {
		expr        string
		hasSeconds  bool
		description string
	}{
		{"0 * * * *", false, "5-field standard cron"},
		{"*/5 * * * *", false, "5-field with wildcard"},
		{"0 0 * * *", false, "5-field daily"},
		{"*/30 * * * * *", true, "6-field with seconds"},
		{"0 */2 * * * *", true, "6-field every 2 minutes"},
		{"15 30 10 * * *", true, "6-field specific time"},
		{"0 0 0 1 * *", true, "6-field monthly"},
		// Test timezone prefixes
		{"CRON_TZ=America/New_York 0 * * * *", false, "5-field with CRON_TZ"},
		{"TZ=UTC */30 * * * * *", true, "6-field with TZ"},
		{"CRON_TZ=Europe/London 15 30 10 * * *", true, "6-field with CRON_TZ"},
	}

	for _, tc := range testCases {
		result := CronHasSeconds(tc.expr)
		assert.Equal(t, tc.hasSeconds, result, "Detection failed for %s: %s", tc.expr, tc.description)
	}
}

func TestValidatorValidCronWithTimezone(t *testing.T) {
	v := newValidator()

	// Test cron expressions with timezone prefixes
	testCases := []string{
		"CRON_TZ=America/New_York 0 * * * *",      // 5-field with CRON_TZ
		"TZ=UTC */15 * * * *",                     // 5-field with TZ
		"CRON_TZ=Europe/London */30 * * * * *",    // 6-field with CRON_TZ
		"TZ=Asia/Tokyo 0 */2 * * * *",             // 6-field with TZ
		"CRON_TZ=America/Los_Angeles 15 30 10 * * *", // 6-field specific time with timezone
	}

	for _, cronExpr := range testCases {
		err := v.Struct(&cronResource{
			Cron: cronExpr,
		})
		assert.NoError(t, err, "should accept valid cron with timezone: %s", cronExpr)
	}
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
