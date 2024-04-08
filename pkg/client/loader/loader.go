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
	f := client.BindAllEnv

	_, err = loaderutils.LoadConfigFromViper(f, cf)

	if err != nil {
		return nil, fmt.Errorf("could not load config from viper: %w", err)
	}

	// if token is empty, throw an error
	if cf.Token == "" {
		return nil, fmt.Errorf("API token is required. Set it via the HATCHET_CLIENT_TOKEN environment variable.")
	}

	grpcBroadcastAddress := cf.HostPort
	serverURL := cf.HostPort

	tokenConf, err := getConfFromJWT(cf.Token)

	if err == nil {
		if grpcBroadcastAddress == "" && tokenConf.grpcBroadcastAddress != "" {
			grpcBroadcastAddress = tokenConf.grpcBroadcastAddress
		}

		if tokenConf.serverURL != "" {
			serverURL = tokenConf.serverURL
		}
	}

	// if there's no broadcast address at this point, throw an error
	if grpcBroadcastAddress == "" {
		return nil, fmt.Errorf("GRPC broadcast address is required. Set it via the HATCHET_CLIENT_HOST_PORT environment variable.")
	}

	if cf.TenantId == "" {
		cf.TenantId = tokenConf.tenantId
	}

	tlsServerName := cf.TLS.TLSServerName

	// if the tls server name is empty, parse the domain from the host:port
	if tlsServerName == "" {
		// parse the domain from the host:port
		domain, err := parseDomain(grpcBroadcastAddress)

		if err != nil {
			return nil, fmt.Errorf("could not parse domain: %w", err)
		}

		tlsServerName = domain.Hostname()
	}

	tlsConf, err := loaderutils.LoadClientTLSConfig(&cf.TLS, tlsServerName)

	if err != nil {
		return nil, fmt.Errorf("could not load TLS config: %w", err)
	}

	namespace := cf.Namespace

	if namespace != "" {
		namespace = strings.ToLower(namespace + "_")
	}

	return &client.ClientConfig{
		TenantId:             cf.TenantId,
		TLSConfig:            tlsConf,
		Token:                cf.Token,
		ServerURL:            serverURL,
		GRPCBroadcastAddress: grpcBroadcastAddress,
		Namespace:            namespace,
	}, nil
}

func parseDomain(domain string) (*url.URL, error) {
	if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
		domain = "https://" + domain
	}
	return url.Parse(domain)
}
