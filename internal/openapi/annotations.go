package openapi

import (
	"regexp"
	"strconv"
	"strings"
)

// Annotation represents a parsed annotation from a comment.
type Annotation struct {
	Type   string   // The annotation type (e.g., "Summary", "Description", "Param")
	Name   string   // The name (for @Param, @Success, @Failure)
	Values []string // The values/arguments
	Raw    string   // The raw annotation text
}

// AnnotationParser parses OpenAPI annotations from comments.
type AnnotationParser struct {
	// Patterns for matching annotations
	patterns map[string]*regexp.Regexp
}

// NewAnnotationParser creates a new annotation parser.
func NewAnnotationParser() *AnnotationParser {
	return &AnnotationParser{
		patterns: map[string]*regexp.Regexp{
			"tag":     regexp.MustCompile(`@(?i)(Summary|Description|Tags|Accept|Produce|Router|Security|ID|Deprecated)\s+(.+)`),
			"param":   regexp.MustCompile(`@(?i)Param\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s*(.*)$`),
			"success": regexp.MustCompile(`@(?i)Success\s+(\d+)\s+(\{[^}]+\})?\s*(.*)$`),
			"failure": regexp.MustCompile(`@(?i)Failure\s+(\d+)\s+(\{[^}]+\})?\s*(.*)$`),
		},
	}
}

// ParseComments parses annotations from a slice of comment lines.
func (p *AnnotationParser) ParseComments(comments []string) []Annotation {
	var annotations []Annotation

	for _, comment := range comments {
		// Strip comment markers
		comment = strings.TrimSpace(comment)
		comment = strings.TrimPrefix(comment, "//")
		comment = strings.TrimPrefix(comment, "/*")
		comment = strings.TrimSuffix(comment, "*/")
		comment = strings.TrimSpace(comment)

		if annotation := p.parseLine(comment); annotation != nil {
			annotations = append(annotations, *annotation)
		}
	}

	return annotations
}

// ParseComment parses annotations from a single comment string.
func (p *AnnotationParser) ParseComment(comment string) []Annotation {
	lines := strings.Split(comment, "\n")
	return p.ParseComments(lines)
}

func (p *AnnotationParser) parseLine(line string) *Annotation {
	if !strings.HasPrefix(line, "@") {
		return nil
	}

	// Try each pattern
	if matches := p.patterns["tag"].FindStringSubmatch(line); matches != nil {
		return &Annotation{
			Type:   strings.ToLower(matches[1]),
			Values: []string{strings.TrimSpace(matches[2])},
			Raw:    line,
		}
	}

	if matches := p.patterns["param"].FindStringSubmatch(line); matches != nil {
		return &Annotation{
			Type:   "param",
			Name:   matches[1],
			Values: []string{matches[2], matches[3], matches[4], strings.TrimSpace(matches[5])},
			Raw:    line,
		}
	}

	if matches := p.patterns["success"].FindStringSubmatch(line); matches != nil {
		return &Annotation{
			Type:   "success",
			Name:   matches[1], // Status code
			Values: []string{matches[2], strings.TrimSpace(matches[3])},
			Raw:    line,
		}
	}

	if matches := p.patterns["failure"].FindStringSubmatch(line); matches != nil {
		return &Annotation{
			Type:   "failure",
			Name:   matches[1], // Status code
			Values: []string{matches[2], strings.TrimSpace(matches[3])},
			Raw:    line,
		}
	}

	return nil
}

// OperationMeta holds parsed metadata for an operation.
type OperationMeta struct {
	Summary     string
	Description string
	Tags        []string
	OperationID string
	Deprecated  bool
	Accept      []string
	Produce     []string
	Security    []string
	Router      *RouterMeta
	Params      []ParamMeta
	Responses   []ResponseMeta
}

// RouterMeta holds router metadata.
type RouterMeta struct {
	Path   string
	Method string
}

// ParamMeta holds parameter metadata.
type ParamMeta struct {
	Name        string
	In          string // path, query, header, body, formData
	Type        string
	Required    bool
	Description string
}

// ResponseMeta holds response metadata.
type ResponseMeta struct {
	StatusCode  int
	Type        string // success or failure
	Schema      string
	Description string
}

// ExtractOperationMeta extracts operation metadata from annotations.
func ExtractOperationMeta(annotations []Annotation) *OperationMeta {
	meta := &OperationMeta{}

	for _, ann := range annotations {
		switch ann.Type {
		case "summary":
			meta.Summary = ann.Values[0]
		case "description":
			meta.Description = ann.Values[0]
		case "tags":
			meta.Tags = strings.Split(ann.Values[0], ",")
			for i := range meta.Tags {
				meta.Tags[i] = strings.TrimSpace(meta.Tags[i])
			}
		case "id":
			meta.OperationID = ann.Values[0]
		case "deprecated":
			meta.Deprecated = true
		case "accept":
			meta.Accept = strings.Split(ann.Values[0], ",")
			for i := range meta.Accept {
				meta.Accept[i] = normalizeMediaType(strings.TrimSpace(meta.Accept[i]))
			}
		case "produce":
			meta.Produce = strings.Split(ann.Values[0], ",")
			for i := range meta.Produce {
				meta.Produce[i] = normalizeMediaType(strings.TrimSpace(meta.Produce[i]))
			}
		case "security":
			meta.Security = append(meta.Security, ann.Values[0])
		case "router":
			parts := strings.Fields(ann.Values[0])
			if len(parts) >= 2 {
				meta.Router = &RouterMeta{
					Path:   parts[0],
					Method: strings.ToUpper(strings.Trim(parts[1], "[]")),
				}
			}
		case "param":
			if len(ann.Values) >= 4 {
				param := ParamMeta{
					Name:        ann.Name,
					In:          ann.Values[0],
					Type:        ann.Values[1],
					Required:    ann.Values[2] == "true",
					Description: ann.Values[3],
				}
				meta.Params = append(meta.Params, param)
			}
		case "success", "failure":
			code, _ := strconv.Atoi(ann.Name)
			resp := ResponseMeta{
				StatusCode:  code,
				Type:        ann.Type,
				Schema:      ann.Values[0],
				Description: ann.Values[1],
			}
			meta.Responses = append(meta.Responses, resp)
		}
	}

	return meta
}

// normalizeMediaType converts shorthand media types to full form.
func normalizeMediaType(mt string) string {
	switch strings.ToLower(mt) {
	case "json":
		return "application/json"
	case "xml":
		return "application/xml"
	case "plain":
		return "text/plain"
	case "html":
		return "text/html"
	case "mpfd":
		return "multipart/form-data"
	case "x-www-form-urlencoded", "form":
		return "application/x-www-form-urlencoded"
	case "json-api":
		return "application/vnd.api+json"
	case "json-stream":
		return "application/x-json-stream"
	case "octet-stream":
		return "application/octet-stream"
	case "png":
		return "image/png"
	case "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	default:
		return mt
	}
}

// ToOperation converts OperationMeta to an OpenAPI Operation.
func (m *OperationMeta) ToOperation() *Operation {
	op := &Operation{
		Summary:     m.Summary,
		Description: m.Description,
		OperationID: m.OperationID,
		Deprecated:  m.Deprecated,
		Tags:        m.Tags,
		Responses:   make(map[string]Response),
	}

	// Add parameters
	for _, param := range m.Params {
		p := Parameter{
			Name:        param.Name,
			In:          param.In,
			Required:    param.Required,
			Description: param.Description,
			Schema:      SchemaFromType(param.Type),
		}
		op.Parameters = append(op.Parameters, p)
	}

	// Add responses
	for _, resp := range m.Responses {
		r := Response{
			Description: resp.Description,
		}
		if resp.Schema != "" {
			r.Content = map[string]MediaType{
				"application/json": {
					Schema: parseSchemaRef(resp.Schema),
				},
			}
		}
		op.Responses[strconv.Itoa(resp.StatusCode)] = r
	}

	// Add security
	if len(m.Security) > 0 {
		for _, sec := range m.Security {
			op.Security = append(op.Security, SecurityRequirement{sec: {}})
		}
	}

	return op
}

// parseSchemaRef parses a schema reference like {object} or {array} Model.
func parseSchemaRef(schema string) *Schema {
	schema = strings.TrimSpace(schema)
	if schema == "" {
		return nil
	}

	// Remove braces
	schema = strings.Trim(schema, "{}")
	parts := strings.Fields(schema)

	if len(parts) == 0 {
		return &Schema{Type: "object"}
	}

	typeName := parts[0]
	var modelName string
	if len(parts) > 1 {
		modelName = parts[1]
	}

	switch strings.ToLower(typeName) {
	case "object":
		if modelName != "" {
			return GenerateRef(modelName)
		}
		return &Schema{Type: "object"}
	case "array":
		if modelName != "" {
			return &Schema{
				Type:  "array",
				Items: GenerateRef(modelName),
			}
		}
		return &Schema{Type: "array"}
	case "string":
		return &Schema{Type: "string"}
	case "integer":
		return &Schema{Type: "integer"}
	case "number":
		return &Schema{Type: "number"}
	case "boolean":
		return &Schema{Type: "boolean"}
	default:
		// Assume it's a model reference
		return GenerateRef(typeName)
	}
}
