package utils

import "strings"

func NormalisePath(path string) string {
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}

	return path
}
