// Package openapi provides OpenAPI 3.0 specification generation from CodeAI AST.
package openapi

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
)

// Handler provides HTTP endpoints for serving OpenAPI specifications.
type Handler struct {
	spec      *OpenAPI
	specJSON  []byte
	specYAML  []byte
	buildOnce sync.Once
}

// NewHandler creates a new OpenAPI HTTP handler.
func NewHandler(spec *OpenAPI) *Handler {
	return &Handler{
		spec: spec,
	}
}

// buildCachedSpec pre-builds the JSON and YAML representations for efficiency.
func (h *Handler) buildCachedSpec() {
	h.buildOnce.Do(func() {
		h.specJSON, _ = json.MarshalIndent(h.spec, "", "  ")
		h.specYAML, _ = yaml.Marshal(h.spec)
	})
}

// RegisterRoutes registers the OpenAPI HTTP routes on a chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/openapi.json", h.ServeJSON)
	r.Get("/openapi.yaml", h.ServeYAML)
	r.Get("/docs", h.ServeSwaggerUI)
	r.Get("/docs/", h.ServeSwaggerUI)
	r.Get("/redoc", h.ServeReDoc)
	r.Get("/redoc/", h.ServeReDoc)
}

// ServeJSON serves the OpenAPI specification as JSON.
func (h *Handler) ServeJSON(w http.ResponseWriter, r *http.Request) {
	h.buildCachedSpec()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write(h.specJSON)
}

// ServeYAML serves the OpenAPI specification as YAML.
func (h *Handler) ServeYAML(w http.ResponseWriter, r *http.Request) {
	h.buildCachedSpec()

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write(h.specYAML)
}

// ServeSwaggerUI serves a Swagger UI page for interactive API documentation.
func (h *Handler) ServeSwaggerUI(w http.ResponseWriter, r *http.Request) {
	html := generateSwaggerUIHTML("/openapi.json", h.spec.Info.Title)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// ServeReDoc serves a ReDoc page for API documentation.
func (h *Handler) ServeReDoc(w http.ResponseWriter, r *http.Request) {
	html := generateReDocHTML("/openapi.json", h.spec.Info.Title)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// generateSwaggerUIHTML generates an HTML page that loads Swagger UI.
func generateSwaggerUIHTML(specURL, title string) string {
	if title == "" {
		title = "API Documentation"
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + title + `</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin: 0; background: #fafafa; }
        .swagger-ui .topbar { display: none; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: "` + specURL + `",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                validatorUrl: null,
                displayRequestDuration: true,
                filter: true,
                showExtensions: true,
                showCommonExtensions: true
            });
            window.ui = ui;
        };
    </script>
</body>
</html>`
}

// generateReDocHTML generates an HTML page that loads ReDoc.
func generateReDocHTML(specURL, title string) string {
	if title == "" {
		title = "API Documentation"
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + title + `</title>
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <style>
        body { margin: 0; padding: 0; }
    </style>
</head>
<body>
    <redoc spec-url="` + specURL + `"
           expand-responses="200,201"
           hide-download-button="false"
           hide-hostname="false"
           native-scrollbars="true"
           theme='{
               "colors": {
                   "primary": {"main": "#32329f"}
               },
               "typography": {
                   "fontSize": "15px",
                   "fontFamily": "Roboto, sans-serif",
                   "headings": {
                       "fontFamily": "Montserrat, sans-serif"
                   }
               },
               "sidebar": {
                   "width": "260px"
               }
           }'>
    </redoc>
    <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>`
}

// Middleware creates a middleware that injects the OpenAPI spec into the context.
// This allows handlers to access the spec if needed.
func (h *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

// UpdateSpec updates the cached OpenAPI specification.
// This is useful when the spec changes at runtime.
func (h *Handler) UpdateSpec(spec *OpenAPI) {
	h.spec = spec
	h.buildOnce = sync.Once{} // Reset the cache
}

// GetSpec returns the current OpenAPI specification.
func (h *Handler) GetSpec() *OpenAPI {
	return h.spec
}
