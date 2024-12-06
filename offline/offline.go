package offline

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/fatih/color"
	"github.com/gorilla/mux"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/terrable-dev/terrable/config"
	"github.com/terrable-dev/terrable/utils"
)

func Run(filePath string, moduleName string, port string) error {
	terrableConfig, err := utils.ParseTerraformFile(filePath, moduleName)

	if err != nil {
		log.Fatalf("error running offline: %s", err)
	}

	// TODO: Validate config

	if err != nil {
		panic(fmt.Errorf("error parsing .terrable.toml file: %w", err))
	}

	listener, activePort, err := getListener(port)

	if err != nil {
		return err
	}

	defer listener.Close()

	printConfig(*terrableConfig, activePort)

	var wg sync.WaitGroup
	defer wg.Wait()

	r := mux.NewRouter()

	// Start compiling and serving each handler
	for _, handler := range terrableConfig.Handlers {
		wg.Add(1)

		go func(handler config.HandlerMapping) {
			defer wg.Done()

			ServeHandler(&HandlerInstance{
				handlerConfig: handler,
				envVars:       mergeEnvMaps(terrableConfig.GlobalEnvironmentVariables, handler.EnvironmentVariables),
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
	t := table.NewWriter()

	t.SetOutputMirror(os.Stdout)

	for _, handler := range config.Handlers {
		for method, path := range handler.Http {
			totalEndpoints++
			methodColor := color.New(color.FgHiBlue).SprintFunc()
			hostColor := color.New(color.FgHiBlack).SprintFunc()
			pathColor := color.New(color.FgHiGreen).SprintFunc()
			handlerNameColor := color.New(color.FgHiBlack).SprintFunc()

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

	color.New(color.FgHiGreen, color.Bold).Printf("\nServer started on :%d\n", port)
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
