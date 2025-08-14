package tools

import (
	"fmt"
	"strings"
)

func GetNodeName(name string) string {
	list := strings.Split(name, ".")

	if len(list) != 1 {
		return strings.ToLower(name)
	}

	return strings.ToLower(list[0])
}

func GetHAVMPinGroupName(name string) string {
	return fmt.Sprintf("crs-vm-pin-%s", GetNodeName(name))
}

func GetHAVMPreferGroupName(name string) string {
	return fmt.Sprintf("crs-vm-prefer-%s", GetNodeName(name))
}
