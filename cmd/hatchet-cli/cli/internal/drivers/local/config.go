package local

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"

	"github.com/hatchet-dev/hatchet/pkg/random"
)

// StoredKeys holds the encryption keys for the local server
type StoredKeys struct {
	MasterKey     string `json:"master_key"`
	PrivateJWT    string `json:"private_jwt"`
	PublicJWT     string `json:"public_jwt"`
	CookieSecrets string `json:"cookie_secrets"`
}

// DatabaseConfigFile is a minimal database config for local mode
type DatabaseConfigFile struct {
	Seed DatabaseSeedConfig `yaml:"seed"`
}

// DatabaseSeedConfig holds seed configuration
type DatabaseSeedConfig struct {
	AdminEmail        string `yaml:"adminEmail"`
	AdminPassword     string `yaml:"adminPassword"`
	AdminName         string `yaml:"adminName"`
	DefaultTenantName string `yaml:"defaultTenantName"`
	DefaultTenantSlug string `yaml:"defaultTenantSlug"`
	DefaultTenantID   string `yaml:"defaultTenantId"`
}

// ServerConfigFile is a minimal server config for local mode
type ServerConfigFile struct {
	Auth              ServerAuthConfig       `yaml:"auth,omitempty"`
	Encryption        ServerEncryptionConfig `yaml:"encryption,omitempty"`
	Runtime           ServerRuntimeConfig    `yaml:"runtime,omitempty"`
	MessageQueue      MessageQueueConfig     `yaml:"msgQueue,omitempty"`
	SecurityCheck     SecurityCheckConfig    `yaml:"securityCheck,omitempty"`
	EnableDataRetention   bool               `yaml:"enableDataRetention,omitempty"`
	EnableWorkerRetention bool               `yaml:"enableWorkerRetention,omitempty"`
}

// ServerAuthConfig holds auth configuration
type ServerAuthConfig struct {
	Cookie           ServerCookieConfig `yaml:"cookie,omitempty"`
	SetEmailVerified bool               `yaml:"setEmailVerified,omitempty"`
}

// ServerCookieConfig holds cookie configuration
type ServerCookieConfig struct {
	Domain   string `yaml:"domain,omitempty"`
	Insecure bool   `yaml:"insecure,omitempty"`
	Secrets  string `yaml:"secrets,omitempty"`
	Name     string `yaml:"name,omitempty"`
}

// ServerEncryptionConfig holds encryption configuration
type ServerEncryptionConfig struct {
	MasterKeyset string          `yaml:"masterKeyset,omitempty"`
	JWT          ServerJWTConfig `yaml:"jwt,omitempty"`
}

// ServerJWTConfig holds JWT configuration
type ServerJWTConfig struct {
	PrivateJWTKeyset string `yaml:"privateJwtKeyset,omitempty"`
	PublicJWTKeyset  string `yaml:"publicJwtKeyset,omitempty"`
}

// ServerRuntimeConfig holds runtime configuration
type ServerRuntimeConfig struct {
	Port                 int    `yaml:"port,omitempty"`
	ServerURL            string `yaml:"serverUrl,omitempty"`
	GRPCPort             int    `yaml:"grpcPort,omitempty"`
	GRPCBindAddress      string `yaml:"grpcBindAddress,omitempty"`
	GRPCBroadcastAddress string `yaml:"grpcBroadcastAddress,omitempty"`
	GRPCInsecure         bool   `yaml:"grpcInsecure,omitempty"`
	AllowSignup          bool   `yaml:"allowSignup,omitempty"`
	AllowInvites         bool   `yaml:"allowInvites,omitempty"`
	AllowCreateTenant    bool   `yaml:"allowCreateTenant,omitempty"`
	AllowChangePassword  bool   `yaml:"allowChangePassword,omitempty"`
}

// MessageQueueConfig holds message queue configuration
type MessageQueueConfig struct {
	Enabled bool   `yaml:"enabled,omitempty"`
	Kind    string `yaml:"kind,omitempty"`
}

// SecurityCheckConfig holds security check configuration
type SecurityCheckConfig struct {
	Enabled bool `yaml:"enabled,omitempty"`
}

func (d *LocalDriver) loadKeys(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var keys StoredKeys
	if err := json.Unmarshal(data, &keys); err != nil {
		return err
	}

	d.masterKey = keys.MasterKey
	d.privateJWT = keys.PrivateJWT
	d.publicJWT = keys.PublicJWT
	d.cookieSecrets = keys.CookieSecrets

	return nil
}

func (d *LocalDriver) saveKeys(path string) error {
	keys := StoredKeys{
		MasterKey:     d.masterKey,
		PrivateJWT:    d.privateJWT,
		PublicJWT:     d.publicJWT,
		CookieSecrets: d.cookieSecrets,
	}

	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func generateCookieSecrets() (string, error) {
	hashKey, err := random.Generate(16)
	if err != nil {
		return "", fmt.Errorf("failed to generate cookie hash key: %w", err)
	}

	blockKey, err := random.Generate(16)
	if err != nil {
		return "", fmt.Errorf("failed to generate cookie block key: %w", err)
	}

	return fmt.Sprintf("%s %s", hashKey, blockKey), nil
}

func (d *LocalDriver) writeConfigFiles() error {
	if err := d.writeDatabaseConfig(); err != nil {
		return fmt.Errorf("failed to write database config: %w", err)
	}
	if err := d.writeServerConfig(); err != nil {
		return fmt.Errorf("failed to write server config: %w", err)
	}
	return nil
}

func (d *LocalDriver) writeDatabaseConfig() error {
	config := &DatabaseConfigFile{
		Seed: DatabaseSeedConfig{
			AdminEmail:        "admin@example.com",
			AdminPassword:     "Admin123!!",
			AdminName:         "Admin",
			DefaultTenantName: "Default",
			DefaultTenantSlug: "default",
			DefaultTenantID:   DefaultTenantID,
		},
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	configPath := filepath.Join(d.configDir, "database.yaml")
	return os.WriteFile(configPath, data, 0600)
}

func (d *LocalDriver) writeServerConfig() error {
	config := &ServerConfigFile{
		Auth: ServerAuthConfig{
			Cookie: ServerCookieConfig{
				Domain:   "localhost",
				Insecure: true,
				Secrets:  d.cookieSecrets,
				Name:     "hatchet",
			},
			SetEmailVerified: true,
		},
		Encryption: ServerEncryptionConfig{
			MasterKeyset: d.masterKey,
			JWT: ServerJWTConfig{
				PrivateJWTKeyset: d.privateJWT,
				PublicJWTKeyset:  d.publicJWT,
			},
		},
		Runtime: ServerRuntimeConfig{
			Port:                 d.apiPort,
			ServerURL:            fmt.Sprintf("http://localhost:%d", d.apiPort),
			GRPCPort:             d.grpcPort,
			GRPCBindAddress:      "0.0.0.0",
			GRPCBroadcastAddress: fmt.Sprintf("localhost:%d", d.grpcPort),
			GRPCInsecure:         true,
			AllowSignup:          true,
			AllowInvites:         true,
			AllowCreateTenant:    true,
			AllowChangePassword:  true,
		},
		MessageQueue: MessageQueueConfig{
			Enabled: true,
			Kind:    "postgres",
		},
		SecurityCheck: SecurityCheckConfig{
			Enabled: false,
		},
		EnableDataRetention:   true,
		EnableWorkerRetention: false,
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	configPath := filepath.Join(d.configDir, "server.yaml")
	return os.WriteFile(configPath, data, 0600)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
