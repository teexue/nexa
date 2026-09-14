package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/teexue/nexa/core/store"
)

// CredentialStore is a thread-safe store for API credentials backed by SQLite.
type CredentialStore struct {
	inner *store.CredentialStore
}

// NewCredentialStore loads credentials from the bound state.db.
func NewCredentialStore(home string) (*CredentialStore, error) {
	_ = home
	db, err := requireDB()
	if err != nil {
		return nil, err
	}
	inner, err := store.NewCredentialStore(db)
	if err != nil {
		return nil, err
	}
	return &CredentialStore{inner: inner}, nil
}

// Set stores a credential key-value pair and persists.
func (cs *CredentialStore) Set(envName, value string) error {
	if cs == nil || cs.inner == nil {
		return fmt.Errorf("credential store is not configured")
	}
	return cs.inner.Set(envName, value)
}

// Get returns a stored credential by env var name.
func (cs *CredentialStore) Get(envName string) string {
	if cs == nil || cs.inner == nil {
		return ""
	}
	return cs.inner.Get(envName)
}

// Lookup returns env first, then store.
func (cs *CredentialStore) Lookup(envName string) string {
	if cs == nil || cs.inner == nil {
		return os.Getenv(envName)
	}
	return cs.inner.Lookup(envName)
}

// Keys returns configured env var names.
func (cs *CredentialStore) Keys() []string {
	if cs == nil || cs.inner == nil {
		return nil
	}
	return cs.inner.Keys()
}

var (
	legacyStore *CredentialStore
	legacyMu    sync.Mutex
)

func initLegacy() error {
	legacyMu.Lock()
	defer legacyMu.Unlock()
	if legacyStore != nil {
		return nil
	}
	home, err := Home(false)
	if err != nil {
		return err
	}
	cs, err := NewCredentialStore(home)
	if err != nil {
		return err
	}
	legacyStore = cs
	return nil
}

// LoadCredentials reads credentials into memory.
// Deprecated: use NewCredentialStore instead.
func LoadCredentials(home string) error {
	cs, err := NewCredentialStore(home)
	if err != nil {
		return err
	}
	legacyMu.Lock()
	defer legacyMu.Unlock()
	legacyStore = cs
	return nil
}

// SetCredential stores a key.
// Deprecated: use CredentialStore.Set instead.
func SetCredential(home, envName, value string) error {
	cs, err := NewCredentialStore(home)
	if err != nil {
		return err
	}
	return cs.Set(envName, value)
}

// GetCredential returns a stored credential.
// Deprecated: use CredentialStore.Get instead.
func GetCredential(envName string) string {
	if legacyStore == nil {
		_ = initLegacy()
	}
	if legacyStore == nil {
		return ""
	}
	return legacyStore.Get(envName)
}

// ListCredentialKeys returns configured env var names.
// Deprecated: use CredentialStore.Keys instead.
func ListCredentialKeys(home string) ([]string, error) {
	cs, err := NewCredentialStore(home)
	if err != nil {
		return nil, err
	}
	return cs.Keys(), nil
}
