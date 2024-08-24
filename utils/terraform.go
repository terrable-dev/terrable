package utils

import (
	"fmt"
	"path/filepath"
	"terrable/config"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

func ParseTerraformFile(filename string, targetModuleName string) (*config.TerrableConfig, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(filename)

	if diags.HasErrors() {
		return nil, diags
	}

	var terrableConfig config.TerrableConfig

	content, _ := file.Body.Content(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "module", LabelNames: []string{"name"}},
		},
	})

	for _, block := range content.Blocks {
		if block.Type == "module" && len(block.Labels) > 0 && block.Labels[0] == targetModuleName {
			moduleContent, _ := block.Body.Content(&hcl.BodySchema{
				Attributes: []hcl.AttributeSchema{
					{Name: "handlers", Required: false},
				},
			})

			if handlers, ok := moduleContent.Attributes["handlers"]; ok {
				handlersValue, _ := handlers.Expr.Value(nil)

				handlersValue.ForEachElement(func(key cty.Value, value cty.Value) bool {
					handlerName := key.AsString()
					handlerMap := value.AsValueMap()

					source := handlerMap["source"].AsString()

					var http config.HttpHandler

					if httpConfig, ok := handlerMap["http"]; ok && !httpConfig.IsNull() {
						httpConfigMap := httpConfig.AsValueMap()

						http = config.HttpHandler{
							Path:   httpConfigMap["path"].AsString(),
							Method: httpConfigMap["method"].AsString(),
						}
					}

					absoluteSourceFilePath, _ := getAbsoluteHandlerSourcePath(filename, source)

					terrableConfig.Handlers = append(terrableConfig.Handlers, config.HandlerMapping{
						Name:   handlerName,
						Source: absoluteSourceFilePath,
						Http:   http,
					})

					return false
				})
			}

			break
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
		return "", fmt.Errorf("Error converting relative path to absolute: %w", err)
	}

	return absolutePath, nil
}
