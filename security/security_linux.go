//go:build linux
// +build linux

package security

func GetWindowsUUID() (string, error) { return "", nil }
func GetSerialNumber() (string, error) {
	return ""
}
