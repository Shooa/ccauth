//go:build !darwin

package creds

import "fmt"

func ReadKeychain() (Blob, error) {
	return Blob{}, fmt.Errorf("keychain: unsupported on this platform")
}

func WriteKeychain(b Blob) error {
	return fmt.Errorf("keychain: unsupported on this platform")
}
