package macutils

import (
	"fmt"
	"github.com/micromdm/plist"
	"log"
	"strings"
	"time"
)

func GetSerialNumber(mapHardwareDataType []map[string]interface{}) (string, error) {
	// log.Println("D! GetSerialNumber: Inside GetSerialNumber")
	var serialNumberString string
	if len(mapHardwareDataType) > 0 {
		if items, ok := mapHardwareDataType[0]["_items"]; ok {
			if itemsInterface, okk := items.([]interface{}); okk {
				if len(itemsInterface) > 0 {
					if itemsInterfaceMap, okk := itemsInterface[0].(map[string]interface{}); okk {
						if serialNumber, okk := itemsInterfaceMap["serial_number"]; okk {
							if serialNumberString, okk = serialNumber.(string); okk {
								//log.Println("D! GetSerialNumber: Serial Number value is ", serialNumberString)
								return serialNumberString, nil
							}
						}
					}
				}
			}
		}
	}
	log.Println("E! Error in fetching GetSerialNumber The HardwareInfo map is", mapHardwareDataType)
	return "", fmt.Errorf("error in fetching the SerialNumber")
}

func GetSerialNumberString() (string, error) {
	return GetSerialNumber(GetHardwareMap())
}

func GetHardwareMap() []map[string]interface{} {

	var mapHardwareDataType []map[string]interface{}
	SPHardwareDataType, errHard := RunCommandMac("system_profiler", 5*time.Second, "SPHardwareDataType -xml")

	if errHard == nil {
		if err := plist.NewXMLDecoder(strings.NewReader(SPHardwareDataType)).Decode(&mapHardwareDataType); err != nil {
			log.Println("E! Error in decoding the SPHardwareDataType", err)
			return nil
			// return err
		}
	} else {
		log.Println("E! error while fetching hardware info", errHard)
		return nil
	}
	return mapHardwareDataType
}
