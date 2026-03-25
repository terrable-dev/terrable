package offline

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"github.com/terrable-dev/terrable/config"
)

type HandlerInstance struct {
	handlerConfig         config.HandlerMapping
	handlerTranspiledPath string
	inputFilePaths        []string
	readCodeMutex         sync.RWMutex
	recompileSyncLock     *sync.Once
	envVars               map[string]string
}

func (handlerInstance *HandlerInstance) GetExecutionPath() string {
	return handlerInstance.handlerTranspiledPath
}

func (handlerInstance *HandlerInstance) SetExecutionPath(path string) {
	handlerInstance.readCodeMutex.Lock()
	defer handlerInstance.readCodeMutex.Unlock()

	handlerInstance.handlerTranspiledPath = path
}

func (handlerInstance *HandlerInstance) GetInputFiles() []string {
	handlerInstance.readCodeMutex.RLock()
	defer handlerInstance.readCodeMutex.RUnlock()

	return append([]string(nil), handlerInstance.inputFilePaths...)
}

func (handlerInstance *HandlerInstance) SetInputFiles(paths []string) {
	handlerInstance.readCodeMutex.Lock()
	defer handlerInstance.readCodeMutex.Unlock()

	handlerInstance.inputFilePaths = append([]string(nil), paths...)
}

func (handlerInstance *HandlerInstance) CompileHandler() (inputFilePaths []string, err error) {
	if err := validateHandlerSourcePath(handlerInstance.handlerConfig); err != nil {
		return nil, err
	}

	workingDirectory, err := os.Getwd()

	if err != nil {
		return nil, fmt.Errorf("error fetching executable location: %w", err)
	}

	result := api.Build(api.BuildOptions{
		EntryPoints: []string{handlerInstance.handlerConfig.Source},
		Bundle:      true,
		Write:       true,
		Format:      api.FormatCommonJS,
		Platform:    api.PlatformNode,
		Target:      api.ES2015,
		Sourcemap:   api.SourceMapLinked,
		Metafile:    true,
		GlobalName:  "exports",
		Outdir:      filepath.Join(workingDirectory, ".terrable", handlerInstance.handlerConfig.Name),
	})

	if len(result.Errors) > 0 {
		return nil, newHandlerCompileError(handlerInstance.handlerConfig, result.Errors)
	}

	handlerInstance.SetExecutionPath(filepath.ToSlash(compiledHandlerPath(workingDirectory, handlerInstance.handlerConfig)))
	inputFiles := extractMetafileInputs(result.Metafile)
	handlerInstance.SetInputFiles(inputFiles)
	return inputFiles, nil
}

func extractMetafileInputs(metafileContents string) []string {
	var data Metafile

	err := json.Unmarshal([]byte(metafileContents), &data)

	if err != nil {
		fmt.Println(fmt.Errorf("error parsing metafile: %w", err))
		return []string{}
	}

	var inputFiles []string

	for key := range data.Inputs {
		inputFiles = append(inputFiles, key)
	}

	return inputFiles
}

type Metafile struct {
	Inputs map[string]interface{} `json:"inputs"`
}

func validateHandlerSourcePath(handlerConfig config.HandlerMapping) error {
	fileInfo, err := os.Stat(handlerConfig.Source)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return newHandlerSourceError(handlerConfig, "no file exists at that path")
		}

		return newHandlerSourceError(handlerConfig, err.Error())
	}

	if fileInfo.IsDir() {
		return newHandlerSourceError(handlerConfig, "the path points to a directory, not a file")
	}

	return nil
}

func compiledHandlerPath(workingDirectory string, handlerConfig config.HandlerMapping) string {
	outputFileName := strings.TrimSuffix(filepath.Base(handlerConfig.Source), filepath.Ext(handlerConfig.Source)) + ".js"
	return filepath.Join(workingDirectory, ".terrable", handlerConfig.Name, outputFileName)
}

func newHandlerSourceError(handlerConfig config.HandlerMapping, problem string) error {
	lines := []string{
		fmt.Sprintf(`Handler %q could not be loaded.`, handlerConfig.Name),
		"",
	}

	lines = append(lines, formatHandlerLocationLines(handlerConfig)...)
	lines = append(lines,
		formatHandlerDetailLine("Problem", problem, false),
		"",
		`Check the handler's "source" setting and try again.`,
	)

	return errors.New(strings.Join(lines, "\n"))
}

func newHandlerCompileError(handlerConfig config.HandlerMapping, result []api.Message) error {
	lines := []string{
		fmt.Sprintf(`Handler %q could not be compiled.`, handlerConfig.Name),
		"",
	}

	lines = append(lines, formatHandlerLocationLines(handlerConfig)...)
	lines = append(lines, "", "  Build errors:")

	for _, buildErrorLine := range formatBuildErrorLines(result) {
		lines = append(lines, fmt.Sprintf("    - %s", formatBuildErrorLine(buildErrorLine)))
	}

	lines = append(lines,
		"",
		"Fix the handler code and try again.",
	)

	return errors.New(strings.Join(lines, "\n"))
}

func formatHandlerLocationLines(handlerConfig config.HandlerMapping) []string {
	var lines []string

	if handlerConfig.ConfiguredSource != "" {
		lines = append(lines, formatHandlerDetailLine("Configured source", handlerConfig.ConfiguredSource, true))
	}

	if handlerConfig.Source != "" {
		label := "Resolved path"
		if handlerConfig.ConfiguredSource == "" || handlerConfig.ConfiguredSource == handlerConfig.Source {
			label = "Path"
		}
		lines = append(lines, formatHandlerDetailLine(label, handlerConfig.Source, true))
	}

	return lines
}

func formatHandlerDetailLine(label string, value string, highlightValue bool) string {
	if highlightValue {
		value = formatHighlightedPath(value)
	}

	return fmt.Sprintf("  %-18s %s", label+":", value)
}

func formatHighlightedPath(value string) string {
	return color.New(color.FgYellow).Sprint(value)
}

func formatBuildErrorLine(value string) string {
	return color.New(color.FgYellow).Sprint(value)
}

func formatBuildErrorLines(result []api.Message) []string {
	var lines []string

	for _, buildErr := range result {
		var builder strings.Builder

		location := buildErr.Location
		if location.File != "" {
			builder.WriteString(location.File)

			if location.Line > 0 {
				builder.WriteString(fmt.Sprintf(":%d", location.Line))
				if location.Column > 0 {
					builder.WriteString(fmt.Sprintf(":%d", location.Column))
				}
			}

			builder.WriteString(": ")
		}

		builder.WriteString(buildErr.Text)
		lines = append(lines, builder.String())
	}

	return lines
}

func (handlerInstance *HandlerInstance) WatchForChanges(inputFiles []string) {
	handlerInstance.recompileSyncLock = new(sync.Once)
	var wg sync.WaitGroup
	defer wg.Done()

	watcher, _ := fsnotify.NewWatcher()
	defer watcher.Close()

	for _, file := range inputFiles {
		watcher.Add(file)
	}

	go func() {
		for event := range watcher.Events {
			if event.Op&fsnotify.Write == fsnotify.Write {
				handlerInstance.recompileSyncLock.Do(func() {
					if _, err := handlerInstance.CompileHandler(); err != nil {
						fmt.Println(err)
					}
					handlerInstance.recompileSyncLock = new(sync.Once)
				})
			}
		}
	}()

	wg.Add(1)
	wg.Wait()
}
