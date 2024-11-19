package config

type TerrableConfig struct {
	Handlers                   []HandlerMapping
	GlobalEnvironmentVariables map[string]EnvironmentVariable
}

type HandlerMapping struct {
	Name                 string
	Source               string
	Http                 map[string]string
	EnvironmentVariables map[string]string
}

type EnvironmentVariable struct {
	Value string
	SSM   string
}
