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
			{
				Name:   "SqsHandler",
				Source: "source3",
				Sqs: map[string]interface{}{
					"queue": "arn:aws:sqs:region:account:queue",
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
		"GET           http://localhost:1234/path1            (Handler1) ",
		"POST          http://localhost:1234/path1            (Handler1) ",
		"GET           http://localhost:1234/path2            (Handler2) ",
		"POST          http://localhost:1234/_sqs/SqsHandler  (SqsHandler)",
	}

	// Verify HTTP endpoints
	for _, endpoint := range expectedEndpoints {
		if !strings.Contains(output, strings.TrimSpace(endpoint)) {
			t.Errorf("Expected output to contain endpoint '%s', but it doesn't.\nActual output:\n%s",
				endpoint, output)
		}
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.TerrableConfig
		expectErr bool
	}{
		{
			name: "ValidConfig",
			config: &config.TerrableConfig{
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
			},
			expectErr: false,
		},
		{
			name: "InvalidConfig",
			config: &config.TerrableConfig{
				Handlers: []config.HandlerMapping{
					{
						Name:   "Handler1",
						Source: "source1",
						Http: map[string]string{
							"GET":  "path1",
							"POST": "/path1",
						},
					},
					{
						Name:   "Handler2",
						Source: "source2",
						Http: map[string]string{
							"GET": "path2",
						},
					},
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.expectErr {
				t.Errorf("validateConfig() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}
