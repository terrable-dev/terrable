package main

import "testing"

func TestBuildInfoNormalizesQuotedValues(t *testing.T) {
	originalConfig := configFile
	originalVersion := version
	t.Cleanup(func() {
		configFile = originalConfig
		version = originalVersion
	})

	configFile = `
version = "1.2.3"
preview-tag = 'rc.1'
`
	version = ""

	info := buildInfo()

	if got := info["version"]; got != "1.2.3" {
		t.Fatalf("expected version 1.2.3, got %q", got)
	}

	if got := info["preview-tag"]; got != "rc.1" {
		t.Fatalf("expected preview-tag rc.1, got %q", got)
	}
}

func TestBuildVersionPrefersInjectedVersion(t *testing.T) {
	originalConfig := configFile
	originalVersion := version
	t.Cleanup(func() {
		configFile = originalConfig
		version = originalVersion
	})

	configFile = "version = 1.2.3\npreview-tag = rc.1\n"
	version = "9.9.9"

	if got := buildVersion(); got != "9.9.9" {
		t.Fatalf("expected injected version, got %q", got)
	}
}

func TestBuildVersionUsesPreviewTagFromBuildInfo(t *testing.T) {
	originalConfig := configFile
	originalVersion := version
	t.Cleanup(func() {
		configFile = originalConfig
		version = originalVersion
	})

	configFile = "version = 1.2.3\npreview-tag = preview.4\n"
	version = ""

	if got := buildVersion(); got != "1.2.3-preview.4" {
		t.Fatalf("expected preview build version, got %q", got)
	}
}

func TestBuildVersionDefaultsToDev(t *testing.T) {
	originalConfig := configFile
	originalVersion := version
	t.Cleanup(func() {
		configFile = originalConfig
		version = originalVersion
	})

	configFile = ""
	version = ""

	if got := buildVersion(); got != "dev" {
		t.Fatalf("expected dev fallback, got %q", got)
	}
}
