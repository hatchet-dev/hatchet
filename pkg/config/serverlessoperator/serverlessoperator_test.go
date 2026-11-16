package serverlessoperator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator"
)

func validConfig() ConfigFile {
	return ConfigFile{
		DatabaseUrl:            "postgres://localhost/hatchet",
		TokenExchange:          TokenExchangeStatic,
		ClientToken:            "token",
		EndpointPartitionCount: 1,
	}
}

// The token exchange mode is explicit and each mode's variable pair is mutually exclusive:
// a mode without its own variable, or with the other mode's, is refused at startup.
func TestValidateTokenExchange(t *testing.T) {
	cases := []struct {
		name        string
		mutate      func(cf *ConfigFile)
		wantErrPart string
	}{
		{name: "static with a client token", mutate: func(cf *ConfigFile) {}},
		{name: "local with a token file", mutate: func(cf *ConfigFile) {
			cf.TokenExchange = TokenExchangeLocal
			cf.ClientToken = ""
			cf.TokenFile = "/run/secrets/tokens.yaml"
		}},
		{name: "mode is case insensitive and trimmed", mutate: func(cf *ConfigFile) {
			cf.TokenExchange = " Static "
		}},
		{name: "no mode", mutate: func(cf *ConfigFile) {
			cf.TokenExchange = ""
		}, wantErrPart: "SERVERLESS_OPERATOR_TOKEN_EXCHANGE is required"},
		{name: "unknown mode", mutate: func(cf *ConfigFile) {
			cf.TokenExchange = "vault"
		}, wantErrPart: `unknown SERVERLESS_OPERATOR_TOKEN_EXCHANGE "vault"`},
		{name: "static without a client token", mutate: func(cf *ConfigFile) {
			cf.ClientToken = ""
		}, wantErrPart: "HATCHET_CLIENT_TOKEN is required"},
		{name: "static with a token file set", mutate: func(cf *ConfigFile) {
			cf.TokenFile = "/run/secrets/tokens.yaml"
		}, wantErrPart: "SERVERLESS_OPERATOR_TOKEN_FILE must not be set"},
		{name: "local without a token file", mutate: func(cf *ConfigFile) {
			cf.TokenExchange = TokenExchangeLocal
			cf.ClientToken = ""
		}, wantErrPart: "SERVERLESS_OPERATOR_TOKEN_FILE is required"},
		{name: "local with a client token set", mutate: func(cf *ConfigFile) {
			cf.TokenExchange = TokenExchangeLocal
			cf.TokenFile = "/run/secrets/tokens.yaml"
		}, wantErrPart: "HATCHET_CLIENT_TOKEN must not be set"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cf := validConfig()
			tc.mutate(&cf)

			err := cf.Validate()

			if tc.wantErrPart == "" {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErrPart)
		})
	}
}

func TestValidateRequiresDatabaseAndBoundsEndpointPartitionCount(t *testing.T) {
	cf := validConfig()
	cf.DatabaseUrl = ""
	require.ErrorContains(t, cf.Validate(), "DATABASE_URL")

	cf = validConfig()
	cf.EndpointPartitionCount = serverlessoperator.MaxEndpointPartitionCount + 1
	require.ErrorContains(t, cf.Validate(), "partition count")

	cf = validConfig()
	cf.EndpointPartitionCount = serverlessoperator.MaxEndpointPartitionCount
	assert.NoError(t, cf.Validate())
}

func TestCoreConfigCarriesTheFile(t *testing.T) {
	cf := validConfig()
	cf.OperatorName = "custom"
	cf.EndpointPartitionCount = 4
	cf.HealthPort = 9090
	cf.InfraBlockedCIDRs = " 10.0.0.0/8, ,192.168.0.0/16 "

	core := cf.CoreConfig()

	assert.Equal(t, "custom", core.OperatorName)
	assert.Equal(t, serverlessoperator.DefaultLinkName, core.LinkName)
	assert.Equal(t, int32(4), core.EndpointPartitionCount)
	assert.Equal(t, 9090, core.HealthPort)
	assert.Equal(t, []string{"10.0.0.0/8", "192.168.0.0/16"}, cf.InfraBlockedCIDRList())
}
