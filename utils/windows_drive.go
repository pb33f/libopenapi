package utils

import "strings"

func ReplaceWindowsDriveWithLinuxPath(path string) string {
	if len(path) > 1 && path[1] == ':' {
		path = strings.ReplaceAll(path, "\\", "/")
		return path[2:]
	}
	return strings.ReplaceAll(path, "\\", "/")
}
