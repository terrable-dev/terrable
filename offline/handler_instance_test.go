package offline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/terrable-dev/terrable/config"
)

func TestCompileHandlerReturnsHelpfulErrorForMissingSourceFile(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing-handler.ts")

	handler := &HandlerInstance{
		handlerConfig: config.HandlerMapping{
			Name:             "MissingHandler",
			Source:           missingPath,
			ConfiguredSource: "./missing-handler.ts",
		},
	}

	_, err := handler.CompileHandler()
	if err == nil {
		t.Fatal("expected missing handler source to return an error")
	}

	expectedFragments := []string{
		`Handler "MissingHandler" could not be loaded.`,
		`Configured source:`,
		`./missing-handler.ts`,
		`Resolved path:`,
		missingPath,
		`Problem:`,
		`no file exists at that path`,
		`Check the handler's "source" setting and try again.`,
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("expected error to contain %q, got %q", fragment, err.Error())
		}
	}
}

func TestCompileHandlerReturnsHelpfulErrorForMalformedSourceFile(t *testing.T) {
	tempDir := t.TempDir()
	brokenPath := filepath.Join(tempDir, "broken-handler.ts")

	if err := os.WriteFile(brokenPath, []byte("export const handler = () => {\n"), 0o644); err != nil {
		t.Fatalf("failed to write broken handler fixture: %v", err)
	}

	handler := &HandlerInstance{
		handlerConfig: config.HandlerMapping{
			Name:             "BrokenHandler",
			Source:           brokenPath,
			ConfiguredSource: "./broken-handler.ts",
		},
	}

	_, err := handler.CompileHandler()
	if err == nil {
		t.Fatal("expected malformed handler source to return an error")
	}

	expectedFragments := []string{
		`Handler "BrokenHandler" could not be compiled.`,
		`Configured source:`,
		`./broken-handler.ts`,
		`Resolved path:`,
		brokenPath,
		`Build errors:`,
		`Fix the handler code and try again.`,
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("expected error to contain %q, got %q", fragment, err.Error())
		}
	}
}
