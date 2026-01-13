// Package openapi provides OpenAPI 3.0 specification generation from CodeAI AST.
package openapi

// OpenAPI represents a complete OpenAPI 3.0 specification.
type OpenAPI struct {
	OpenAPI    string                `json:"openapi" yaml:"openapi"`
	Info       Info                  `json:"info" yaml:"info"`
	Servers    []Server              `json:"servers,omitempty" yaml:"servers,omitempty"`
	Paths      map[string]PathItem   `json:"paths" yaml:"paths"`
	Components Components            `json:"components,omitempty" yaml:"components,omitempty"`
	Security   []SecurityRequirement `json:"security,omitempty" yaml:"security,omitempty"`
	Tags       []Tag                 `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// Info contains metadata about the API.
type Info struct {
	Title          string   `json:"title" yaml:"title"`
	Description    string   `json:"description,omitempty" yaml:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty" yaml:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty" yaml:"contact,omitempty"`
	License        *License `json:"license,omitempty" yaml:"license,omitempty"`
	Version        string   `json:"version" yaml:"version"`
}

// Contact provides contact information for the API.
type Contact struct {
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	URL   string `json:"url,omitempty" yaml:"url,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}

// License provides license information for the API.
type License struct {
	Name string `json:"name" yaml:"name"`
	URL  string `json:"url,omitempty" yaml:"url,omitempty"`
}

// Server represents an API server.
type Server struct {
	URL         string                    `json:"url" yaml:"url"`
	Description string                    `json:"description,omitempty" yaml:"description,omitempty"`
	Variables   map[string]ServerVariable `json:"variables,omitempty" yaml:"variables,omitempty"`
}

// ServerVariable represents a variable for server URL template substitution.
type ServerVariable struct {
	Enum        []string `json:"enum,omitempty" yaml:"enum,omitempty"`
	Default     string   `json:"default" yaml:"default"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
}

// PathItem describes the operations available on a single path.
type PathItem struct {
	Ref         string     `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Summary     string     `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string     `json:"description,omitempty" yaml:"description,omitempty"`
	Get         *Operation `json:"get,omitempty" yaml:"get,omitempty"`
	Put         *Operation `json:"put,omitempty" yaml:"put,omitempty"`
	Post        *Operation `json:"post,omitempty" yaml:"post,omitempty"`
	Delete      *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
	Options     *Operation `json:"options,omitempty" yaml:"options,omitempty"`
	Head        *Operation `json:"head,omitempty" yaml:"head,omitempty"`
	Patch       *Operation `json:"patch,omitempty" yaml:"patch,omitempty"`
	Trace       *Operation `json:"trace,omitempty" yaml:"trace,omitempty"`
	Parameters  []Parameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// Operation describes a single API operation on a path.
type Operation struct {
	Tags         []string              `json:"tags,omitempty" yaml:"tags,omitempty"`
	Summary      string                `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description  string                `json:"description,omitempty" yaml:"description,omitempty"`
	ExternalDocs *ExternalDocs         `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
	OperationID  string                `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters   []Parameter           `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody  *RequestBody          `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses    map[string]Response   `json:"responses" yaml:"responses"`
	Callbacks    map[string]Callback   `json:"callbacks,omitempty" yaml:"callbacks,omitempty"`
	Deprecated   bool                  `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Security     []SecurityRequirement `json:"security,omitempty" yaml:"security,omitempty"`
	Servers      []Server              `json:"servers,omitempty" yaml:"servers,omitempty"`

	// CodeAI extensions
	XCodeAIHandler    string   `json:"x-codeai-handler,omitempty" yaml:"x-codeai-handler,omitempty"`
	XCodeAIMiddleware []string `json:"x-codeai-middleware,omitempty" yaml:"x-codeai-middleware,omitempty"`
	XCodeAISource     string   `json:"x-codeai-source,omitempty" yaml:"x-codeai-source,omitempty"`
}

// ExternalDocs provides a link to external documentation.
type ExternalDocs struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	URL         string `json:"url" yaml:"url"`
}

// Parameter describes a single operation parameter.
type Parameter struct {
	Ref             string  `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Name            string  `json:"name,omitempty" yaml:"name,omitempty"`
	In              string  `json:"in,omitempty" yaml:"in,omitempty"` // query, header, path, cookie
	Description     string  `json:"description,omitempty" yaml:"description,omitempty"`
	Required        bool    `json:"required,omitempty" yaml:"required,omitempty"`
	Deprecated      bool    `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	AllowEmptyValue bool    `json:"allowEmptyValue,omitempty" yaml:"allowEmptyValue,omitempty"`
	Schema          *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example         any     `json:"example,omitempty" yaml:"example,omitempty"`
}

// RequestBody describes a request body.
type RequestBody struct {
	Ref         string               `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Description string               `json:"description,omitempty" yaml:"description,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
	Required    bool                 `json:"required,omitempty" yaml:"required,omitempty"`
}

// MediaType provides schema and examples for a media type.
type MediaType struct {
	Schema   *Schema            `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example  any                `json:"example,omitempty" yaml:"example,omitempty"`
	Examples map[string]Example `json:"examples,omitempty" yaml:"examples,omitempty"`
	Encoding map[string]Encoding `json:"encoding,omitempty" yaml:"encoding,omitempty"`
}

// Example represents an example value.
type Example struct {
	Ref           string `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Summary       string `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description   string `json:"description,omitempty" yaml:"description,omitempty"`
	Value         any    `json:"value,omitempty" yaml:"value,omitempty"`
	ExternalValue string `json:"externalValue,omitempty" yaml:"externalValue,omitempty"`
}

// Encoding provides encoding information for a property.
type Encoding struct {
	ContentType   string            `json:"contentType,omitempty" yaml:"contentType,omitempty"`
	Headers       map[string]Header `json:"headers,omitempty" yaml:"headers,omitempty"`
	Style         string            `json:"style,omitempty" yaml:"style,omitempty"`
	Explode       bool              `json:"explode,omitempty" yaml:"explode,omitempty"`
	AllowReserved bool              `json:"allowReserved,omitempty" yaml:"allowReserved,omitempty"`
}

// Header represents a header parameter.
type Header struct {
	Ref             string  `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Description     string  `json:"description,omitempty" yaml:"description,omitempty"`
	Required        bool    `json:"required,omitempty" yaml:"required,omitempty"`
	Deprecated      bool    `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	AllowEmptyValue bool    `json:"allowEmptyValue,omitempty" yaml:"allowEmptyValue,omitempty"`
	Schema          *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example         any     `json:"example,omitempty" yaml:"example,omitempty"`
}

// Response describes a single response from an API operation.
type Response struct {
	Ref         string               `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Description string               `json:"description" yaml:"description"`
	Headers     map[string]Header    `json:"headers,omitempty" yaml:"headers,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
	Links       map[string]Link      `json:"links,omitempty" yaml:"links,omitempty"`
}

// Link represents a possible design-time link for a response.
type Link struct {
	Ref          string         `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	OperationRef string         `json:"operationRef,omitempty" yaml:"operationRef,omitempty"`
	OperationID  string         `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters   map[string]any `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody  any            `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Description  string         `json:"description,omitempty" yaml:"description,omitempty"`
	Server       *Server        `json:"server,omitempty" yaml:"server,omitempty"`
}

// Callback represents a callback object.
type Callback map[string]PathItem

// Schema defines the structure of input and output data.
type Schema struct {
	Ref                  string             `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type                 string             `json:"type,omitempty" yaml:"type,omitempty"`
	Format               string             `json:"format,omitempty" yaml:"format,omitempty"`
	Title                string             `json:"title,omitempty" yaml:"title,omitempty"`
	Description          string             `json:"description,omitempty" yaml:"description,omitempty"`
	Default              any                `json:"default,omitempty" yaml:"default,omitempty"`
	Nullable             bool               `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	ReadOnly             bool               `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	WriteOnly            bool               `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`
	Deprecated           bool               `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Example              any                `json:"example,omitempty" yaml:"example,omitempty"`
	ExternalDocs         *ExternalDocs      `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`

	// Validation keywords
	MultipleOf       *float64 `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`
	Maximum          *float64 `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	Minimum          *float64 `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	MaxLength        *int     `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	MinLength        *int     `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	Pattern          string   `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	MaxItems         *int     `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	MinItems         *int     `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	UniqueItems      bool     `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`
	MaxProperties    *int     `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	MinProperties    *int     `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`

	// Object keywords
	Properties           map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	Required             []string           `json:"required,omitempty" yaml:"required,omitempty"`

	// Array keywords
	Items *Schema `json:"items,omitempty" yaml:"items,omitempty"`

	// Composition keywords
	AllOf         []*Schema `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	OneOf         []*Schema `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	AnyOf         []*Schema `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	Not           *Schema   `json:"not,omitempty" yaml:"not,omitempty"`
	Discriminator *Discriminator `json:"discriminator,omitempty" yaml:"discriminator,omitempty"`

	// Enumeration
	Enum []any `json:"enum,omitempty" yaml:"enum,omitempty"`

	// CodeAI MongoDB extensions
	XMongoCollection string `json:"x-mongo-collection,omitempty" yaml:"x-mongo-collection,omitempty"`
	XDatabaseType    string `json:"x-database-type,omitempty" yaml:"x-database-type,omitempty"`
}

// Discriminator can be used to aid in serialization, deserialization, and validation.
type Discriminator struct {
	PropertyName string            `json:"propertyName" yaml:"propertyName"`
	Mapping      map[string]string `json:"mapping,omitempty" yaml:"mapping,omitempty"`
}

// Components holds a set of reusable objects.
type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	Responses       map[string]*Response       `json:"responses,omitempty" yaml:"responses,omitempty"`
	Parameters      map[string]*Parameter      `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Examples        map[string]*Example        `json:"examples,omitempty" yaml:"examples,omitempty"`
	RequestBodies   map[string]*RequestBody    `json:"requestBodies,omitempty" yaml:"requestBodies,omitempty"`
	Headers         map[string]*Header         `json:"headers,omitempty" yaml:"headers,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
	Links           map[string]*Link           `json:"links,omitempty" yaml:"links,omitempty"`
	Callbacks       map[string]*Callback       `json:"callbacks,omitempty" yaml:"callbacks,omitempty"`
}

// SecurityScheme defines a security scheme that can be used by the operations.
type SecurityScheme struct {
	Ref              string     `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type             string     `json:"type,omitempty" yaml:"type,omitempty"` // apiKey, http, oauth2, openIdConnect
	Description      string     `json:"description,omitempty" yaml:"description,omitempty"`
	Name             string     `json:"name,omitempty" yaml:"name,omitempty"`             // for apiKey
	In               string     `json:"in,omitempty" yaml:"in,omitempty"`                 // for apiKey: query, header, cookie
	Scheme           string     `json:"scheme,omitempty" yaml:"scheme,omitempty"`         // for http
	BearerFormat     string     `json:"bearerFormat,omitempty" yaml:"bearerFormat,omitempty"` // for http bearer
	Flows            *OAuthFlows `json:"flows,omitempty" yaml:"flows,omitempty"`           // for oauth2
	OpenIDConnectURL string     `json:"openIdConnectUrl,omitempty" yaml:"openIdConnectUrl,omitempty"` // for openIdConnect
}

// OAuthFlows defines the configuration for OAuth flows.
type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty" yaml:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty" yaml:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty" yaml:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty" yaml:"authorizationCode,omitempty"`
}

// OAuthFlow defines the configuration details for a specific OAuth flow.
type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty" yaml:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty" yaml:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty" yaml:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes" yaml:"scopes"`
}

// SecurityRequirement lists the required security schemes to execute an operation.
type SecurityRequirement map[string][]string

// Tag adds metadata to a single tag.
type Tag struct {
	Name         string        `json:"name" yaml:"name"`
	Description  string        `json:"description,omitempty" yaml:"description,omitempty"`
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}
