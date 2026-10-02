package creds

import "runtime"

// ReadCurrent returns the live credentials: Keychain on macOS, the
// credentials file elsewhere (with file fallback when Keychain fails).
func ReadCurrent() (Blob, string, error) {
	if runtime.GOOS == "darwin" {
		b, err := ReadKeychain()
		if err == nil {
			return b, "keychain", nil
		}
		if FileExists() {
			fb, ferr := ReadFile()
			if ferr == nil {
				return fb, "file", nil
			}
		}
		return Blob{}, "", err
	}
	b, err := ReadFile()
	return b, "file", err
}

// Restore writes the profile back to the live location(s).
func Restore(b Blob) error {
	if runtime.GOOS == "darwin" {
		if err := WriteKeychain(b); err != nil {
			return err
		}
		// Refresh the on-disk file only if it already exists, so a stale
		// copy can't shadow the Keychain token.
		if FileExists() {
			return WriteFile(b)
		}
		return nil
	}
	return WriteFile(b)
}
