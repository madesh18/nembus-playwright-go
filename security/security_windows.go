//go:build windows
// +build windows

package security

import (
	"fmt"
	"github.com/StackExchange/wmi"
	"log"
	"strings"
)

func GetSerialNumber() (string, error) {
	return "", nil
}

// https://docs.microsoft.com/en-us/windows/win32/cimwin32prov/win32-computersystemproduct
type Win32_ComputerSystemProduct struct {
	//Caption string
	//Description string
	//IdentifyingNumber string
	//Name string
	//SKUNumber string
	//Vendor string
	//Version string
	UUID string
}

// GetWindowsUUID - Return Windows UUID
func GetWindowsUUID() (string, error) {
	var dst []Win32_ComputerSystemProduct

	err := wmi.Query(wmi.CreateQuery(&dst, ""), &dst)
	if err != nil {
		log.Printf("E! Error while fetching Win32_ComputerSystemProduct info %v", err)
		//return Enc_key, err
	}
	var key string
	if len(dst) > 0 {
		//log.Println("D! [GetWindowsUUID] UUID - ", dst[0].UUID)
		key = strings.ReplaceAll(dst[0].UUID, "-", "")
	}

	//fmt.Println(key)
	key = fmt.Sprintf("%s%s", Enc_key, key)
	keyArr := []byte(key)
	key = string(keyArr[len(keyArr)-32:])
	return key, nil
}
