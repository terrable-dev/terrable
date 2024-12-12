package offline

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/terrable-dev/terrable/config"
)

func TestPrintConfig(t *testing.T) {
	testConfig := config.TerrableConfig{
		Handlers: []config.HandlerMapping{
			{
				Name:   "Handler1",
				Source: "source1",
				Http: map[string]string{
					"GET":  "/path1",
					"POST": "/path1",
				},
			},
			{
				Name:   "Handler2",
				Source: "source2",
				Http: map[string]string{
					"GET": "/path2",
				},
			},
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call the function
	printConfig(testConfig, 1234)

	// Restore stdout
	w.Close()
	os.Stdout = old

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Test for minimal required content without formatting
	expectedEndpoints := []string{
		"GET   http://localhost:1234/path1  (Handler1)",
		"POST  http://localhost:1234/path1  (Handler1)",
		"GET   http://localhost:1234/path2  (Handler2)",
	}

	for _, endpoint := range expectedEndpoints {
		if !strings.Contains(output, strings.TrimSpace(endpoint)) {
			t.Errorf("Expected output to contain endpoint '%s', but it doesn't.\nActual output:\n%s",
				endpoint, output)
		}
	}

	// Test order of endpoints (GET before POST)
	getIndex := strings.Index(output, "GET")
	postIndex := strings.Index(output, "POST")
	if getIndex > postIndex {
		t.Error("Expected GET endpoint to appear before POST endpoint")
	}
}
