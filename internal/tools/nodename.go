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

func GetHAPinGroupName(name string) string {
	return fmt.Sprintf("crs-pin-%s", GetNodeName(name))
}

func GetHAPreferGroupName(name string) string {
	return fmt.Sprintf("crs-prefer-%s", GetNodeName(name))
}
