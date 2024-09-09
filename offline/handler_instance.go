package offline

import (
	"fmt"
	"strings"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/fsnotify/fsnotify"
	"github.com/terrable-dev/terrable/config"
)

type HandlerInstance struct {
	handlerConfig     config.HandlerMapping
	handlerCode       string
	readCodeMutex     sync.RWMutex
	recompileSyncLock *sync.Once
	envVars           map[string]interface{}
}

func (handlerInstance *HandlerInstance) GetExecutionCode() string {
	handlerInstance.readCodeMutex.RLock()
	defer handlerInstance.readCodeMutex.RUnlock()

	return handlerInstance.handlerCode
}

func (handlerInstance *HandlerInstance) SetExecutionCode(code string) {
	handlerInstance.readCodeMutex.Lock()
	defer handlerInstance.readCodeMutex.Unlock()

	handlerInstance.handlerCode = code
}

func (handlerInstance *HandlerInstance) RecompileHandler() {
	result := api.Build(api.BuildOptions{
		EntryPoints: []string{handlerInstance.handlerConfig.Source},
		Bundle:      true,
		Write:       false,
		Format:      api.FormatIIFE,
		Target:      api.ES2015,
		Sourcemap:   api.SourceMapInline,
		GlobalName:  "exports",
	})

	if len(result.Errors) > 0 {
		printBuildErrors(result.Errors)
		return
	}

	handlerInstance.SetExecutionCode(string(result.OutputFiles[0].Contents))
}

func printBuildErrors(result []api.Message) {
	fmt.Println("\n🚨 Build Errors:")
	fmt.Println(strings.Repeat("=", 50))

	for i, err := range result {
		fmt.Printf("Error #%d:\n", i+1)
		fmt.Printf("  File: %s\n", err.Location.File)
		fmt.Printf("  Line: %d, Column: %d\n", err.Location.Line, err.Location.Column)
		fmt.Printf("  Message: %s\n", err.Text)

		if err.Location.LineText != "" {
			fmt.Printf("  Code:\n")
			fmt.Printf("    %s\n", err.Location.LineText)
			fmt.Printf("    %s^\n", strings.Repeat(" ", err.Location.Column-1))
		}

		if i < len(result)-1 {
			fmt.Println(strings.Repeat("-", 50))
		}
	}

	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("\nTotal Errors: %d\n", len(result))
}

func (handlerInstance *HandlerInstance) WatchForChanges() {
	handlerInstance.recompileSyncLock = new(sync.Once)
	var wg sync.WaitGroup
	defer wg.Done()

	watcher, _ := fsnotify.NewWatcher()
	defer watcher.Close()

	watcher.Add(handlerInstance.handlerConfig.Source)

	go func() {
		for event := range watcher.Events {
			if event.Op&fsnotify.Write == fsnotify.Write {
				handlerInstance.recompileSyncLock.Do(func() {
					handlerInstance.RecompileHandler()
					handlerInstance.recompileSyncLock = new(sync.Once)
				})
			}
		}
	}()

	wg.Add(1)
	wg.Wait()
}
