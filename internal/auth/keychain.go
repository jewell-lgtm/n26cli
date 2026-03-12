package auth

import (
	"github.com/zalando/go-keyring"
)

const keychainService = "n26"

// GetPasswordFromKeychain retrieves the N26 password from the OS keychain.
// Returns empty string (not error) if no entry exists.
func GetPasswordFromKeychain(email string) string {
	pw, err := keyring.Get(keychainService, email)
	if err != nil {
		return ""
	}
	return pw
}

// SavePasswordToKeychain stores the N26 password in the OS keychain.
func SavePasswordToKeychain(email, password string) error {
	return keyring.Set(keychainService, email, password)
}

// DeletePasswordFromKeychain removes the N26 password from the OS keychain.
func DeletePasswordFromKeychain(email string) error {
	return keyring.Delete(keychainService, email)
}
