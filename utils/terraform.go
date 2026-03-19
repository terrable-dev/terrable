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
			{Name: "environment_variables", Required: false},
			{Name: "http_api", Required: false},
			{Name: "rest_api", Required: false},
			{Name: "timeout", Required: false},
		},
	})

	// Extract environment variables
	if environmentVariables, ok := moduleContent.Attributes["environment_variables"]; ok {
		envsValue, _ := environmentVariables.Expr.Value(nil)
		parsedEnvs, err := parseEnvironmentVariables(envsValue)

		if err != nil {
			return nil, fmt.Errorf("error parsing environment variables: %w", err)
		}

		terrableConfig.EnvironmentVariables = parsedEnvs
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

	if httpAPI, ok := moduleContent.Attributes["http_api"]; ok {
		httpAPIValue, diags := httpAPI.Expr.Value(nil)
		if diags.HasErrors() {
			return nil, fmt.Errorf("error parsing http_api configuration: %s", diags.Error())
		}

		parsedHTTPAPI, err := parseAPIGatewayConfig(httpAPIValue)
		if err != nil {
			return nil, fmt.Errorf("error parsing http_api configuration: %w", err)
		}

		terrableConfig.HTTPAPI = parsedHTTPAPI
	}

	if restAPI, ok := moduleContent.Attributes["rest_api"]; ok {
		restAPIValue, diags := restAPI.Expr.Value(nil)
		if diags.HasErrors() {
			return nil, fmt.Errorf("error parsing rest_api configuration: %s", diags.Error())
		}

		parsedRESTAPI, err := parseAPIGatewayConfig(restAPIValue)
		if err != nil {
			return nil, fmt.Errorf("error parsing rest_api configuration: %w", err)
		}

		terrableConfig.RESTAPI = parsedRESTAPI
	}

	if handlers, ok := moduleContent.Attributes["handlers"]; ok {
		handlersValue, _ := handlers.Expr.Value(nil)
		handlerMap := handlersValue.AsValueMap()

		for handlerName, handlerValue := range handlerMap {
			handlerConfig := handlerValue.AsValueMap()

			source := handlerConfig["source"].AsString()

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
				Name:    handlerName,
				Source:  absoluteSourceFilePath,
				Http:    http,
				Sqs:     sqs,
				Timeout: timeout,
			})
		}
	}

	return &terrableConfig, nil
}

func parseAPIGatewayConfig(apiConfig cty.Value) (*config.APIGatewayConfig, error) {
	if apiConfig.IsNull() {
		return nil, nil
	}

	parsedConfig := &config.APIGatewayConfig{}
	apiConfigMap := apiConfig.AsValueMap()

	corsConfig, ok := apiConfigMap["cors_configuration"]
	if !ok {
		corsConfig, ok = apiConfigMap["cors"]
	}

	if ok && !corsConfig.IsNull() {
		parsedCORSConfig, err := parseCORSConfig(corsConfig)
		if err != nil {
			return nil, err
		}

		parsedConfig.CORS = parsedCORSConfig
	}

	return parsedConfig, nil
}

func parseCORSConfig(corsConfig cty.Value) (*config.CORSConfig, error) {
	if corsConfig.IsNull() {
		return nil, nil
	}

	parsedConfig := &config.CORSConfig{}
	corsConfigMap := corsConfig.AsValueMap()

	if allowOrigins, ok := corsConfigMap["allow_origins"]; ok {
		parsedAllowOrigins, err := parseStringList(allowOrigins, "allow_origins")
		if err != nil {
			return nil, err
		}

		parsedConfig.AllowOrigins = parsedAllowOrigins
	}

	if allowMethods, ok := corsConfigMap["allow_methods"]; ok {
		parsedAllowMethods, err := parseStringList(allowMethods, "allow_methods")
		if err != nil {
			return nil, err
		}

		parsedConfig.AllowMethods = parsedAllowMethods
	}

	if allowHeaders, ok := corsConfigMap["allow_headers"]; ok {
		parsedAllowHeaders, err := parseStringList(allowHeaders, "allow_headers")
		if err != nil {
			return nil, err
		}

		parsedConfig.AllowHeaders = parsedAllowHeaders
	}

	if exposeHeaders, ok := corsConfigMap["expose_headers"]; ok {
		parsedExposeHeaders, err := parseStringList(exposeHeaders, "expose_headers")
		if err != nil {
			return nil, err
		}

		parsedConfig.ExposeHeaders = parsedExposeHeaders
	}

	if allowCredentials, ok := corsConfigMap["allow_credentials"]; ok {
		if allowCredentials.Type() != cty.Bool {
			return nil, fmt.Errorf("allow_credentials must be a boolean")
		}

		parsedConfig.AllowCredentials = allowCredentials.True()
	}

	if maxAge, ok := corsConfigMap["max_age"]; ok {
		if maxAge.Type() != cty.Number {
			return nil, fmt.Errorf("max_age must be a number")
		}

		parsedMaxAge, _ := maxAge.AsBigFloat().Int64()
		parsedConfig.MaxAge = int(parsedMaxAge)
	}

	return parsedConfig, nil
}

func parseStringList(value cty.Value, fieldName string) ([]string, error) {
	if value.IsNull() {
		return nil, nil
	}

	if !value.CanIterateElements() {
		return nil, fmt.Errorf("%s must be a list of strings", fieldName)
	}

	var values []string
	iterator := value.ElementIterator()

	for iterator.Next() {
		_, element := iterator.Element()
		if element.Type() != cty.String {
			return nil, fmt.Errorf("%s must be a list of strings", fieldName)
		}

		values = append(values, element.AsString())
	}

	return values, nil
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
