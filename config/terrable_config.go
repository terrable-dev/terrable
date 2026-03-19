package config

type TerrableConfig struct {
	Handlers             []HandlerMapping
	EnvironmentVariables map[string]string
	HttpApi              *APIGatewayConfig
	RestApi              *APIGatewayConfig
	Timeout              int
}

type HandlerMapping struct {
	Name    string
	Source  string
	Http    map[string]string
	Sqs     map[string]interface{}
	Timeout int
}

type APIGatewayConfig struct {
	Cors *CorsConfig
}

type CorsConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

func (config TerrableConfig) EffectiveCorsConfig() *CorsConfig {
	if config.HttpApi != nil && config.HttpApi.Cors != nil {
		return config.HttpApi.Cors
	}

	if config.RestApi != nil && config.RestApi.Cors != nil {
		return config.RestApi.Cors
	}

	return nil
}
