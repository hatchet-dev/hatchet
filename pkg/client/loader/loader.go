package loader

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

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
	tlsServerName := cf.TLS.TLSServerName

	// if the tls server name is empty, parse the domain from the host:port
	if tlsServerName == "" {
		// parse the domain from the host:port
		domain, err := parseDomain(cf.HostPort)

		if err != nil {
			return nil, fmt.Errorf("could not parse domain: %w", err)
		}

		tlsServerName = domain.Hostname()
	}

	tlsConf, err := loaderutils.LoadClientTLSConfig(&cf.TLS, tlsServerName)

	if err != nil {
		return nil, fmt.Errorf("could not load TLS config: %w", err)
	}

	return &client.ClientConfig{
		TenantId:  cf.TenantId,
		TLSConfig: tlsConf,
		Token:     cf.Token,
	}, nil
}

func parseDomain(domain string) (*url.URL, error) {
	if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
		domain = "https://" + domain
	}
	return url.Parse(domain)
}
