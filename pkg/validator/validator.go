package validator

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/Masterminds/semver/v3"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

var NameRegex = regexp.MustCompile("^[a-zA-Z0-9\\.\\-_]+$") //nolint:gosimple

func newValidator() *validator.Validate {
	validate := validator.New()

	celParser := cel.NewCELParser()
	cronParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	_ = validate.RegisterValidation("hatchetName", func(fl validator.FieldLevel) bool {
		return NameRegex.MatchString(fl.Field().String())
	})

	_ = validate.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		return passwordValidation(fl.Field().String())
	})

	_ = validate.RegisterValidation("uuid", func(fl validator.FieldLevel) bool {
		return IsValidUUID(fl.Field().String())
	})

	_ = validate.RegisterValidation("cron", func(fl validator.FieldLevel) bool {
		var err error
		cronExpr := fl.Field().String()

		// Extract just the cron part for field count validation
		cronPart := cronExpr
		if strings.HasPrefix(cronExpr, "CRON_TZ=") {
			parts := strings.SplitN(cronExpr, " ", 2)
			if len(parts) == 2 {
				cronPart = parts[1]
			}
		} else if strings.HasPrefix(cronExpr, "TZ=") {
			parts := strings.SplitN(cronExpr, " ", 2)
			if len(parts) == 2 {
				cronPart = parts[1]
			}
		}

		// Validate field count first - only allow 5 or 6 fields
		fields := strings.Fields(cronPart)
		if len(fields) != 5 && len(fields) != 6 {
			return false
		}

		// Parse the cron part only - robfig/cron doesn't support timezone prefixes
		if len(fields) == 6 {
			// uses same logic as gocron internally does
			p := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
			_, err = p.Parse(cronPart)
		} else {
			_, err = cronParser.Parse(cronPart)
		}

		return err == nil
	})

	_ = validate.RegisterValidation("actionId", func(fl validator.FieldLevel) bool {
		action, err := types.ParseActionID(fl.Field().String())

		if err != nil {
			return false
		}

		return action.Service != "" && action.Verb != ""
	})

	_ = validate.RegisterValidation("semver", func(fl validator.FieldLevel) bool {
		_, err := semver.NewVersion(fl.Field().String())

		return err == nil
	})

	_ = validate.RegisterValidation("json", func(fl validator.FieldLevel) bool {
		return isValidJSON(fl.Field().String())
	})

	_ = validate.RegisterValidation("duration", func(fl validator.FieldLevel) bool {
		_, err := time.ParseDuration(fl.Field().String())

		return err == nil
	})

	_ = validate.RegisterValidation("celworkflowrunstr", func(fl validator.FieldLevel) bool {
		_, err := celParser.ParseWorkflowString(fl.Field().String())

		return err == nil
	})

	_ = validate.RegisterValidation("celsteprunstr", func(fl validator.FieldLevel) bool {
		_, err := celParser.ParseStepRun(fl.Field().String())

		return err == nil
	})

	_ = validate.RegisterValidation("future", func(fl validator.FieldLevel) bool {
		if t, ok := fl.Field().Interface().(time.Time); ok {
			return t.After(time.Now())
		}
		return false
	})

	return validate
}

func passwordValidation(pw string) bool {
	pwLen := len(pw)
	var hasNumber, hasUpper, hasLower bool

	for _, char := range pw {
		switch {
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		}
	}

	return hasNumber && hasUpper && hasLower && pwLen >= 8 && pwLen <= 32
}

func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func isValidJSON(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

// CronHasSeconds checks if a cron expression includes seconds (6 fields) or not (5 fields)
// Also validates that the expression has exactly 5 or 6 fields (rejects 7+ field expressions)
// Supports timezone prefixes like CRON_TZ= and TZ=
func CronHasSeconds(cronExpr string) bool {
	// Extract just the cron part for field count analysis
	cronPart := cronExpr
	if strings.HasPrefix(cronExpr, "CRON_TZ=") {
		parts := strings.SplitN(cronExpr, " ", 2)
		if len(parts) == 2 {
			cronPart = parts[1]
		}
	} else if strings.HasPrefix(cronExpr, "TZ=") {
		parts := strings.SplitN(cronExpr, " ", 2)
		if len(parts) == 2 {
			cronPart = parts[1]
		}
	}

	fields := strings.Fields(cronPart)
	return len(fields) == 6
}
