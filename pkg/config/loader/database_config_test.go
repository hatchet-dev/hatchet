package loader

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDatabaseConfigFile_OLAPDefaultsToUnset(t *testing.T) {
	cf, err := LoadDatabaseConfigFile()
	require.NoError(t, err)

	assert.Empty(t, cf.OLAP.DatabaseURL)
	assert.Zero(t, cf.OLAP.MaxConns)
	assert.Zero(t, cf.OLAP.FairpoolTenantMaxPercent)
	assert.Zero(t, cf.OLAP.ConnectTimeout)
}

func TestLoadDatabaseConfigFile_OLAPEnv(t *testing.T) {
	t.Setenv("OLAP_DATABASE_URL", "postgresql://hatchet:hatchet@olap:5432/hatchet_olap")
	t.Setenv("OLAP_DATABASE_MAX_CONNS", "12")
	t.Setenv("OLAP_DATABASE_MIN_CONNS", "3")
	t.Setenv("OLAP_DATABASE_MAX_CONN_LIFETIME", "7m")
	t.Setenv("OLAP_DATABASE_MAX_CONN_IDLE_TIME", "90s")
	t.Setenv("OLAP_DATABASE_DIAL_TIMEOUT", "4s")
	t.Setenv("OLAP_DATABASE_CONNECT_TIMEOUT", "20s")
	t.Setenv("OLAP_DATABASE_FAIRPOOL_TENANT_MAX_PERCENT", "40")
	t.Setenv("OLAP_DATABASE_FAIRPOOL_SHARED_MAX_PERCENT", "60")
	t.Setenv("OLAP_DATABASE_FAIRPOOL_TENANT_MAX_WAIT", "2s")

	cf, err := LoadDatabaseConfigFile()
	require.NoError(t, err)

	assert.Equal(t, "postgresql://hatchet:hatchet@olap:5432/hatchet_olap", cf.OLAP.DatabaseURL)
	assert.Equal(t, 12, cf.OLAP.MaxConns)
	assert.Equal(t, 3, cf.OLAP.MinConns)
	assert.Equal(t, 7*time.Minute, cf.OLAP.MaxConnLifetime)
	assert.Equal(t, 90*time.Second, cf.OLAP.MaxConnIdleTime)
	assert.Equal(t, 4*time.Second, cf.OLAP.DialTimeout)
	assert.Equal(t, 20*time.Second, cf.OLAP.ConnectTimeout)
	assert.Equal(t, 40, cf.OLAP.FairpoolTenantMaxPercent)
	assert.Equal(t, 60, cf.OLAP.FairpoolSharedMaxPercent)
	assert.Equal(t, 2*time.Second, cf.OLAP.FairpoolTenantMaxWait)

	// the main settings are untouched by the OLAP overrides
	assert.Equal(t, 50, cf.MaxConns)
	assert.Equal(t, 100, cf.FairpoolTenantMaxPercent)
}
