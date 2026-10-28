package serverlessoperator

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckRuntimeSupportRefusesFIPSOnlyMode(t *testing.T) {
	err := CheckRuntimeSupport()

	if strings.Contains(os.Getenv("GODEBUG"), "fips140=only") {
		assert.ErrorIs(t, err, ErrFIPSOnlyMode)
	} else {
		assert.NoError(t, err)
	}
}
