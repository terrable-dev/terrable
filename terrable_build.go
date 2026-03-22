package main

import (
	_ "embed"
	"strings"
)

// These are set by GoReleaser via ldflags on tagged and snapshot builds.
var version string

//go:embed terrable_build
var configFile string

func buildInfo() map[string]string {
	config := make(map[string]string)

	lines := strings.Split(configFile, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		parts := strings.SplitN(line, "=", 2)

		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			config[key] = value
		}
	}

	return config
}

func buildVersion() string {
	if version != "" {
		return version
	}

	if fileVersion := buildInfo()["version"]; fileVersion != "" {
		return fileVersion
	}

	return "dev"
}
