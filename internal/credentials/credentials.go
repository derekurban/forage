package credentials

import (
	"os"

	"github.com/zalando/go-keyring"
)

const servicePrefix = "github.com/derekurban/forage/"

type Store interface {
	Get(provider, envVar string) (Credential, error)
	Set(provider, value string) error
	Delete(provider string) error
	GetField(provider, field, envVar string) (Credential, error)
	SetField(provider, field, value string) error
	DeleteField(provider, field string) error
}

type Credential struct {
	Value  string `json:"-"`
	Source string `json:"source"`
	Found  bool   `json:"found"`
}

type KeychainStore struct{}

func NewKeychainStore() KeychainStore {
	return KeychainStore{}
}

func (KeychainStore) Get(provider, envVar string) (Credential, error) {
	return KeychainStore{}.GetField(provider, "api_key", envVar)
}

func (KeychainStore) GetField(provider, field, envVar string) (Credential, error) {
	if envVar != "" {
		if v := os.Getenv(envVar); v != "" {
			return Credential{Value: v, Source: "env:" + envVar, Found: true}, nil
		}
	}
	v, err := keyring.Get(servicePrefix+provider, field)
	if err == nil && v != "" {
		return Credential{Value: v, Source: "keychain:" + field, Found: true}, nil
	}
	return Credential{Source: "missing", Found: false}, nil
}

func (KeychainStore) Set(provider, value string) error {
	return KeychainStore{}.SetField(provider, "api_key", value)
}

func (KeychainStore) SetField(provider, field, value string) error {
	return keyring.Set(servicePrefix+provider, field, value)
}

func (KeychainStore) Delete(provider string) error {
	return KeychainStore{}.DeleteField(provider, "api_key")
}

func (KeychainStore) DeleteField(provider, field string) error {
	return keyring.Delete(servicePrefix+provider, field)
}
