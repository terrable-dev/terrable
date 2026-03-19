package config

type TerrableConfig struct {
	Handlers             []HandlerMapping
	EnvironmentVariables map[string]string
	HTTPAPI              *APIGatewayConfig
	RESTAPI              *APIGatewayConfig
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
	CORS *CORSConfig
}

type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

func (config TerrableConfig) EffectiveCORSConfig() *CORSConfig {
	if config.HTTPAPI != nil && config.HTTPAPI.CORS != nil {
		return config.HTTPAPI.CORS
	}

	if config.RESTAPI != nil && config.RESTAPI.CORS != nil {
		return config.RESTAPI.CORS
	}

	return nil
}
