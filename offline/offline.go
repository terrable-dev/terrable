package offline

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/gorilla/mux"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/terrable-dev/terrable/config"
	"github.com/terrable-dev/terrable/utils"
)

var DebugConfig config.DebugConfig

func Run(filePath string, moduleName string, port string, debugConfig config.DebugConfig, envFile string) error {
	DebugConfig = debugConfig
	terrableConfig, err := utils.ParseTerraformFile(filePath, moduleName)

	if err != nil {
		log.Fatalf("error running offline: %s", err)
	}

	err = validateConfig(terrableConfig)

	if err != nil {
		return fmt.Errorf(`error validating configuration: %s`, err.Error())
	}

	listener, activePort, err := getListener(port)

	if err != nil {
		return err
	}

	defer listener.Close()

	printConfig(*terrableConfig, activePort)

	var wg sync.WaitGroup
	defer wg.Wait()

	// Read environment variables from the specified env file (if provided)
	var fileEnvVars map[string]string
	if envFile != "" {
		fileEnvVars, err = readEnvFile(envFile)
		if err != nil {
			return fmt.Errorf("could not read env file: %w", err)
		}
	}

	r := mux.NewRouter()

	// Not Found handlers
	r.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Not Found"}`))
	})

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Not Found"}`))
	})

	// Start compiling and serving each handler
	for _, handler := range terrableConfig.Handlers {
		wg.Add(1)

		go func(handler config.HandlerMapping) {
			defer wg.Done()

			ServeHandler(&HandlerInstance{
				handlerConfig: handler,
				envVars:       mergeEnvMaps(terrableConfig.GlobalEnvironmentVariables, mergeEnvMaps(handler.EnvironmentVariables, fileEnvVars)),
			}, r)
		}(handler)
	}

	server := &http.Server{
		Handler: r,
	}

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("could not start server on port %d. Error: %w", activePort, err)
	}

	return nil
}

func validateConfig(config *config.TerrableConfig) error {
	var errs []string

	for _, handler := range config.Handlers {
		for method, path := range handler.Http {
			if !strings.HasPrefix(path, "/") {
				errs = append(errs, fmt.Sprintf("Handler '%s' does not have a '/' prefix for the HTTP route %s '%s'.", handler.Name, method, path))
			}
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

func getListener(port string) (net.Listener, int, error) {
	var specificPortDesired bool = (port != "")

	if port == "" {
		port = "8080"
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%s", port))

	if err != nil {
		if specificPortDesired {
			return nil, 0, fmt.Errorf("could not start server on specified port %s: %w", port, err)
		}

		// Try with a random free-port
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, 0, fmt.Errorf("could not start server on any available port: %w", err)
		}
	}

	return listener, listener.Addr().(*net.TCPAddr).Port, nil
}

func printConfig(config config.TerrableConfig, port int) {
	totalEndpoints := 0
	totalSqsQueues := 0

	t := table.NewWriter()

	t.SetOutputMirror(os.Stdout)

	// Check for SQS queues
	var hasSqsQueues bool
	for _, handler := range config.Handlers {
		if len(handler.Sqs) > 0 {
			hasSqsQueues = true
			break
		}
	}

	methodColor := color.New(color.FgHiBlue).SprintFunc()
	hostColor := color.New(color.FgHiBlack).SprintFunc()
	pathColor := color.New(color.FgHiGreen).SprintFunc()
	handlerNameColor := color.New(color.FgHiBlack).SprintFunc()

	for _, handler := range config.Handlers {
		for method, path := range handler.Http {
			totalEndpoints++

			url := fmt.Sprintf("%s%s",
				hostColor(fmt.Sprintf("http://localhost:%d", port)),
				pathColor(path))

			t.AppendRow(table.Row{
				methodColor(method),
				url,
				handlerNameColor(fmt.Sprintf("(%s)", handler.Name)),
			})
		}
	}

	if hasSqsQueues {
		t.AppendRow(table.Row{
			"\nSQS Handlers\n",
			"",
			"",
		})
	}

	for _, handler := range config.Handlers {
		for range handler.Sqs {
			totalSqsQueues++
			handlerNameColor := color.New(color.FgHiBlack).SprintFunc()

			url := fmt.Sprintf("%s%s",
				hostColor(fmt.Sprintf("http://localhost:%d/_sqs/", port)),
				pathColor(handler.Name))

			t.AppendRow(table.Row{
				"POST",
				url,
				handlerNameColor(fmt.Sprintf("(%s)", handler.Name)),
			})
		}
	}

	color.New(color.FgHiGreen, color.Bold).Println("Starting terrable local server...")

	endpointMessage := "Endpoint to prepare..."

	if totalEndpoints != 1 {
		endpointMessage = "Endpoints to prepare..."
	}

	color.New(color.FgHiBlue, color.Bold).Printf("%d %s\n\n", totalEndpoints, endpointMessage)

	t.SetStyle(table.StyleLight)
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false
	t.Style().Options.SeparateHeader = false

	t.Render()

	color.New(color.FgHiGreen, color.Bold).Printf("\nServer started on :%d\n\n", port)
}

func mergeEnvMaps(global, local map[string]string) map[string]string {
	merged := make(map[string]string, len(global)+len(local))

	for k, v := range global {
		merged[k] = v
	}

	for k, v := range local {
		merged[k] = v
	}

	return merged
}

func readEnvFile(filePath string) (map[string]string, error) {
	envVars := make(map[string]string)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			envVars[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return envVars, nil
}
