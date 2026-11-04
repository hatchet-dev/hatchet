package profilestore

import (
	"errors"
	"runtime"
	"sync"
	"testing"

	"github.com/zalando/go-keyring"
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

// keyringAvailable probes the system keyring once per process with a set, get
// and delete. Headless Linux without a Secret Service, a locked macOS Keychain
// over SSH, and most containers fail here; tests never touch it.
var keyringAvailable = sync.OnceValue(func() bool {
	if testing.Testing() {
		return false
	}
	k := systemKeyring{}
	if err := k.Set("probe", "ok"); err != nil {
		return false
	}
	if _, err := k.Get("probe"); err != nil {
		return false
	}
	return k.Delete("probe") == nil
})
