package config

type TerrableConfig struct {
	Handlers []HandlerMapping
}

type HandlerMapping struct {
	Name   string
	Source string
	Http   HttpHandler
}

type HttpHandler struct {
	Path   string
	Method string
}
