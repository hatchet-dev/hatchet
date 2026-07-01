package client

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindAllEnvNoRetry(t *testing.T) {
	t.Setenv("HATCHET_CLIENT_NO_RETRY", "true")
	t.Setenv("HATCHET_CLIENT_NO_GRPC_RETRY", "true")

	v := viper.New()
	BindAllEnv(v)

	assert.Equal(t, "true", v.GetString("noRetry"))
	assert.Equal(t, "true", v.GetString("noGrpcRetry"))
}

func TestClientConfigFileNoRetryYAML(t *testing.T) {
	t.Parallel()

	v := viper.New()
	BindAllEnv(v)
	require.NoError(t, v.MergeConfigMap(map[string]interface{}{
		"noRetry":     true,
		"noGrpcRetry": true,
	}))

	cf := &ClientConfigFile{}
	require.NoError(t, v.Unmarshal(cf))

	assert.True(t, cf.NoRetry)
	assert.True(t, cf.NoGrpcRetry)
}

func TestRetryControlSemantics(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		noRetry     bool
		noGrpcRetry bool
		grpcRetry   bool
		restRetry   bool
	}{
		{name: "defaults", grpcRetry: true, restRetry: true},
		{name: "no grpc only", noGrpcRetry: true, grpcRetry: false, restRetry: true},
		{name: "no retry all", noRetry: true, grpcRetry: false, restRetry: false},
		{name: "both set", noRetry: true, noGrpcRetry: true, grpcRetry: false, restRetry: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cf := &ClientConfigFile{NoRetry: tc.noRetry, NoGrpcRetry: tc.noGrpcRetry}
			grpcRetry := !cf.NoRetry && !cf.NoGrpcRetry
			restRetry := !cf.NoRetry

			assert.Equal(t, tc.grpcRetry, grpcRetry)
			assert.Equal(t, tc.restRetry, restRetry)
		})
	}
}
