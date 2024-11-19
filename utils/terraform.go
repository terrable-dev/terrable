package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

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

func ParseModuleConfiguration(filename string, moduleBlock *hcl.Block) (*config.TerrableConfig, error) {
	var terrableConfig config.TerrableConfig

	moduleContent, _ := moduleBlock.Body.Content(&hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "handlers", Required: false},
			{Name: "global_environment_variables", Required: false},
		},
	})

	globalEnvVars := make(map[string]string)

	// Reconcile global environment variables
	if globalEnvs, ok := moduleContent.Attributes["global_environment_variables"]; ok {
		globalEnv, _ := globalEnvs.Expr.Value(nil)
		globalEnvVars = parseEnvironmentVariables(convertCtyToMap(globalEnv))
	}

	if handlers, ok := moduleContent.Attributes["handlers"]; ok {
		handlersValue, _ := handlers.Expr.Value(nil)

		handlersValue.ForEachElement(func(key cty.Value, value cty.Value) bool {
			handlerName := key.AsString()
			handlerMap := value.AsValueMap()

			handlerSpecificEnvVars := make(map[string]string)

			// Parse any environment variables specified for this handler
			if handlerEnvs, ok := handlerMap["environment_variables"]; ok {
				handlerSpecificEnvVars = parseEnvironmentVariables(convertCtyToMap(handlerEnvs))
			}

			handlerEnvVars := mergeMaps(globalEnvVars, handlerSpecificEnvVars)

			source := handlerMap["source"].AsString()

			http := make(map[string]string)

			if httpConfig, ok := handlerMap["http"]; ok && !httpConfig.IsNull() {
				httpConfigMap := httpConfig.AsValueMap()

				for method, path := range httpConfigMap {
					http[method] = path.AsString()
				}
			}

			absoluteSourceFilePath, _ := getAbsoluteHandlerSourcePath(filename, source)

			terrableConfig.Handlers = append(terrableConfig.Handlers, config.HandlerMapping{
				Name:                 handlerName,
				Source:               absoluteSourceFilePath,
				Http:                 http,
				EnvironmentVariables: handlerEnvVars,
			})

			return false
		})
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

func parseEnvironmentVariables(input map[string]interface{}) map[string]string {
	results := make(map[string]string)

	for environmentVariableKey, value := range input {
		envMap, ok := value.(map[string]interface{})

		if !ok {
			fmt.Printf("Unexpected value type for key %s\n", environmentVariableKey)
			continue
		}

		if ssm, ok := envMap["ssm"]; ok && ssm != nil {
			ssmParameterName, ok := ssm.(string)
			if !ok {
				fmt.Printf("SSM parameter for %s is not a string\n", environmentVariableKey)
				continue
			}

			ssmValue, err := GetSsmParameter(ssmParameterName)
			if err != nil {
				fmt.Printf("Error fetching SSM parameter %s: %v\n", ssmParameterName, err)
			} else {
				results[environmentVariableKey] = ssmValue
			}
		}

		if val, ok := envMap["value"]; ok && val != nil {
			if strVal, ok := val.(string); ok {
				results[environmentVariableKey] = strVal
			} else {
				fmt.Printf("Value for %s is not a string\n", environmentVariableKey)
			}
		}
	}

	return results
}

func convertCtyToMap(input cty.Value) map[string]interface{} {
	convertedMap := make(map[string]interface{})

	if input.Type().IsMapType() || input.Type().IsObjectType() {
		for it := input.ElementIterator(); it.Next(); {
			k, v := it.Element()
			key := k.AsString()
			value := make(map[string]interface{})

			for innerKey, innerVal := range v.AsValueMap() {
				if !innerVal.IsNull() {
					value[innerKey] = innerVal.AsString()
				}
			}

			convertedMap[key] = value
		}

		return convertedMap
	}

	return nil
}

func mergeMaps[K comparable, V any](maps ...map[K]V) map[K]V {
	result := make(map[K]V)

	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}

	return result
}
