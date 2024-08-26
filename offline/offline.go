package offline

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/terrable-dev/terrable/config"
	"github.com/terrable-dev/terrable/utils"
)

func Run(filePath string, moduleName string) {
	config, err := utils.ParseTerraformFile(filePath, moduleName)

	if err != nil {
		log.Fatalf("error running offline: %s", err)
	}

	// TODO: Validate config

	printConfig(*config)

	var wg sync.WaitGroup
	defer wg.Done()

	for _, handler := range config.Handlers {
		go ServeHandler(&HandlerInstance{
			handlerConfig: handler,
		})
	}

	fmt.Println("Starting server on :8080")
	http.ListenAndServe("127.0.0.1:8080", nil)

	wg.Add(1)
}

func printConfig(config config.TerrableConfig) {
	fmt.Printf("Starting terrable local server... \n")
	fmt.Printf("%d Endpoints to prepare... \n", len(config.Handlers))

	for _, handler := range config.Handlers {
		fmt.Printf(" - %s %s \n", handler.Http.Method, handler.Http.Path)
	}
}
