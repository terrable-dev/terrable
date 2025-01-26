package config

type TerrableConfig struct {
	Handlers                   []HandlerMapping
	GlobalEnvironmentVariables map[string]string
}

type HandlerMapping struct {
	Name                 string
	Source               string
	Http                 map[string]string
	Sqs                  map[string]interface{}
	EnvironmentVariables map[string]string
}
