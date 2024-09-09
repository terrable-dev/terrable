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

func Run(filePath string, moduleName string) {
	terrableConfig, err := utils.ParseTerraformFile(filePath, moduleName)

	if err != nil {
		log.Fatalf("error running offline: %s", err)
	}

	// TODO: Validate config

	tomlConfig, err := config.ParseTerrableToml(filepath.Dir(filePath))

	if err != nil {
		log.Fatalf("error parsing .terrable.toml file: %w", err)
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

	fmt.Println("Starting server on :8080")
	http.ListenAndServe("127.0.0.1:8080", r)

	wg.Add(1)
}

func printConfig(config config.TerrableConfig) {
	fmt.Printf("Starting terrable local server... \n")
	fmt.Printf("%d Endpoints to prepare... \n", len(config.Handlers))

	for _, handler := range config.Handlers {
		fmt.Printf(" - %s %s \n", handler.Http.Method, handler.Http.Path)
	}
}
