package offline

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/gorilla/mux"
	"github.com/terrable-dev/terrable/config"
	"github.com/terrable-dev/terrable/utils"
)

func Run(filePath string, moduleName string, port string) error {
	terrableConfig, err := utils.ParseTerraformFile(filePath, moduleName)

	if err != nil {
		log.Fatalf("error running offline: %s", err)
	}

	// TODO: Validate config

	tomlConfig, err := config.ParseTerrableToml(filepath.Dir(filePath))

	if err != nil {
		panic(fmt.Errorf("error parsing .terrable.toml file: %w", err))
	}

	printConfig(*terrableConfig)

	var wg sync.WaitGroup
	defer wg.Done()

	r := mux.NewRouter()

	for _, handler := range terrableConfig.Handlers {
		go ServeHandler(&HandlerInstance{
			handlerConfig: handler,
			envVars:       tomlConfig.Environment,
		}, r)
	}

	fmt.Printf("Starting server on :%s", port)

	if err := http.ListenAndServe(fmt.Sprintf("127.0.0.1:%s", port), r); err != nil {
		return fmt.Errorf("could not start server on port %s. Error: %s", port, err.Error())
	}

	return nil
}

func printConfig(config config.TerrableConfig) {
	fmt.Printf("Starting terrable local server... \n")
	fmt.Printf("%d Endpoints to prepare... \n", len(config.Handlers))

	for _, handler := range config.Handlers {
		fmt.Printf(" - %s %s \n", handler.Http.Method, handler.Http.Path)
	}
}
