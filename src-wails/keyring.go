package main

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	CredentialService = "com.lisitede.preset.wails"
	DemoAccount       = "PRESET_WAILS_K1"
)

func loadCredential(account string) (string, error) {
	value, err := keyring.Get(CredentialService, account)
	if err == nil {
		if value == "" {
			return "", fmt.Errorf("credential %q is empty", account)
		}
		return value, nil
	}

	if !errors.Is(err, keyring.ErrNotFound) {
		return "", fmt.Errorf("read service %q account %q: %w", CredentialService, account, err)
	}

	return "", fmt.Errorf("credential %q not found in keychain", account)
}
