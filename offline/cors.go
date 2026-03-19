package offline

import (
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/terrable-dev/terrable/config"
)

type implicitOptionsRoute struct {
	Path           string
	AllowedMethods []string
}

func registerCORSMiddleware(r *mux.Router, terrableConfig *config.TerrableConfig) {
	corsConfig := terrableConfig.EffectiveCorsConfig()
	if corsConfig == nil {
		return
	}

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			applyCORSResponseHeaders(w, r, corsConfig)
			next.ServeHTTP(w, r)
		})
	})
}

func registerImplicitOptionsRoutes(r *mux.Router, terrableConfig *config.TerrableConfig) {
	corsConfig := terrableConfig.EffectiveCorsConfig()
	if corsConfig == nil {
		return
	}

	for _, route := range buildImplicitOptionsRoutes(terrableConfig) {
		route := route
		r.HandleFunc(route.Path, func(w http.ResponseWriter, r *http.Request) {
			applyCORSResponseHeaders(w, r, corsConfig)
			if len(route.AllowedMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(route.AllowedMethods, ", "))
			}

			if len(corsConfig.AllowHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(corsConfig.AllowHeaders, ", "))
			} else if requestedHeaders := r.Header.Get("Access-Control-Request-Headers"); requestedHeaders != "" {
				w.Header().Set("Access-Control-Allow-Headers", requestedHeaders)
				addVaryHeader(w, "Access-Control-Request-Headers")
			}

			if corsConfig.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(corsConfig.MaxAge))
			}

			w.WriteHeader(http.StatusNoContent)
		}).Methods(http.MethodOptions)
	}
}

func buildImplicitOptionsRoutes(terrableConfig *config.TerrableConfig) []implicitOptionsRoute {
	corsConfig := terrableConfig.EffectiveCorsConfig()
	if corsConfig == nil {
		return nil
	}

	explicitOptionsPaths := make(map[string]struct{})
	pathMethods := make(map[string]map[string]struct{})

	for _, handler := range terrableConfig.Handlers {
		for method, path := range handler.Http {
			normalisedMethod := strings.ToUpper(method)

			if _, ok := pathMethods[path]; !ok {
				pathMethods[path] = make(map[string]struct{})
			}

			pathMethods[path][normalisedMethod] = struct{}{}

			if normalisedMethod == http.MethodOptions {
				explicitOptionsPaths[path] = struct{}{}
			}
		}
	}

	var routes []implicitOptionsRoute

	for path, methods := range pathMethods {
		if _, hasExplicitOptions := explicitOptionsPaths[path]; hasExplicitOptions {
			continue
		}

		routes = append(routes, implicitOptionsRoute{
			Path:           path,
			AllowedMethods: resolveAllowedMethods(methods, corsConfig.AllowMethods),
		})
	}

	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Path < routes[j].Path
	})

	return routes
}

func resolveAllowedMethods(routeMethods map[string]struct{}, configuredMethods []string) []string {
	if len(configuredMethods) > 0 {
		return uniqueSortedUppercase(configuredMethods)
	}

	allowedMethods := make([]string, 0, len(routeMethods)+1)

	for method := range routeMethods {
		allowedMethods = append(allowedMethods, method)
	}

	if _, hasOptions := routeMethods[http.MethodOptions]; !hasOptions {
		allowedMethods = append(allowedMethods, http.MethodOptions)
	}

	sort.Strings(allowedMethods)
	return allowedMethods
}

func applyCORSResponseHeaders(w http.ResponseWriter, r *http.Request, corsConfig *config.CorsConfig) {
	allowOrigin := resolveAllowOrigin(r.Header.Get("Origin"), corsConfig)
	if allowOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		if allowOrigin != "*" {
			addVaryHeader(w, "Origin")
		}
	}

	if corsConfig.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	if len(corsConfig.ExposeHeaders) > 0 && r.Method != http.MethodOptions {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(corsConfig.ExposeHeaders, ", "))
	}
}

func resolveAllowOrigin(requestOrigin string, corsConfig *config.CorsConfig) string {
	if len(corsConfig.AllowOrigins) == 0 {
		return ""
	}

	if slices.Contains(corsConfig.AllowOrigins, "*") && !corsConfig.AllowCredentials {
		return "*"
	}

	if requestOrigin != "" && slices.Contains(corsConfig.AllowOrigins, requestOrigin) {
		return requestOrigin
	}

	if len(corsConfig.AllowOrigins) == 1 {
		return corsConfig.AllowOrigins[0]
	}

	return ""
}

func uniqueSortedUppercase(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalised := make([]string, 0, len(values))

	for _, value := range values {
		upperValue := strings.ToUpper(value)
		if _, ok := seen[upperValue]; ok {
			continue
		}

		seen[upperValue] = struct{}{}
		normalised = append(normalised, upperValue)
	}

	sort.Strings(normalised)
	return normalised
}

func addVaryHeader(w http.ResponseWriter, value string) {
	existingValues := w.Header().Values("Vary")
	for _, existingValue := range existingValues {
		for _, vary := range strings.Split(existingValue, ",") {
			if strings.TrimSpace(vary) == value {
				return
			}
		}
	}

	w.Header().Add("Vary", value)
}
