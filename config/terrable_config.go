package config

type TerrableConfig struct {
	Handlers []HandlerMapping
}

type HandlerMapping struct {
	Name   string
	Source string
	Http   map[string]string
}
