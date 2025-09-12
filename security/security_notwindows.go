//go:build !windows && !linux && !darwin
// +build !windows,!linux,!darwin

package security

func GetWindowsUUID() (string, error) {
	return "", nil
}
