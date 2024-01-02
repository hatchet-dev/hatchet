package loader

import (
	"fmt"
	"path/filepath"

	"github.com/hatchet-dev/hatchet/internal/config/client"
	"github.com/hatchet-dev/hatchet/internal/config/loader/loaderutils"
)

type ConfigLoader struct {
	directory string
}

// LoadClientConfig loads the client configuration
func (c *ConfigLoader) LoadClientConfig() (res *client.ClientConfig, err error) {
	sharedFilePath := filepath.Join(c.directory, "client.yaml")
	configFileBytes, err := loaderutils.GetConfigBytes(sharedFilePath)

	if err != nil {
		return nil, err
	}

	cf, err := LoadClientConfigFile(configFileBytes...)

	if err != nil {
		return nil, err
	}

	return GetClientConfigFromConfigFile(cf)
}

// LoadClientConfigFile loads the worker config file via viper
func LoadClientConfigFile(files ...[]byte) (*client.ClientConfigFile, error) {
	configFile := &client.ClientConfigFile{}
	f := client.BindAllEnv

	_, err := loaderutils.LoadConfigFromViper(f, configFile, files...)

	return configFile, err
}

func GetClientConfigFromConfigFile(cf *client.ClientConfigFile) (res *client.ClientConfig, err error) {
	tlsConf, err := loaderutils.LoadClientTLSConfig(&cf.TLS)

	if err != nil {
		return nil, fmt.Errorf("could not load TLS config: %w", err)
	}

	return &client.ClientConfig{
		TenantId:  cf.TenantId,
		TLSConfig: tlsConf,
	}, nil
}
