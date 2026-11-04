package profilestore

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/zalando/go-keyring"
)

const (
	TokenStoreAuto    = "auto"
	TokenStoreKeyring = "keyring"
	TokenStoreFile    = "file"
)

const keyringService = "hatchet"

type Secrets interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key string) error
}

func KeyringName() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS Keychain"
	case "windows":
		return "Windows Credential Manager"
	default:
		return "system keyring"
	}
}

type systemKeyring struct{}

func (systemKeyring) Get(key string) (string, error) { return keyring.Get(keyringService, key) }
func (systemKeyring) Set(key, value string) error    { return keyring.Set(keyringService, key, value) }
func (systemKeyring) Delete(key string) error        { return keyring.Delete(keyringService, key) }

type MemorySecrets map[string]string

func (m MemorySecrets) Get(key string) (string, error) {
	v, ok := m[key]
	if !ok {
		return "", errors.New("secret not found")
	}
	return v, nil
}
func (m MemorySecrets) Set(key, value string) error { m[key] = value; return nil }
func (m MemorySecrets) Delete(key string) error     { delete(m, key); return nil }

var probeKeyring = sync.OnceValue(func() error {
	k := systemKeyring{}
	if err := k.Set("probe", "ok"); err != nil {
		return err
	}
	if _, err := k.Get("probe"); err != nil {
		return err
	}
	return k.Delete("probe")
})

func effectiveMode(mode string) string {
	if env := os.Getenv("HATCHET_CLI_TOKEN_STORE"); env != "" {
		mode = env
	}
	mode = strings.ToLower(mode)
	if mode == "" {
		mode = TokenStoreAuto
	}
	if mode == TokenStoreAuto && testing.Testing() {
		mode = TokenStoreFile
	}
	return mode
}

func resolveSecrets(mode string) (Secrets, error) {
	switch effectiveMode(mode) {
	case TokenStoreFile:
		return nil, nil
	case TokenStoreKeyring:
		if err := probeKeyring(); err != nil {
			return nil, fmt.Errorf("%s is not available: %w", KeyringName(), err)
		}
		return systemKeyring{}, nil
	case TokenStoreAuto:
		if err := probeKeyring(); err != nil {
			return nil, nil
		}
		return systemKeyring{}, nil
	default:
		return nil, fmt.Errorf("tokenStore must be one of auto, keyring, file (got %q)", mode)
	}
}
