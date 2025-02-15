package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/terrable-dev/terrable/config"
	"github.com/zclconf/go-cty/cty"
)

func ParseTerraformFile(filename string, targetModuleName string) (*config.TerrableConfig, error) {
	content, err := ReadFile(filename)

	if err != nil {
		return nil, err
	}

	file, err := ParseHCL(content)

	if err != nil {
		return nil, err
	}

	targetModule, err := FindTargetModule(file, targetModuleName)

	if err != nil {
		return nil, err
	}

	return ParseModuleConfiguration(filename, targetModule)
}

func ParseHCL(content string) (*hcl.File, error) {
	parser := hclparse.NewParser()

	file, diags := parser.ParseHCL([]byte(content), "")

	if diags.HasErrors() {
		return nil, diags
	}

	return file, nil
}

func ReadFile(filename string) (string, error) {
	file, err := os.Open(filename)

	if err != nil {
		return "", fmt.Errorf("error opening file: %w", err)
	}

	defer file.Close()

	content, err := io.ReadAll(file)

	if err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	return string(content), nil
}

func FindTargetModule(file *hcl.File, targetModuleName string) (*hcl.Block, error) {
	content, _ := file.Body.Content(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "module", LabelNames: []string{"name"}},
		},
	})

	for _, block := range content.Blocks {
		if block.Type == "module" && len(block.Labels) > 0 && block.Labels[0] == targetModuleName {
			return block, nil
		}
	}

	return nil, fmt.Errorf("target module '%s' not found", targetModuleName)
}

const DefaultTimeout = 3

func ParseModuleConfiguration(filename string, moduleBlock *hcl.Block) (*config.TerrableConfig, error) {
	terrableConfig := config.TerrableConfig{
		Timeout: DefaultTimeout,
	}

	moduleContent, _ := moduleBlock.Body.Content(&hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "handlers", Required: false},
			{Name: "global_environment_variables", Required: false},
			{Name: "timeout", Required: false},
		},
	})

	// Extract global environment variables
	if globalEnvs, ok := moduleContent.Attributes["global_environment_variables"]; ok {
		globalEnvsValue, _ := globalEnvs.Expr.Value(nil)
		parsedGlobalEnvs, err := parseEnvironmentVariables(globalEnvsValue)

		if err != nil {
			return nil, fmt.Errorf("error parsing global environment variables: %w", err)
		}

		terrableConfig.GlobalEnvironmentVariables = parsedGlobalEnvs
	}

	// Extract global timeout
	if timeout, ok := moduleContent.Attributes["timeout"]; ok {
		timeoutValue, diags := timeout.Expr.Value(nil)
		if diags.HasErrors() {
			return nil, fmt.Errorf("error parsing global timeout: %s", diags.Error())
		}
		if timeoutValue.Type() == cty.Number {
			timeoutInt, _ := timeoutValue.AsBigFloat().Int64()
			terrableConfig.Timeout = int(timeoutInt)
		} else {
			return nil, fmt.Errorf("global timeout must be a number")
		}
	}

	if handlers, ok := moduleContent.Attributes["handlers"]; ok {
		handlersValue, _ := handlers.Expr.Value(nil)
		handlerMap := handlersValue.AsValueMap()

		for handlerName, handlerValue := range handlerMap {
			handlerConfig := handlerValue.AsValueMap()

			source := handlerConfig["source"].AsString()

			environmentVariables := make(map[string]string)
			if envVars, ok := handlerConfig["environment_variables"]; ok && !envVars.IsNull() {
				parsedEnvVars, err := parseEnvironmentVariables(envVars)
				if err != nil {
					return nil, fmt.Errorf("error parsing environment variables for handler %s: %w", handlerName, err)
				}
				environmentVariables = parsedEnvVars
			}

			http := make(map[string]string)
			if httpConfig, ok := handlerConfig["http"]; ok && !httpConfig.IsNull() {
				httpConfigMap := httpConfig.AsValueMap()
				for method, path := range httpConfigMap {
					http[method] = path.AsString()
				}
			}

			sqs := make(map[string]interface{})
			if sqsConfig, ok := handlerConfig["sqs"]; ok && !sqsConfig.IsNull() {
				sqsConfigMap := sqsConfig.AsValueMap()
				for key, value := range sqsConfigMap {
					sqs[key] = value
				}
			}

			// Use global timeout as default for handler
			timeout := terrableConfig.Timeout

			if handlerTimeout, ok := handlerConfig["timeout"]; ok && !handlerTimeout.IsNull() {
				if handlerTimeout.Type() == cty.Number {
					timeoutInt, _ := handlerTimeout.AsBigFloat().Int64()
					timeout = int(timeoutInt)
				} else {
					return nil, fmt.Errorf("handler timeout must be a number for handler %s", handlerName)
				}
			}

			absoluteSourceFilePath, err := getAbsoluteHandlerSourcePath(filename, source)
			if err != nil {
				return nil, fmt.Errorf("error getting absolute source path for handler %s: %w", handlerName, err)
			}

			terrableConfig.Handlers = append(terrableConfig.Handlers, config.HandlerMapping{
				Name:                 handlerName,
				Source:               absoluteSourceFilePath,
				Http:                 http,
				Sqs:                  sqs,
				EnvironmentVariables: environmentVariables,
				Timeout:              timeout,
			})
		}
	}

	return &terrableConfig, nil
}

func getAbsoluteHandlerSourcePath(basePath string, sourcePath string) (string, error) {
	if filepath.IsAbs(sourcePath) {
		return sourcePath, nil
	}

	dir := filepath.Dir(basePath)
	relativePath := filepath.Join(dir, sourcePath)

	absolutePath, err := filepath.Abs(relativePath)

	if err != nil {
		return "", fmt.Errorf("error converting relative path to absolute: %w", err)
	}

	return absolutePath, nil
}

func parseEnvironmentVariables(envVars cty.Value) (map[string]string, error) {
	parsedEnvVars := make(map[string]string)

	if envVars.IsNull() {
		return parsedEnvVars, nil
	}

	for k, v := range envVars.AsValueMap() {
		value := v.AsString()
		if strings.HasPrefix(value, "SSM:") {
			ssmValue, err := FetchSSMParameter(strings.TrimPrefix(value, "SSM:"))
			if err != nil {
				return nil, fmt.Errorf("error fetching SSM parameter for env var %s: %w", k, err)
			}
			parsedEnvVars[k] = ssmValue
		} else {
			parsedEnvVars[k] = value
		}
	}

	return parsedEnvVars, nil
}
