package config

type TerrableConfig struct {
	Handlers             []HandlerMapping
	EnvironmentVariables map[string]string
	Timeout              int
}

type HandlerMapping struct {
	Name    string
	Source  string
	Http    map[string]string
	Sqs     map[string]interface{}
	Timeout int
}
