package offline

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/terrable-dev/terrable/config"
)

func TestBuildImplicitOptionsRoutes(t *testing.T) {
	terrableConfig := &config.TerrableConfig{
		HTTPAPI: &config.APIGatewayConfig{
			CORS: &config.CORSConfig{
				AllowMethods: []string{"get", "post", "options"},
			},
		},
		Handlers: []config.HandlerMapping{
			{
				Name: "ItemsHandler",
				Http: map[string]string{
					"GET":  "/items",
					"POST": "/items",
				},
			},
			{
				Name: "ExplicitOptionsHandler",
				Http: map[string]string{
					"GET":     "/health",
					"OPTIONS": "/health",
				},
			},
		},
	}

	routes := buildImplicitOptionsRoutes(terrableConfig)
	if len(routes) != 1 {
		t.Fatalf("expected exactly one implicit OPTIONS route, got %d", len(routes))
	}

	if routes[0].Path != "/items" {
		t.Fatalf("expected /items implicit OPTIONS route, got %s", routes[0].Path)
	}

	expectedMethods := []string{"GET", "OPTIONS", "POST"}
	if len(routes[0].AllowedMethods) != len(expectedMethods) {
		t.Fatalf("expected %d allowed methods, got %d", len(expectedMethods), len(routes[0].AllowedMethods))
	}

	for index, expectedMethod := range expectedMethods {
		if routes[0].AllowedMethods[index] != expectedMethod {
			t.Fatalf("expected allowed method %s at index %d, got %s", expectedMethod, index, routes[0].AllowedMethods[index])
		}
	}
}

func TestRegisterImplicitOptionsRoutes(t *testing.T) {
	terrableConfig := &config.TerrableConfig{
		HTTPAPI: &config.APIGatewayConfig{
			CORS: &config.CORSConfig{
				AllowOrigins:     []string{"https://app.example.com"},
				AllowMethods:     []string{"GET", "POST"},
				AllowHeaders:     []string{"content-type", "authorization"},
				AllowCredentials: true,
				MaxAge:           600,
			},
		},
		Handlers: []config.HandlerMapping{
			{
				Name: "ItemsHandler",
				Http: map[string]string{
					"GET": "/items",
				},
			},
		},
	}

	router := mux.NewRouter()
	registerCORSMiddleware(router, terrableConfig)
	registerImplicitOptionsRoutes(router, terrableConfig)

	request := httptest.NewRequest(http.MethodOptions, "/items", nil)
	request.Header.Set("Origin", "https://app.example.com")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204 response, got %d", recorder.Code)
	}

	if allowOrigin := recorder.Header().Get("Access-Control-Allow-Origin"); allowOrigin != "https://app.example.com" {
		t.Fatalf("expected Access-Control-Allow-Origin header to be echoed, got %q", allowOrigin)
	}

	if allowMethods := recorder.Header().Get("Access-Control-Allow-Methods"); allowMethods != "GET, POST" {
		t.Fatalf("expected configured allow methods, got %q", allowMethods)
	}

	if allowHeaders := recorder.Header().Get("Access-Control-Allow-Headers"); allowHeaders != "content-type, authorization" {
		t.Fatalf("expected configured allow headers, got %q", allowHeaders)
	}

	if allowCredentials := recorder.Header().Get("Access-Control-Allow-Credentials"); allowCredentials != "true" {
		t.Fatalf("expected Access-Control-Allow-Credentials=true, got %q", allowCredentials)
	}

	if maxAge := recorder.Header().Get("Access-Control-Max-Age"); maxAge != "600" {
		t.Fatalf("expected Access-Control-Max-Age=600, got %q", maxAge)
	}
}

func TestCORSMiddlewareAppliesHeadersToStandardResponses(t *testing.T) {
	terrableConfig := &config.TerrableConfig{
		RESTAPI: &config.APIGatewayConfig{
			CORS: &config.CORSConfig{
				AllowOrigins:     []string{"https://app.example.com"},
				ExposeHeaders:    []string{"x-request-id"},
				AllowCredentials: true,
			},
		},
	}

	router := mux.NewRouter()
	registerCORSMiddleware(router, terrableConfig)
	router.HandleFunc("/items", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods(http.MethodGet)

	request := httptest.NewRequest(http.MethodGet, "/items", nil)
	request.Header.Set("Origin", "https://app.example.com")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", recorder.Code)
	}

	if allowOrigin := recorder.Header().Get("Access-Control-Allow-Origin"); allowOrigin != "https://app.example.com" {
		t.Fatalf("expected Access-Control-Allow-Origin header to be echoed, got %q", allowOrigin)
	}

	if exposeHeaders := recorder.Header().Get("Access-Control-Expose-Headers"); exposeHeaders != "x-request-id" {
		t.Fatalf("expected Access-Control-Expose-Headers to be set, got %q", exposeHeaders)
	}

	if vary := recorder.Header().Get("Vary"); vary != "Origin" {
		t.Fatalf("expected Vary header to include Origin, got %q", vary)
	}
}
