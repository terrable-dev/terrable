package offline

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/terrable-dev/terrable/config"
)

func TestPrintConfig(t *testing.T) {
	testConfig := config.TerrableConfig{
		HttpApi: &config.APIGatewayConfig{
			Cors: &config.CorsConfig{
				AllowOrigins: []string{"*"},
			},
		},
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
			{
				Name:   "ScheduledHandler",
				Source: "source4",
				Schedule: &config.ScheduleConfig{
					Expression: "rate(5 minutes)",
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

	expectedFragments := []string{
		"http://localhost:1234/path1",
		"http://localhost:1234/path2",
		"http://localhost:1234/_sqs/SqsHandler",
		"http://localhost:1234/_scheduled/ScheduledHandler",
		"(Handler1)",
		"(Handler2)",
		"(SqsHandler)",
		"(ScheduledHandler)",
		"(CORS)",
		"SQS Handlers",
		"Scheduled",
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(output, fragment) {
			t.Errorf("Expected output to contain fragment '%s', but it doesn't.\nActual output:\n%s",
				fragment, output)
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

func TestCombineHandlerPreparationErrors(t *testing.T) {
	firstErr := errors.New("first handler failed")
	secondErr := errors.New("second handler failed")

	err := combineHandlerPreparationErrors([]error{firstErr, nil, secondErr})
	if err == nil {
		t.Fatal("expected combined handler preparation error")
	}

	expectedFragments := []string{
		"Terrable could not start because one or more handlers failed to prepare.",
		"first handler failed",
		"second handler failed",
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("expected combined error to contain %q, got %q", fragment, err.Error())
		}
	}

	if strings.Index(err.Error(), "first handler failed") > strings.Index(err.Error(), "second handler failed") {
		t.Fatalf("expected combined errors to preserve handler order, got %q", err.Error())
	}
}
