//go:build darwin
// +build darwin

package security

import "github.com/playwright-community/playwright-go/macutils"

func GetWindowsUUID() (string, error) { return "", nil }
func GetSerialNumber() (string, error) {
	return macutils.GetSerialNumberString()
}
